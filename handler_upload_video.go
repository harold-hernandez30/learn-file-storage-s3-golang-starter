package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("unable to run command with filePath %s. cmd: %s, error: %s", filePath, cmd.String(), err)
	}


	type stream struct {
		Width int `json:"width"`
		Height int `json:"height"`
	}
	type streams struct {
		Streams []stream `json:"streams"`
	}

	outputStream := streams{}
	err = json.Unmarshal(buffer.Bytes(), &outputStream)

	if err != nil {
		return "", fmt.Errorf("unable to unmarshal the stream: %s", err)
	}


	return GetVideoAspectRatio(outputStream.Streams[0].Width, outputStream.Streams[0].Height), nil

}

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	maxUploadLimit := 1 << 30
	http.MaxBytesReader(w, r.Body, int64(maxUploadLimit))

	videoID := r.PathValue("videoID")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Upload failed", errors.New("video not owned by user"))
		return
	}

	videoFile, videoFileHeader, err := r.FormFile("video")	
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get video file", err)
		return
	}

	defer videoFile.Close()

	mediaType, _, err := mime.ParseMediaType(videoFileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Unsupported media type", fmt.Errorf("provided media type: %s", mediaType))
		return
	}

	placeholderFilename := "tubely-upload-placeholder.mp4"
	tempFile, err := os.CreateTemp("", placeholderFilename)	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temporary placeholder", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()	

	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy to temp file", err)
		return
	}

	processedVideoFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to fast start the video", err)
		return
	}

	processedVideoFile, err := os.Open(processedVideoFilePath)	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open processed video file", err)
		return
	}

	_, err = processedVideoFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to offset to the beginning of the temp file", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(processedVideoFile.Name())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video aspect ratio", err)
		return
	}

	aspectRatioText := aspectRatioToText(aspectRatio)

	
	encoding := base64.RawURLEncoding
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	randBase64String := encoding.EncodeToString(randBytes)
	awsFullPath := fmt.Sprintf("%s/%s.mp4", aspectRatioText, randBase64String)
	params := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &awsFullPath,
		Body: processedVideoFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to add the object to the bucket", err)
		return
	}
	
	newURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, awsFullPath)
	video.VideoURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	signedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to sign video", err)
		return
	}


	videoInBytes, err := json.Marshal(&signedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to marshal video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoInBytes)

}


func aspectRatioToText(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "landscape"
	case "9:16":
		return "portrait"
	default:
		return "other"
	}
}
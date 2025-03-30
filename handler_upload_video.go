package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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
		respondWithError(w, http.StatusBadRequest, "Unsupported media type", fmt.Errorf("provided media type: %s\n", mediaType))
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload-placeholder.mp4")	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temporary placeholder", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to offset to the beginning of the temp file", err)
		return
	}

	encoding := base64.RawURLEncoding
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	randBase64String := encoding.EncodeToString(randBytes)
	params := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &randBase64String,
		Body: tempFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to add the object to the bucket", err)
		return
	}

	newURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s.mp4", cfg.s3Bucket, cfg.s3Region, randBase64String)
	video.VideoURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	videoInBytes, err := json.Marshal(&video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to marshal video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoInBytes)

}

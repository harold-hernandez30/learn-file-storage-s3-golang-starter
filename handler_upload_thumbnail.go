package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

var supportedImageTypes = []string {
	"image/jpeg",
	"image/png",
}

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse media type", err)
		return
	}

	isMediaTypeSupported := slices.Contains(supportedImageTypes, mediaType)

	if !isMediaTypeSupported {
		
		respondWithError(w, http.StatusBadRequest, "Unsupported media type", fmt.Errorf(mediaType))
		return
	}
	

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read request body", err)
		return
	}

	if video.UserID.String() != userID.String() {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of the video", fmt.Errorf("error"))
		return
	}

	encoding := base64.RawURLEncoding
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	randBase64String := encoding.EncodeToString(randBytes)

	fileExtension := strings.Split(mediaType, "/")[1]
	videoFilePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", randBase64String, fileExtension))
	destFile, err := os.Create(videoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create video file", err)
		return
	}

	result, err := io.Copy(destFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file", err)

		return
	}

	fmt.Printf("Bytes copied: %d\n", result)

	hostWithPort := fmt.Sprintf("%s:%s",cfg.host, cfg.port)
	fullPath := fmt.Sprintf("%s/%s", hostWithPort, videoFilePath)  
	video.ThumbnailURL = &fullPath

	updateVideoErr := cfg.db.UpdateVideo(video)
	if updateVideoErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", updateVideoErr)
		return
	}

	videoInBytes, err := json.Marshal(&video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to marshal video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoInBytes)
}

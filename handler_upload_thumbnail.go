package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	// TODO: implement the upload here
	
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read request body", err)
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


	encodedImageData := base64.StdEncoding.EncodeToString(imageData)
	dataUri := fmt.Sprintf("data:%s;base64,%s", mediaType, encodedImageData)
	thumbnailUrlPtr := &dataUri

	video.ThumbnailURL = thumbnailUrlPtr

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

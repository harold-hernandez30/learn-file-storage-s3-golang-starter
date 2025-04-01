package main

import (
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return video, nil
	}

	videoInfoSplice := strings.Split(*video.VideoURL, ",")
	if len(videoInfoSplice) == 1 {
		return video, nil
	}

	bucket := videoInfoSplice[0]
	key := videoInfoSplice[1]

	presignedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, 30 * time.Second)

	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedUrl
	return video, nil
}
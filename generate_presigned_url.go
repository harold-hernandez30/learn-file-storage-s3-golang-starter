package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expiryTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	getObjectInputParams := s3.GetObjectInput {
		Bucket: &bucket,
		Key: &key,

	}
	
	presignedHttpRequest, err := presignedClient.PresignGetObject(
		context.Background(), 
		&getObjectInputParams, 
		s3.WithPresignExpires(30 * time.Second),
	)

	if err != nil {
		return "", err
	}

	return presignedHttpRequest.URL, nil
}
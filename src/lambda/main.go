package main

import (
	"log"
	"os"

	"Hippocampus/src/lambda/handlers"
	"Hippocampus/src/lambda/storage"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	efsPath := os.Getenv("EFS_PATH")
	if efsPath == "" {
		efsPath = "/tmp/agents"
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-southeast-2"
	}

	storageManager, err := storage.NewManager(efsPath, s3Bucket, region)
	if err != nil {
		log.Fatalf("failed to initialize storage manager: %v", err)
	}

	handler := handlers.New(storageManager, nil)

	lambda.Start(handler.Route)
}

package minio

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// InitMinIOClient initializes and returns a MinIO client
func InitMinIOClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minIO client init error: %s", err)
	}

	fmt.Printf("✓ MinIO client initialized: %s\n", endpoint)
	return minioClient, nil
}

// EnsureBucketExists ensures a bucket exists, creates it if not
func EnsureBucketExists(ctx context.Context, client *minio.Client, bucketName string) error {
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("error creating bucket: %w", err)
		}
		log.Printf("✓ Created bucket: %s\n", bucketName)
	} else {
		log.Printf("✓ Bucket exists: %s\n", bucketName)
	}

	return nil
}

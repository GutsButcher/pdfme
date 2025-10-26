package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// EnsureBucket creates bucket if it doesn't exist
func EnsureBucket(bucketName string, client *minio.Client) error {
	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket existence: %s", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("error creating bucket: %s", err)
		}
		fmt.Printf("✓ Created bucket: %s\n", bucketName)
	} else {
		fmt.Printf("✓ Bucket exists: %s\n", bucketName)
	}

	return nil
}

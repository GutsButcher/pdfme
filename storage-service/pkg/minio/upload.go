package minio

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// UploadObject uploads bytes to MinIO bucket
func UploadObject(objBytes []byte, objName, bucketName string, client *minio.Client) error {
	uploadInfo, err := client.PutObject(
		context.Background(),
		bucketName,
		objName,
		bytes.NewReader(objBytes),
		int64(len(objBytes)),
		minio.PutObjectOptions{ContentType: "application/pdf"},
	)
	if err != nil {
		return fmt.Errorf("upload object error: %s", err)
	}

	fmt.Printf("✓ Successfully uploaded: %s/%s (size: %d bytes)\n", bucketName, objName, uploadInfo.Size)
	return nil
}

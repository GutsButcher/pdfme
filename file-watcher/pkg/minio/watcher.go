package minio

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// FileWatcher watches a MinIO bucket for new files
type FileWatcher struct {
	client         *minio.Client
	bucketName     string
	processedFiles map[string]bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(client *minio.Client, bucketName string) (*FileWatcher, error) {
	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("error checking bucket: %s", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("error creating bucket: %s", err)
		}
		log.Printf("✓ Created bucket: %s\n", bucketName)
	} else {
		log.Printf("✓ Bucket exists: %s\n", bucketName)
	}

	return &FileWatcher{
		client:         client,
		bucketName:     bucketName,
		processedFiles: make(map[string]bool),
	}, nil
}

// PollForNewFiles polls the bucket for new files
func (w *FileWatcher) PollForNewFiles(interval time.Duration, callback func(filename string, content []byte) error) error {
	log.Printf("[*] Starting to poll bucket '%s' every %v\n", w.bucketName, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial scan
	if err := w.scanBucket(callback); err != nil {
		log.Printf("[!] Error during initial scan: %s\n", err)
	}

	// Poll periodically
	for range ticker.C {
		if err := w.scanBucket(callback); err != nil {
			log.Printf("[!] Error during scan: %s\n", err)
		}
	}

	return nil
}

// scanBucket scans the bucket for new files
func (w *FileWatcher) scanBucket(callback func(filename string, content []byte) error) error {
	ctx := context.Background()

	// List objects in bucket
	objectCh := w.client.ListObjects(ctx, w.bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("error listing objects: %s", object.Err)
		}

		// Skip if already processed
		if w.processedFiles[object.Key] {
			continue
		}

		// Skip directories
		if strings.HasSuffix(object.Key, "/") {
			continue
		}

		log.Printf("\n[→] Found new file: %s (size: %d bytes)\n", object.Key, object.Size)

		// Download file content
		content, err := w.downloadFile(object.Key)
		if err != nil {
			log.Printf("[✗] Error downloading %s: %s\n", object.Key, err)
			continue
		}

		// Process file via callback
		if err := callback(object.Key, content); err != nil {
			log.Printf("[✗] Error processing %s: %s\n", object.Key, err)
			continue
		}

		// Mark as processed
		w.processedFiles[object.Key] = true
		log.Printf("[✓] Processed: %s\n", object.Key)

		// Optionally delete file after processing
		// Uncomment if you want to delete files after processing:
		// if err := w.deleteFile(object.Key); err != nil {
		//     log.Printf("[!] Error deleting %s: %s\n", object.Key, err)
		// }
	}

	return nil
}

// downloadFile downloads a file from MinIO
func (w *FileWatcher) downloadFile(filename string) ([]byte, error) {
	ctx := context.Background()

	object, err := w.client.GetObject(ctx, w.bucketName, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting object: %s", err)
	}
	defer object.Close()

	content, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("error reading object: %s", err)
	}

	return content, nil
}

// deleteFile deletes a file from MinIO
func (w *FileWatcher) deleteFile(filename string) error {
	ctx := context.Background()

	err := w.client.RemoveObject(ctx, w.bucketName, filename, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("error deleting object: %s", err)
	}

	log.Printf("  Deleted: %s\n", filename)
	return nil
}

// EncodeToBase64 encodes file content to base64
func EncodeToBase64(content []byte) string {
	return base64.StdEncoding.EncodeToString(content)
}

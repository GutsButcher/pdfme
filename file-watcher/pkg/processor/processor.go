package processor

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/pdfme/file-watcher/pkg/cache"
	"github.com/pdfme/file-watcher/pkg/database"
	"github.com/pdfme/file-watcher/pkg/rabbitmq"
	"github.com/pdfme/file-watcher/pkg/types"
)

// FileProcessor processes files from S3 with DB and Redis integration
type FileProcessor struct {
	minioClient *minio.Client
	db          *database.DB
	redis       *cache.RedisCache
	producer    *rabbitmq.Producer
	bucketName  string
	batchSize   int
	rateLimit   int
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(
	minioClient *minio.Client,
	db *database.DB,
	redis *cache.RedisCache,
	producer *rabbitmq.Producer,
	bucketName string,
	batchSize int,
	rateLimit int,
) *FileProcessor {
	return &FileProcessor{
		minioClient: minioClient,
		db:          db,
		redis:       redis,
		producer:    producer,
		bucketName:  bucketName,
		batchSize:   batchSize,
		rateLimit:   rateLimit,
	}
}

// ProcessFiles scans S3 and processes new files
func (p *FileProcessor) ProcessFiles(ctx context.Context) error {
	// List objects from S3 (metadata only, fast!)
	objects, err := p.listObjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	if len(objects) == 0 {
		log.Println("[i] No files found in bucket")
		return nil
	}

	log.Printf("[*] Found %d files in bucket\n", len(objects))

	// Process in batches with rate limiting
	processed := 0
	rateLimiter := time.NewTicker(time.Second / time.Duration(p.rateLimit))
	defer rateLimiter.Stop()

	for i := 0; i < len(objects); i += p.batchSize {
		end := i + p.batchSize
		if end > len(objects) {
			end = len(objects)
		}

		batch := objects[i:end]
		log.Printf("\n[*] Processing batch %d-%d of %d\n", i+1, end, len(objects))

		for _, obj := range batch {
			<-rateLimiter.C // Rate limit

			if err := p.processFile(ctx, obj); err != nil {
				log.Printf("[✗] Error processing %s: %v\n", obj.Key, err)
				continue
			}
			processed++
		}

		// Pause between batches
		if end < len(objects) {
			log.Println("[i] Pausing 1 second between batches...")
			time.Sleep(1 * time.Second)
		}
	}

	log.Printf("\n[✓] Batch complete: processed %d files\n", processed)
	return nil
}

// listObjects lists all objects in the bucket (metadata only)
func (p *FileProcessor) listObjects(ctx context.Context) ([]minio.ObjectInfo, error) {
	objectCh := p.minioClient.ListObjects(ctx, p.bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	var objects []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		// Skip directories
		if strings.HasSuffix(object.Key, "/") {
			continue
		}

		objects = append(objects, object)
	}

	return objects, nil
}

// processFile processes a single file
func (p *FileProcessor) processFile(ctx context.Context, obj minio.ObjectInfo) error {
	// Use S3 ETag as file hash (MD5)
	fileHash := strings.Trim(obj.ETag, "\"")
	filename := obj.Key

	log.Printf("\n[→] File: %s (hash: %s, size: %d bytes)\n", filename, fileHash[:12]+"...", obj.Size)

	// Step 1: Check Redis cache (fast path)
	status, err := p.redis.GetFileStatus(ctx, fileHash)
	if err != nil {
		log.Printf("    [!] Redis error (continuing): %v\n", err)
	} else if status == "completed" {
		log.Printf("    [↷] Skip: already completed (Redis cache)\n")
		return nil
	} else if status == "processing" {
		// Trust Redis TTL (1h) - if it exists, job is < 1h old
		// CheckStuckJobs() handles stuck job detection separately
		log.Printf("    [↷] Skip: still processing (Redis cache, < 1h)\n")
		return nil
	}

	// Step 2: Try to create job in DB (atomic with duplicate check)
	jobID, err := p.db.CreateJob(ctx, fileHash, filename)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	if jobID == "" {
		// Duplicate detected by DB, check its status
		job, err := p.db.GetJobByFileHash(ctx, fileHash)
		if err != nil {
			return fmt.Errorf("failed to get existing job: %w", err)
		}

		if job == nil {
			log.Printf("    [!] Unexpected: job not found after duplicate detection\n")
			return nil
		}

		// Check if we should retry
		if job.Status == "processing" && job.ProcessingStartedAt != nil {
			elapsed := time.Since(*job.ProcessingStartedAt)
			if elapsed > time.Hour && job.RetryCount < job.MaxRetries {
				log.Printf("    [!] Job stuck (%.0f min), marking for retry\n", elapsed.Minutes())
				success, err := p.db.MarkJobForRetry(ctx, job.ID)
				if err != nil {
					return fmt.Errorf("failed to mark for retry: %w", err)
				}
				if success {
					jobID = job.ID // Retry this job
				} else {
					log.Printf("    [✗] Max retries exceeded, marked as failed\n")
					return nil
				}
			} else {
				log.Printf("    [↷] Skip: already processing\n")
				return nil
			}
		} else if job.Status == "completed" {
			log.Printf("    [↷] Skip: already completed (DB)\n")
			// Update Redis cache
			p.redis.SetFileStatus(ctx, fileHash, "completed", 24*time.Hour)
			return nil
		} else {
			log.Printf("    [↷] Skip: duplicate job (status: %s)\n", job.Status)
			return nil
		}
	}

	log.Printf("    [+] Created job: %s\n", jobID[:8]+"...")

	// Step 3: Download file with timeout (only now, after confirming we need to process it!)
	log.Printf("    [↓] Downloading file...\n")

	// Create context with 5-minute timeout for download
	downloadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	content, err := p.downloadFile(downloadCtx, filename)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || downloadCtx.Err() == context.DeadlineExceeded {
			// Download timed out
			errMsg := "Download timeout after 5 minutes"
			log.Printf("    [✗] %s\n", errMsg)
			p.db.UpdateJobWithError(ctx, jobID, errMsg)
			// Mark for retry
			p.db.MarkJobForRetry(ctx, jobID)
			return fmt.Errorf("%s", errMsg)
		}
		p.db.UpdateJobWithError(ctx, jobID, fmt.Sprintf("Download failed: %v", err))
		return fmt.Errorf("failed to download file: %w", err)
	}

	fileSize := int64(len(content))
	log.Printf("    [✓] Downloaded %d bytes (%.2f MB)\n", fileSize, float64(fileSize)/1024/1024)

	// Step 4: Upload file to Redis as blob (temporary storage)
	log.Printf("    [↑] Uploading to Redis...\n")
	redisKey := fmt.Sprintf("blob:%s", fileHash)

	// Store with 1-hour TTL (parser should consume within this time)
	if err := p.redis.StoreFileBlob(ctx, fileHash, content, 1*time.Hour); err != nil {
		p.db.UpdateJobWithError(ctx, jobID, fmt.Sprintf("Redis upload failed: %v", err))
		return fmt.Errorf("failed to upload to Redis: %w", err)
	}
	log.Printf("    [✓] Uploaded to Redis (key: %s, TTL: 1h)\n", redisKey)

	// Step 5: Publish metadata to MQ (NOT file content!)
	log.Printf("    [→] Publishing metadata to MQ...\n")
	message := &types.FileMessage{
		JobID:    jobID,
		FileHash: fileHash,
		Filename: filename,
		RedisKey: redisKey,
		FileSize: fileSize,
	}

	if err := p.producer.PublishFile(message); err != nil {
		p.db.UpdateJobWithError(ctx, jobID, fmt.Sprintf("MQ publish failed: %v", err))
		// Cleanup Redis blob if MQ publish fails
		p.redis.DeleteFileBlob(ctx, fileHash)
		return fmt.Errorf("failed to publish to MQ: %w", err)
	}

	// Step 6: Update status to processing
	if err := p.db.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		log.Printf("    [!] Warning: failed to update status: %v\n", err)
	}

	// Step 7: Set Redis status cache
	if err := p.redis.SetFileStatus(ctx, fileHash, "processing", 1*time.Hour); err != nil {
		log.Printf("    [!] Warning: failed to set Redis status: %v\n", err)
	}

	log.Printf("    [✓] Queued for processing\n")
	return nil
}

// downloadFile downloads file content from S3
func (p *FileProcessor) downloadFile(ctx context.Context, filename string) ([]byte, error) {
	object, err := p.minioClient.GetObject(ctx, p.bucketName, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	content, err := io.ReadAll(object)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// CheckStuckJobs finds and retries stuck jobs (processing > 1 hour)
func (p *FileProcessor) CheckStuckJobs(ctx context.Context) error {
	stuckJobs, err := p.db.FindStuckJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to find stuck jobs: %w", err)
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	log.Printf("\n[!] Found %d stuck jobs in 'processing' status\n", len(stuckJobs))

	for _, job := range stuckJobs {
		elapsed := time.Since(*job.ProcessingStartedAt)
		log.Printf("  - Job %s: %s (stuck for %.0f min, retry %d/%d)\n",
			job.ID[:8]+"...", job.Filename, elapsed.Minutes(), job.RetryCount, job.MaxRetries)

		success, err := p.db.MarkJobForRetry(ctx, job.ID)
		if err != nil {
			log.Printf("    [✗] Failed to retry: %v\n", err)
			continue
		}

		if success {
			// Clear Redis cache so it gets reprocessed
			p.redis.DeleteFileStatus(ctx, job.FileHash)
			log.Printf("    [✓] Marked for retry\n")
		} else {
			log.Printf("    [✗] Max retries exceeded, marked as failed\n")
		}
	}

	return nil
}

// CheckStuckPendingJobs finds and retries jobs stuck in pending (> 10 minutes)
func (p *FileProcessor) CheckStuckPendingJobs(ctx context.Context) error {
	stuckJobs, err := p.db.FindStuckPendingJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to find stuck pending jobs: %w", err)
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	log.Printf("\n[!] Found %d stuck jobs in 'pending' status (likely pod crash during download)\n", len(stuckJobs))

	for _, job := range stuckJobs {
		elapsed := time.Since(job.CreatedAt)
		log.Printf("  - Job %s: %s (pending for %.0f min, retry %d/%d)\n",
			job.ID[:8]+"...", job.Filename, elapsed.Minutes(), job.RetryCount, job.MaxRetries)

		success, err := p.db.MarkJobForRetry(ctx, job.ID)
		if err != nil {
			log.Printf("    [✗] Failed to retry: %v\n", err)
			continue
		}

		if success {
			// Clear Redis cache so it gets reprocessed
			p.redis.DeleteFileStatus(ctx, job.FileHash)
			log.Printf("    [✓] Marked for retry (will be picked up in next scan)\n")
		} else {
			log.Printf("    [✗] Max retries exceeded, marked as failed\n")
		}
	}

	return nil
}

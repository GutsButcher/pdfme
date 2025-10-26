package types

// StorageMessage represents the message format for storage_ready queue
type StorageMessage struct {
	JobID       string `json:"job_id"`       // UUID from database
	FileHash    string `json:"file_hash"`    // For Redis cache update
	BucketName  string `json:"bucket_name"`
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"` // Base64 encoded PDF
}

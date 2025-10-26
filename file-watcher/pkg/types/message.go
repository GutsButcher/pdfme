package types

// FileMessage represents a file to be processed
type FileMessage struct {
	JobID       string `json:"job_id"`        // UUID from database
	FileHash    string `json:"file_hash"`     // S3 ETag (MD5)
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"`  // base64 encoded
}

package types

// FileMessage represents a file to be processed
// NOTE: File content is stored in Redis, not in this message
type FileMessage struct {
	JobID    string `json:"job_id"`     // UUID from database
	FileHash string `json:"file_hash"`  // S3 ETag (MD5), also Redis key
	Filename string `json:"filename"`
	RedisKey string `json:"redis_key"`  // Redis key where file content is stored (blob:{file_hash})
	FileSize int64  `json:"file_size"`  // File size in bytes
}

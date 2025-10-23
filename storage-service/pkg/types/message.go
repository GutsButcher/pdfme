package types

// StorageMessage represents the message format for storage_ready queue
type StorageMessage struct {
	BucketName  string `json:"bucket_name"`
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"` // Base64 encoded PDF
}

package types

// FileMessage represents a file to be parsed
type FileMessage struct {
	Filename    string `json:"filename"`
	FileContent string `json:"file_content"` // Base64 encoded file
	OrgID       string `json:"org_id"`       // Optional: extracted from filename or metadata
}

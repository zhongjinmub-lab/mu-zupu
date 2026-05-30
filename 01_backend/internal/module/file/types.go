package file

import (
	"path/filepath"
	"strings"
	"time"
)

type File struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Bucket    string    `json:"bucket"`
	ObjectKey string    `json:"object_key"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	Checksum  string    `json:"checksum"`
	PublicURL string    `json:"public_url,omitempty"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateFileInput struct {
	TenantID  string
	Bucket    string
	ObjectKey string
	Filename  string
	MimeType  string
	SizeBytes int64
	Checksum  string
	CreatedBy string
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "." || name == "" {
		return "file"
	}
	return name
}

func objectKey(tenantID, fileID, filename string) string {
	return "tenants/" + tenantID + "/files/" + fileID + "/" + sanitizeFilename(filename)
}

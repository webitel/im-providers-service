package model

type UploadRequest struct {
	DomainID   int64
	Name       string
	MimeType   string
	URL        string
	ExternalID string
}

type UploadResponse struct {
	ID                 string
	URL                string
	Size               int64
	ResponseStatusCode int32
	Malware            bool
}

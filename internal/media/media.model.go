package media

type UploadFileMetadata struct {
	ID                 int64
	URL                string
	Size               int64
	ResponseStatusCode int32
	Malware            bool
}

type UploadFileRequestMetadata struct {
	DomainID   int64
	MimeType   string
	Name       string
	UUID       string
	URL        string
	ExternalID string
}

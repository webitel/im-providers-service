package model

// https://core.telegram.org/bots/api#video
type Video struct {
	FileID         string         `json:"file_id"`
	FileUniqueID   string         `json:"file_unique_id"`
	Width          int64          `json:"width"`
	Height         int64          `json:"height"`
	Duration       int64          `json:"duration"`
	Thumbnail      *PhotoSize     `json:"thumbnail,omitempty"`
	Cover          []PhotoSize    `json:"cover,omitempty"`
	StartTimestamp *int64         `json:"start_timestamp,omitempty"`
	Qualities      []VideoQuality `json:"qualities,omitempty"`
	FileName       *string        `json:"file_name,omitempty"`
	MimeType       *string        `json:"mime_type,omitempty"`
	FileSize       *int64         `json:"file_size,omitempty"`
}

// https://core.telegram.org/bots/api#videoquality
type VideoQuality struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int64  `json:"width"`
	Height       int64  `json:"height"`
	Codec        string `json:"codec"`
	FileSize     *int64 `json:"file_size,omitempty"`
}

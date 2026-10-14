package viberbm

import (
	"mime"
	"path/filepath"
	"strings"
)

type mediaKind int

const (
	mediaFile mediaKind = iota
	mediaImage
	mediaVideo
	mediaAudio
)

// Go's builtin mime table lacks most audio/video types, so the extension
// fallback is explicit.
var extMediaKinds = map[string]mediaKind{
	".jpg": mediaImage, ".jpeg": mediaImage, ".png": mediaImage, ".gif": mediaImage, ".webp": mediaImage,
	".mp4": mediaVideo, ".mov": mediaVideo, ".m4v": mediaVideo, ".3gp": mediaVideo, ".webm": mediaVideo,
	".mkv": mediaVideo, ".avi": mediaVideo,
	".mp3": mediaAudio, ".m4a": mediaAudio, ".ogg": mediaAudio, ".oga": mediaAudio, ".opus": mediaAudio,
	".wav": mediaAudio, ".aac": mediaAudio, ".amr": mediaAudio, ".flac": mediaAudio,
}

// videoExtensions is the set Infobip VIDEO content accepts (max 200 MB).
var videoExtensions = map[string]struct{}{".mp4": {}, ".mov": {}, ".m4v": {}, ".3gp": {}}

var videoMimes = map[string]struct{}{"video/mp4": {}, "video/quicktime": {}, "video/x-m4v": {}, "video/3gpp": {}}

// classifyMedia trusts an explicit mime type over the file name; the extension
// decides only when the mime is missing or generic.
func classifyMedia(mimeType, fileName string) mediaKind {
	m := normalizeMime(mimeType)
	if m == "" || m == "application/octet-stream" {
		return extMediaKinds[strings.ToLower(filepath.Ext(fileName))]
	}

	switch {
	case strings.HasPrefix(m, "image/"):
		return mediaImage
	case strings.HasPrefix(m, "video/"):
		return mediaVideo
	case strings.HasPrefix(m, "audio/"):
		return mediaAudio
	default:
		return mediaFile
	}
}

func isSupportedVideo(mimeType, fileName string) bool {
	if m := normalizeMime(mimeType); m != "" && m != "application/octet-stream" {
		_, ok := videoMimes[m]

		return ok
	}

	_, ok := videoExtensions[strings.ToLower(filepath.Ext(fileName))]

	return ok
}

func normalizeMime(v string) string {
	if v == "" {
		return ""
	}

	parsed, _, err := mime.ParseMediaType(v)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(v))
	}

	return parsed
}

package model

import (
	"fmt"
	"runtime"
)

// Server Identity constants

const (
	ServerVersion = "v1"
)

// Service identity constants
const (
	ServiceName      = "im-providers-service"
	ServiceNamespace = "webitel"
)

// Build-time variables.
// These are placeholders that GoReleaser will overwrite using -ldflags.
var (
	Version        = "0.0.0"
	Commit         = "hash"
	CommitDate     = "unknown"
	Branch         = "branch"
	BuildTimestamp = "none"
	BuiltBy        = "manual"
)

// BuildInfo holds all service metadata.
type BuildInfo struct {
	Service        string `json:"service"`
	Namespace      string `json:"namespace"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	CommitDate     string `json:"commit_date"`
	Branch         string `json:"branch"`
	BuildTimestamp string `json:"build_timestamp"`
	GoVersion      string `json:"go_version"`
	Platform       string `json:"platform"`
}

// GetBuildInfo returns a populated BuildInfo struct.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Service:        ServiceName,
		Namespace:      ServiceNamespace,
		Version:        Version,
		Commit:         Commit,
		CommitDate:     CommitDate,
		Branch:         Branch,
		BuildTimestamp: BuildTimestamp,
		GoVersion:      runtime.Version(),
		Platform:       fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

package client

import (
	"regexp"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var apiVersionRegex = regexp.MustCompile(`^v\d+\.\d+$`)

type RequestClientConfig struct {
	apiVersion  string
	baseURL     string
	accessToken string
}

func (config *RequestClientConfig) Validate() error {
	if config == nil {
		return errors.InvalidArgument("client config is required", errors.WithID("client.options.validate"))
	}

	if config.accessToken == "" || strings.Trim(config.accessToken, " ") == "" {
		return errors.InvalidArgument("access token is required", errors.WithID("client.options.validate"))
	}

	if config.apiVersion == "" || strings.Trim(config.apiVersion, " ") == "" {
		return errors.InvalidArgument("api version is required", errors.WithID("client.options.validate"))
	}

	if !apiVersionRegex.MatchString(config.apiVersion) {
		return errors.InvalidArgument("invalid format for graph api", errors.WithID("client.options.validate"), errors.WithValue("version", config.apiVersion))
	}

	if !strings.Contains(config.baseURL, BaseURL) {
		return errors.InvalidArgument(`base URL must contain "graph.facebook.com"`, errors.WithID("client.options.validate"), errors.WithValue("base-url", config.baseURL))
	}

	if strings.HasPrefix(config.baseURL, "https://") {
		return errors.InvalidArgument(`base URL must not include protocol as "http://" adds by default`, errors.WithID("client.options.validate"), errors.WithValue("base-url", config.baseURL))
	}

	return nil
}

func getDefaultClientConfig() RequestClientConfig {
	return RequestClientConfig{
		apiVersion:  APIVersion,
		baseURL:     BaseURL,
		accessToken: "",
	}
}

func WithApiVersionConfig(version string) func(cfg *RequestClientConfig) {
	return func(cfg *RequestClientConfig) {
		cfg.apiVersion = version
	}
}

func WithBaseURLConfig(base string) func(cfg *RequestClientConfig) {
	return func(cfg *RequestClientConfig) {
		cfg.baseURL = base
	}
}

func WithAccessTokenConfig(token string) func(cfg *RequestClientConfig) {
	return func(cfg *RequestClientConfig) {
		cfg.accessToken = token
	}
}

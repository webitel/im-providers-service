package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

const (
	APIVersion      string = "v25.0"
	BaseURL         string = "graph.facebook.com"
	RequestProtocol string = "https"
)

type WhatsAppApiType string

const (
	WhatsAppApiTypeBusiness WhatsAppApiType = "business"
)

type RequestClient struct {
	apiVersion  string
	baseUrl     string
	accessToken string
}

func (client *RequestClient) BaseURL() string     { return client.baseUrl }
func (client *RequestClient) ApiVersion() string  { return client.apiVersion }
func (client *RequestClient) AccessToken() string { return client.accessToken }

func NewRequesClient(options ...func(cfg *RequestClientConfig)) (*RequestClient, error) {
	cfg := getDefaultClientConfig()
	cfgPtr := &cfg

	for _, option := range options {
		option(cfgPtr)
	}

	if err := cfgPtr.Validate(); err != nil {
		return nil, err
	}

	client := RequestClient{
		apiVersion:  cfg.apiVersion,
		baseUrl:     cfg.baseURL,
		accessToken: cfg.accessToken,
	}

	return &client, nil
}

type RequestCloudApiParams struct {
	Body       string
	Path       string
	Method     string
	QueryParam map[string]string
}

func (client *RequestClient) requestWithContext(ctx context.Context, params RequestCloudApiParams) (string, error) {
	queryParamsString := ""
	if len(params.QueryParam) > 0 {
		queryParamsString = "?"
		for k, v := range params.QueryParam {
			if queryParamsString != "?" {
				queryParamsString += "&"
				queryParamsString += strings.Join([]string{
					queryParamsString, k, "=", v,
				}, "")
			} else {
				queryParamsString += strings.Join([]string{k, "=", v}, "")
			}
		}
	}

	requestPath := strings.Join(
		[]string{RequestProtocol, "://", client.baseUrl, "/", client.apiVersion, "/", params.Path, queryParamsString},
		"",
	)

	httpRequest, err := http.NewRequestWithContext(ctx, params.Method, requestPath, strings.NewReader(params.Body))
	if err != nil {
		return "", errors.Internal("creating new http request", errors.WithCause(err), errors.WithID("client.request.client.request"))
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set(
		"Authorization",
		fmt.Sprintf("Bearer %s", client.accessToken),
	)

	httpClient := http.DefaultClient
	response, err := httpClient.Do(httpRequest)
	if err != nil || response.StatusCode != http.StatusOK {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.Internal("reading response body", errors.WithCause(err), errors.WithID("client.request.client.request"))
	}

	return string(body), nil
}

func (client *RequestClient) request(params RequestCloudApiParams) (string, error) {
	return client.requestWithContext(context.Background(), params)
}

type ApiRequest struct {
	Path        string
	Method      string
	Body        string
	Fields      []ApiRequestParamField
	QueryParams map[string]string
	Requester   *RequestClient
}

type ApiRequestParamField struct {
	Name    string
	Filters map[string]string
}

func (field *ApiRequestParamField) AddFilter(key, value string) {
	field.Filters[key] = value
}

func (client *RequestClient) NewApiRequest(path, method string) *ApiRequest {
	return &ApiRequest{
		Path:        path,
		Method:      method,
		Fields:      []ApiRequestParamField{},
		QueryParams: map[string]string{},
		Requester:   client,
	}
}

func (request *ApiRequest) AddField(field ApiRequestParamField) *ApiRequestParamField {
	request.Fields = append(request.Fields, field)
	return &field
}

func (request *ApiRequest) AddQueryParam(key, value string) {
	request.QueryParams[key] = value
}

func (request *ApiRequest) SetMethod(method string) {
	request.Method = method
}

func (request *ApiRequest) SetBody(body string) {
	request.Body = body
}

func (request *ApiRequest) ExecuteWithContext(ctx context.Context) (string, error) {
	queryParam := map[string]string{}
	if len(request.Fields) > 0 {
		fieldString := ""
		for _, field := range request.Fields {
			newFieldString := ""
			if fieldString != "" {
				newFieldString = ","
			}
			filterString := ""
			for k, v := range field.Filters {
				filterString += strings.Join([]string{".", k, "(" + v + ")"}, "")
			}
			newFieldString += strings.Join([]string{field.Name, filterString}, "")
			fieldString += newFieldString
		}

		queryParam["fields"] = fieldString
	}

	if len(request.QueryParams) > 0 {
		for k, v := range request.QueryParams {
			queryParam[k] = v
		}
	}

	response, err := request.Requester.requestWithContext(ctx, RequestCloudApiParams{
		Body:       request.Body,
		Path:       request.Path,
		Method:     request.Method,
		QueryParam: queryParam,
	})

	if err != nil {
		return "", err
	}

	return response, nil
}

func (request *ApiRequest) Execute() (string, error) {
	return request.ExecuteWithContext(context.Background())
}

func (client *RequestClient) RequestMultipart(method, path, contentType string, body io.Reader) (string, error) {
	return client.RequestMultipartWithContext(context.Background(), method, path, contentType, body)
}

func (client *RequestClient) RequestMultipartWithContext(ctx context.Context, method, path, contentType string, body io.Reader) (string, error) {
	requestPath := strings.Join([]string{
		RequestProtocol, "://", client.baseUrl, "/", client.apiVersion, "/", path,
	}, "")

	httpRequest, err := http.NewRequestWithContext(ctx, method, requestPath, body)
	if err != nil {
		return "", errors.Internal("creating http request", errors.WithID("client.request.client.request_multipart"), errors.WithCause(err))
	}

	httpClient := http.DefaultClient
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.Internal("reading response body", errors.WithID("client.request.client.request_multipart"), errors.WithCause(err))
	}
	return string(respBody), nil
}

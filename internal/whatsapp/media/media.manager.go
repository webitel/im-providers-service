package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"

	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type MediaMetadata struct {
	MessageProduct string `json:"message_product"`
	URL            string `json:"url"`
	MimeType       string `json:"mime_type"`
	Sha256         string `json:"sha_256"`
	FileSize       int    `json:"file_size"`
	ID             string `json:"id"`
}

type MediaManager struct {
	client client.RequestClient
}

func NewMediaManager(client client.RequestClient) *MediaManager {
	return &MediaManager{client: client}
}

// https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/#get-media-url
// Use the Media API to get a media URL by querying the media ID directly. You can then use the URL with your access token to download the media asset.
// A successful response includes an object with a media url. The URL is only valid for 5 minutes.
func (mediaManager *MediaManager) GetMediaURLByID(ctx context.Context, id string) (string, error) {
	apiRequest := mediaManager.client.NewApiRequest(id, http.MethodGet)

	raw, err := apiRequest.ExecuteWithContext(ctx)
	if err != nil {
		return "", errors.New("executing get media url by id request", errors.WithCause(err), errors.WithID("whatsapp.media.manager.get_media_url_by_id"))
	}

	var res MediaMetadata
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return "", errors.Internal("unmarshaling get media url by id response", errors.WithCause(err), errors.WithID("whatsapp.media.manager.get_media_url_by_id"))
	}

	if res.URL == "" {
		return "", errors.NotFound("no media url found in response", errors.WithCause(err), errors.WithID("whatsapp.media.manager.get_media_url_by_id"), errors.WithValue("id", id))
	}

	return res.URL, nil
}

type DeleteSuccessResponse struct {
	Success bool `json:"success"`
}

// https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/#delete-media
// Use the Media API to delete a media asset.
func (mediaManager *MediaManager) DeleteMedia(ctx context.Context, id string) (bool, error) {
	apiRequest := mediaManager.client.NewApiRequest(
		strings.Join([]string{"media", id}, "/"),
		http.MethodDelete,
	)

	raw, err := apiRequest.ExecuteWithContext(ctx) //TODO: make retriable
	if err != nil {
		return false, errors.New("executing delete media request", errors.WithCause(err), errors.WithID("whatsapp.media.manager.delete_media"), errors.WithValue("id", id))
	}

	var deleteSuccessResponse DeleteSuccessResponse
	if err := json.Unmarshal([]byte(raw), &deleteSuccessResponse); err != nil {
		return false, errors.Internal("unmarshaling raw to delete success response", errors.WithCause(err), errors.WithID("whatsapp.media.manager.delete_media"), errors.WithValue("id", id))
	}

	if !deleteSuccessResponse.Success { //TODO: make retriable with exponential backoff?
		return false, errors.New("media deletion", errors.WithID("whatsapp.media.manager.delete_media"))
	}

	return deleteSuccessResponse.Success, nil
}

// https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/#upload-media
// Use the Media Upload API to upload media.
func (mediaManager *MediaManager) UploadMedia(ctx context.Context, file io.Reader, phoneNumberID, filename, mimeType string) (string, error) {
	readPipe, writePipe := io.Pipe()
	writer := multipart.NewWriter(writePipe)

	go func() { //TODO: add context propagation and better error handling in goroutine
		var err error
		defer func() {
			writer.Close()
			writePipe.CloseWithError(err)
		}()

		if err = writer.WriteField("messaging_product", "whatsapp"); err != nil {
			return
		}

		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%s`, filepath.Base(filename)))
		partHeader.Set("Content-Type", mimeType)

		filePart, err := writer.CreatePart(partHeader)
		if err != nil {
			return
		}

		_, err = io.Copy(filePart, file)
	}()

	apiPath := strings.Join([]string{phoneNumberID, "media"}, "/")
	contentType := writer.FormDataContentType()

	responseBody, err := mediaManager.client.RequestMultipartWithContext(ctx, http.MethodPost, apiPath, contentType, readPipe)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal([]byte(responseBody), &result); err != nil {
		return "", errors.Internal("unmarshaling upload media result", errors.WithCause(err), errors.WithID("whatsapp.media.manager.upload_media"))
	}

	if result.ID == "" {
		return "", errors.New("no media id in response", errors.WithID("whatsapp.media.manager.upload_media"), errors.WithValue("response", responseBody))
	}

	return result.ID, nil
}

// https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/#download-media
// Upon success, the API will respond with the binary data of the media asset.
// Response headers contain a content-type header to indicate the mime type of returned data.
// Check supported media types for supported media types.
// If the download attempt fails, you will receive a 404 Not Found response code.
// In that case, try to get a new media URL and download it again.
// If doing so doesn’t resolve the issue, renew your access token and attempt to download the media asset again.
func (mediaManager *MediaManager) DownloadMedia(ctx context.Context, id string) (io.ReadCloser, string, error) {
	url, err := mediaManager.GetMediaURLByID(ctx, id)
	if err != nil {
		return nil, "", errors.Wrap(err, errors.WithID("whatsapp.media.manager.download_media"))
	}

	mediaBody, mimeType, err := mediaManager.DownloadMediaByURL(ctx, url)
	if err != nil {
		return nil, "", errors.Wrap(err, errors.WithID("whatsapp.media.manager.download_media"))
	}

	return mediaBody, mimeType, nil
}

func (mediaManager *MediaManager) DownloadMediaByURL(ctx context.Context, url string) (io.ReadCloser, string, error) {
	//TODO: add response body io limit?
	// -add correct request url build
	if url == "" {
		return nil, "", errors.InvalidArgument("url is required", errors.WithID("whatsapp.media.manager.download_media_by_url"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", errors.Internal("creating outgoing download media request", errors.WithCause(err), errors.WithID("whatsapp.media.manager.download_media_by_url"))
	}
	req.Header.Set("Authorization", "Bearer "+mediaManager.client.AccessToken())

	response, err := http.DefaultClient.Do(req)
	if err != nil { //TODO: add 404 response check
		return nil, "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, "", errors.NotFound("bad media download response status", errors.WithCause(err), errors.WithID("whatsapp.media.manager.download_media_by_url"))
	}

	return response.Body, response.Header.Get("Content-Type"), nil
}

package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// GraphBaseURL is the versioned Graph API endpoint.
// https://developers.facebook.com/docs/graph-api/overview#versions
const GraphBaseURL = "https://graph.facebook.com/v25.0"

// Media type constants for the Send API attachment type field.
// https://developers.facebook.com/docs/messenger-platform/reference/send-api#attachment
const (
	MediaImage = "image"
	MediaFile  = "file"
)

// graphAPI is the contract used by facebookProvider to talk to the Graph API.
// Keeping it as an interface allows the provider to be tested without network calls.
type graphAPI interface {
	GetUserProfile(ctx context.Context, psid, token string) (*UserProfile, error)
	ParseWebhook(data []byte) (*WebhookRequest, error)
	SendText(ctx context.Context, token, psid, text string) (*sharedmodel.MessageResponse, error)
	SendMedia(ctx context.Context, token, psid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error)
}

// ErrTokenInvalid is returned when Facebook rejects the page token (OAuth error code 190).
// The gate must be re-authorized via StartMetaOAuth → MetaOAuthCallback → UpdateFacebookGate.
//
// https://developers.facebook.com/docs/graph-api/guides/error-handling#errorcodes
var ErrTokenInvalid = errors.New("facebook: page token invalid or revoked")

type apiClient struct {
	client *http.Client
	logger *slog.Logger
	apiURL string
}

var _ graphAPI = (*apiClient)(nil)

func newAPIClient(l *slog.Logger) *apiClient {
	return &apiClient{
		client: &http.Client{Timeout: 15 * time.Second},
		logger: l.With("component", "fb.api"),
		apiURL: GraphBaseURL,
	}
}

// --- User profile ---

// Profile field names for the Graph API user node.
// https://developers.facebook.com/docs/messenger-platform/identity/user-profile#fields
const (
	fieldID         = "id"
	fieldFirstName  = "first_name"
	fieldLastName   = "last_name"
	fieldProfilePic = "profile_pic"
	fieldLocale     = "locale"
	fieldTimezone   = "timezone"
)

func (c *apiClient) GetUserProfile(ctx context.Context, psid, token string) (*UserProfile, error) {
	rawURL, err := buildProfileQuery(c.apiURL, psid,
		fieldID, fieldFirstName, fieldLastName, fieldProfilePic, fieldLocale, fieldTimezone,
	)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fb profile: status %d: %s", resp.StatusCode, b)
	}

	var profile UserProfile
	return &profile, json.NewDecoder(resp.Body).Decode(&profile)
}

// buildProfileQuery constructs a Graph API node URL with field selection.
func buildProfileQuery(apiURL, psid string, fields ...string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(apiURL, "/") + "/" + psid)
	if err != nil {
		return "", err
	}
	if len(fields) > 0 {
		u.RawQuery = url.Values{"fields": {strings.Join(fields, ",")}}.Encode()
	}
	return u.String(), nil
}

// --- Send API outbound types ---
// https://developers.facebook.com/docs/messenger-platform/reference/send-api

const msgTypeResponse = "RESPONSE"

type outboundPayload struct {
	Type      string            `json:"messaging_type"`
	Recipient outboundRecipient `json:"recipient"`
	Message   outboundMessage   `json:"message"`
}

type outboundRecipient struct {
	ID string `json:"id"`
}

type outboundMessage struct {
	Text       string              `json:"text,omitempty"`
	Attachment *outboundAttachment `json:"attachment,omitempty"`
}

type outboundAttachment struct {
	Type    string            `json:"type"`
	Payload outboundAttachURL `json:"payload"`
}

type outboundAttachURL struct {
	URL string `json:"url"`
}

func newTextPayload(psid, text string) outboundPayload {
	return outboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: psid},
		Message:   outboundMessage{Text: text},
	}
}

func newMediaPayload(psid, mediaType, rawURL string) outboundPayload {
	return outboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: psid},
		Message: outboundMessage{
			Attachment: &outboundAttachment{
				Type:    mediaType,
				Payload: outboundAttachURL{URL: rawURL},
			},
		},
	}
}

func (c *apiClient) send(ctx context.Context, token string, body outboundPayload) (*sharedmodel.MessageResponse, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal send payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/me/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if isTokenInvalidError(respBody) {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("fb send: status %d: %s", resp.StatusCode, respBody)
	}

	var res struct {
		ID string `json:"message_id"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		c.logger.Warn("failed to decode send response", "err", err)
	}
	return &sharedmodel.MessageResponse{ID: res.ID}, nil
}

func (c *apiClient) ParseWebhook(data []byte) (*WebhookRequest, error) {
	var r WebhookRequest
	return &r, json.Unmarshal(data, &r)
}

func (c *apiClient) SendText(ctx context.Context, token, psid, text string) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, newTextPayload(psid, text))
}

func (c *apiClient) SendMedia(ctx context.Context, token, psid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, newMediaPayload(psid, mediaType, rawURL))
}

// isTokenInvalidError reports whether the Facebook API error body signals OAuth error code 190.
func isTokenInvalidError(body []byte) bool {
	var e struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(body, &e) == nil && e.Error.Code == 190
}

package instagram

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

// GraphBaseURL is the versioned Graph API endpoint for Instagram.
// https://developers.instagram.com/docs/instagram-api/overview/authentication
const GraphBaseURL = "https://graph.instagram.com/v22.0"

// Media type constants for Instagram Send API.
const MediaImage = "image"

// graphAPI is the contract used by instagramProvider to talk to the Graph API.
// Keeping it as an interface allows the provider to be tested without network calls.
type graphAPI interface {
	GetUserProfile(ctx context.Context, igsid, token string) (*UserProfile, error)
	ParseWebhook(data []byte) (*WebhookRequest, error)
	SendText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error)
	SendMedia(ctx context.Context, token, igsid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error)
	SendInteractive(ctx context.Context, token, igsid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error)
	SendHumanAgentText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error)
	SendTyping(ctx context.Context, token, igsid string, on bool) error
}

// ErrTokenInvalid is returned when Instagram rejects the user access token (OAuth error code 190).
// The gate must be re-authorized.
//
// https://developers.instagram.com/docs/instagram-api/reference/error-handling
var ErrTokenInvalid = errors.New("instagram: user access token invalid or revoked")

// ErrTextTooLong is returned when message text exceeds 1000 characters.
var ErrTextTooLong = errors.New("instagram: message text exceeds 1000 characters")

// ErrMediaTypeNotSupported is returned when media type is not image/png/jpeg.
var ErrMediaTypeNotSupported = errors.New("instagram: only image/png/jpeg media types supported")

type apiClient struct {
	client *http.Client
	logger *slog.Logger
	apiURL string
}

var _ graphAPI = (*apiClient)(nil)

func NewAPIClient(l *slog.Logger) *apiClient {
	return &apiClient{
		client: &http.Client{Timeout: 15 * time.Second},
		logger: l.With("component", "ig.api"),
		apiURL: GraphBaseURL,
	}
}

// --- User profile ---

// Profile field names for the Graph API user node.
// https://developers.instagram.com/docs/instagram-api/reference/user
const (
	fieldID         = "id"
	fieldName       = "name"
	fieldUsername   = "username"
	fieldProfilePic = "profile_pic"
)

func (c *apiClient) GetUserProfile(ctx context.Context, igsid, token string) (*UserProfile, error) {
	rawURL, err := buildProfileQuery(c.apiURL, igsid,
		fieldID, fieldName, fieldUsername, fieldProfilePic,
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

		return nil, fmt.Errorf("ig profile: status %d: %s", resp.StatusCode, b)
	}

	var profile UserProfile

	return &profile, json.NewDecoder(resp.Body).Decode(&profile)
}

// buildProfileQuery constructs a Graph API node URL with field selection.
func buildProfileQuery(apiURL, igsid string, fields ...string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(apiURL, "/") + "/" + igsid)
	if err != nil {
		return "", err
	}

	if len(fields) > 0 {
		u.RawQuery = url.Values{"fields": {strings.Join(fields, ",")}}.Encode()
	}

	return u.String(), nil
}

// --- Send API outbound types ---
// https://developers.instagram.com/docs/instagram-api/reference/ig-user/messages

const msgTypeResponse = "RESPONSE"

type outboundPayload struct {
	Type      string            `json:"messaging_type"`
	Tag       string            `json:"tag,omitempty"`
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

func newTextPayload(igsid, text string) outboundPayload {
	return outboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: igsid},
		Message:   outboundMessage{Text: text},
	}
}

func newMediaPayload(igsid, mediaType, rawURL string) outboundPayload {
	return outboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: igsid},
		Message: outboundMessage{
			Attachment: &outboundAttachment{
				Type:    mediaType,
				Payload: outboundAttachURL{URL: rawURL},
			},
		},
	}
}

// NewHumanAgentTextPayload adds messaging_type:"MESSAGE_TAG", tag:"HUMAN_AGENT"
// alongside recipient/message for human agent takeover messages.
func NewHumanAgentTextPayload(igsid, text string) outboundPayload {
	return outboundPayload{
		Type:      "MESSAGE_TAG",
		Tag:       "HUMAN_AGENT",
		Recipient: outboundRecipient{ID: igsid},
		Message:   outboundMessage{Text: text},
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

		return nil, fmt.Errorf("ig send: status %d: %s", resp.StatusCode, respBody)
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
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	// Drop foreign payloads (object != "instagram") before they reach gate
	// resolution — a spoofed "page" body must not surface a non-200 to Meta.
	return r.ParseWebhook(), nil
}

func (c *apiClient) SendText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
	if len(text) > 1000 {
		return nil, ErrTextTooLong
	}

	return c.send(ctx, token, newTextPayload(igsid, text))
}

// SendHumanAgentText sends a text message using MESSAGE_TAG with HUMAN_AGENT tag
// for messages sent outside the 24-hour messaging window.
func (c *apiClient) SendHumanAgentText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
	if len(text) > 1000 {
		return nil, ErrTextTooLong
	}

	payload := outboundPayload{
		Type:      "MESSAGE_TAG",
		Tag:       "HUMAN_AGENT",
		Recipient: outboundRecipient{ID: igsid},
		Message:   outboundMessage{Text: text},
	}

	return c.send(ctx, token, payload)
}

// senderActionPayload is the Instagram sender_action request shape.
type senderActionPayload struct {
	Recipient    outboundRecipient `json:"recipient"`
	SenderAction string            `json:"sender_action"`
}

// SendTyping shows or hides the typing indicator for the recipient.
// It is fire-and-forget: no message id is returned.
func (c *apiClient) SendTyping(ctx context.Context, token, igsid string, on bool) error {
	action := "typing_off"
	if on {
		action = "typing_on"
	}

	body, err := json.Marshal(senderActionPayload{
		Recipient:    outboundRecipient{ID: igsid},
		SenderAction: action,
	})
	if err != nil {
		return fmt.Errorf("marshal sender action: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/me/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		if isTokenInvalidError(respBody) {
			return ErrTokenInvalid
		}

		return fmt.Errorf("ig sender_action: status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}

func (c *apiClient) SendMedia(ctx context.Context, token, igsid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error) {
	// Instagram standalone only supports image/png/jpeg delivered by URL.
	if mediaType != MediaImage {
		return nil, fmt.Errorf("%w: got %q", ErrMediaTypeNotSupported, mediaType)
	}

	return c.send(ctx, token, newMediaPayload(igsid, mediaType, rawURL))
}

// --- Interactive outbound types ---
// https://developers.instagram.com/docs/instagram-api/reference/ig-user/messages

type interactiveOutboundPayload struct {
	Type      string                     `json:"messaging_type"`
	Recipient outboundRecipient          `json:"recipient"`
	Message   interactiveOutboundMessage `json:"message"`
}

type interactiveOutboundMessage struct {
	Text       string              `json:"text,omitempty"`
	Attachment *templateAttachment `json:"attachment,omitempty"`
}

type templateAttachment struct {
	Type    string `json:"type"` // always "template"
	Payload any    `json:"payload"`
}

type igButton struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Payload string `json:"payload,omitempty"`
	URL     string `json:"url,omitempty"`
}

type igButtonTemplatePayload struct {
	TemplateType string     `json:"template_type"` // "button"
	Text         string     `json:"text"`
	Buttons      []igButton `json:"buttons"`
}

// SendInteractive sends an interactive message with buttons.
// KeyboardMarkup with only callback buttons → Button Template.
func (c *apiClient) SendInteractive(ctx context.Context, token, igsid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error) {
	msg, err := buildInteractiveMessage(body, interactive)
	if err != nil {
		return nil, err
	}

	payload := interactiveOutboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: igsid},
		Message:   msg,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal interactive payload: %w", err)
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

		return nil, fmt.Errorf("ig send interactive: status %d: %s", resp.StatusCode, respBody)
	}

	var res struct {
		ID string `json:"message_id"`
	}

	if err := json.Unmarshal(respBody, &res); err != nil {
		c.logger.Warn("failed to decode interactive send response", "err", err)
	}

	return &sharedmodel.MessageResponse{ID: res.ID}, nil
}

func buildInteractiveMessage(body string, interactive *sharedmodel.Interactive) (interactiveOutboundMessage, error) {
	if interactive == nil {
		return interactiveOutboundMessage{}, errors.New("interactive payload is nil") //nolint:goerr113 // error message varies
	}

	switch {
	case interactive.Markup != nil:
		return buildMarkupMessage(body, interactive.Markup)
	default:
		return interactiveOutboundMessage{}, errors.New("interactive has no kind set") //nolint:goerr113 // error message varies
	}
}

// buildMarkupMessage maps KeyboardMarkup to Button Template for Instagram.
func buildMarkupMessage(body string, markup *sharedmodel.KeyboardMarkup) (interactiveOutboundMessage, error) {
	var allButtons []sharedmodel.KeyboardButton

	for _, row := range markup.Rows {
		allButtons = append(allButtons, row.Buttons...)
	}

	return buildButtonTemplate(body, allButtons)
}

// buildButtonTemplate maps buttons to an Instagram Button Template.
func buildButtonTemplate(body string, buttons []sharedmodel.KeyboardButton) (interactiveOutboundMessage, error) {
	const maxButtons = 3

	igButtons := make([]igButton, 0, maxButtons)

	for _, b := range buttons {
		if len(igButtons) >= maxButtons {
			break
		}

		switch {
		case b.URL != nil:
			igButtons = append(igButtons, igButton{Type: "web_url", Title: b.Label, URL: b.URL.URL})
		case b.Callback != nil:
			igButtons = append(igButtons, igButton{Type: "postback", Title: b.Label, Payload: b.Callback.Data})
		}
	}

	if len(igButtons) == 0 {
		return interactiveOutboundMessage{}, errors.New("no valid template buttons") //nolint:goerr113 // error message varies
	}

	text := body
	if text == "" {
		text = "Choose an option"
	}

	return interactiveOutboundMessage{
		Attachment: &templateAttachment{
			Type: "template",
			Payload: igButtonTemplatePayload{
				TemplateType: "button",
				Text:         text,
				Buttons:      igButtons,
			},
		},
	}, nil
}

// isTokenInvalidError reports whether the Instagram API error body signals OAuth error code 190.
func isTokenInvalidError(body []byte) bool {
	var e struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}

	return json.Unmarshal(body, &e) == nil && e.Error.Code == 190
}

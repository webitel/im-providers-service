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
	SendInteractive(ctx context.Context, token, psid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error)
	SetMessengerProfile(ctx context.Context, token string, profile any) error
	DeleteMessengerProfile(ctx context.Context, token string, fields []string) error
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

// --- Interactive outbound types ---
// https://developers.facebook.com/docs/messenger-platform/send-messages/quick-replies
// https://developers.facebook.com/docs/messenger-platform/send-messages/template/button
// https://developers.facebook.com/docs/messenger-platform/send-messages/template/generic

type interactiveOutboundPayload struct {
	Type      string                     `json:"messaging_type"`
	Recipient outboundRecipient          `json:"recipient"`
	Message   interactiveOutboundMessage `json:"message"`
}

type interactiveOutboundMessage struct {
	Text         string               `json:"text,omitempty"`
	Attachment   *templateAttachment  `json:"attachment,omitempty"`
	QuickReplies []fbQuickReply       `json:"quick_replies,omitempty"`
}

type templateAttachment struct {
	Type    string `json:"type"` // always "template"
	Payload any    `json:"payload"`
}

type fbQuickReply struct {
	ContentType string `json:"content_type"`
	Title       string `json:"title,omitempty"`
	Payload     string `json:"payload,omitempty"`
}

type fbButton struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type fbButtonTemplatePayload struct {
	TemplateType string     `json:"template_type"` // "button"
	Text         string     `json:"text"`
	Buttons      []fbButton `json:"buttons"`
}

type fbGenericTemplatePayload struct {
	TemplateType string      `json:"template_type"` // "generic"
	Elements     []fbElement `json:"elements"`
}

type fbElement struct {
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle,omitempty"`
	Buttons  []fbButton `json:"buttons,omitempty"`
}

// SendInteractive sends a Quick Replies, Button Template, or Generic Template message
// depending on the interactive payload kind and button types.
//
// KeyboardMarkup with only callback/request buttons → Quick Replies.
// KeyboardMarkup with any URL button              → Button Template (max 3 buttons).
// KeyboardListReply                               → Generic Template (one card per section).
func (c *apiClient) SendInteractive(ctx context.Context, token, psid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error) {
	msg, err := buildInteractiveMessage(body, interactive)
	if err != nil {
		return nil, err
	}
	payload := interactiveOutboundPayload{
		Type:      msgTypeResponse,
		Recipient: outboundRecipient{ID: psid},
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
		return nil, fmt.Errorf("fb send interactive: status %d: %s", resp.StatusCode, respBody)
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
		return interactiveOutboundMessage{}, fmt.Errorf("interactive payload is nil")
	}

	switch {
	case interactive.Markup != nil:
		return buildMarkupMessage(body, interactive.Markup)
	case interactive.ListReply != nil:
		return buildListReplyMessage(interactive.ListReply)
	default:
		return interactiveOutboundMessage{}, fmt.Errorf("interactive has no kind set")
	}
}

// buildMarkupMessage maps KeyboardMarkup to Quick Replies or Button Template.
func buildMarkupMessage(body string, markup *sharedmodel.KeyboardMarkup) (interactiveOutboundMessage, error) {
	var allButtons []sharedmodel.KeyboardButton
	for _, row := range markup.Rows {
		allButtons = append(allButtons, row.Buttons...)
	}

	if hasURLButton(allButtons) {
		return buildButtonTemplate(body, allButtons)
	}
	return buildQuickReplies(body, allButtons)
}

func hasURLButton(buttons []sharedmodel.KeyboardButton) bool {
	for _, b := range buttons {
		if b.URL != nil {
			return true
		}
	}
	return false
}

// buildQuickReplies maps callback/request buttons to Facebook Quick Replies.
// https://developers.facebook.com/docs/messenger-platform/send-messages/quick-replies#sending
func buildQuickReplies(body string, buttons []sharedmodel.KeyboardButton) (interactiveOutboundMessage, error) {
	const maxQuickReplies = 13
	qrs := make([]fbQuickReply, 0, len(buttons))
	for i, b := range buttons {
		if i >= maxQuickReplies {
			break
		}
		switch {
		case b.Callback != nil:
			qrs = append(qrs, fbQuickReply{
				ContentType: "text",
				Title:       b.Label,
				Payload:     b.Callback.Data,
			})
		case b.Request != nil:
			qrs = append(qrs, mapRequestQuickReply(b.Label, b.Request.Action))
		}
	}
	if len(qrs) == 0 {
		return interactiveOutboundMessage{}, fmt.Errorf("no valid quick reply buttons")
	}
	return interactiveOutboundMessage{Text: body, QuickReplies: qrs}, nil
}

// mapRequestQuickReply converts a request action string to the corresponding FB content_type.
func mapRequestQuickReply(label, action string) fbQuickReply {
	switch action {
	case "location":
		return fbQuickReply{ContentType: "location"}
	case "phone", "user_phone_number", "contact":
		return fbQuickReply{ContentType: "user_phone_number"}
	case "email", "user_email":
		return fbQuickReply{ContentType: "user_email"}
	default:
		return fbQuickReply{ContentType: "text", Title: label, Payload: action}
	}
}

// buildButtonTemplate maps buttons (including URL) to a Facebook Button Template.
// https://developers.facebook.com/docs/messenger-platform/send-messages/template/button
func buildButtonTemplate(body string, buttons []sharedmodel.KeyboardButton) (interactiveOutboundMessage, error) {
	const maxButtons = 3
	fbButtons := make([]fbButton, 0, maxButtons)
	for _, b := range buttons {
		if len(fbButtons) >= maxButtons {
			break
		}
		switch {
		case b.URL != nil:
			fbButtons = append(fbButtons, fbButton{Type: "web_url", Title: b.Label, URL: b.URL.URL})
		case b.Callback != nil:
			fbButtons = append(fbButtons, fbButton{Type: "postback", Title: b.Label, Payload: b.Callback.Data})
		}
	}
	if len(fbButtons) == 0 {
		return interactiveOutboundMessage{}, fmt.Errorf("no valid template buttons")
	}
	text := body
	if text == "" {
		text = "Choose an option"
	}
	return interactiveOutboundMessage{
		Attachment: &templateAttachment{
			Type: "template",
			Payload: fbButtonTemplatePayload{
				TemplateType: "button",
				Text:         text,
				Buttons:      fbButtons,
			},
		},
	}, nil
}

// buildListReplyMessage maps KeyboardListReply to a Facebook Generic Template.
// https://developers.facebook.com/docs/messenger-platform/send-messages/template/generic
func buildListReplyMessage(list *sharedmodel.KeyboardListReply) (interactiveOutboundMessage, error) {
	const maxElements = 10
	const maxButtonsPerElement = 3

	elements := make([]fbElement, 0, len(list.Sections))
	for i, section := range list.Sections {
		if i >= maxElements {
			break
		}
		el := fbElement{Title: section.Section}
		for _, b := range section.Buttons {
			if len(el.Buttons) >= maxButtonsPerElement {
				break
			}
			switch {
			case b.URL != nil:
				el.Buttons = append(el.Buttons, fbButton{Type: "web_url", Title: b.Label, URL: b.URL.URL})
			case b.Callback != nil:
				el.Buttons = append(el.Buttons, fbButton{Type: "postback", Title: b.Label, Payload: b.Callback.Data})
			}
		}
		elements = append(elements, el)
	}
	if len(elements) == 0 {
		return interactiveOutboundMessage{}, fmt.Errorf("no sections in list reply")
	}
	return interactiveOutboundMessage{
		Attachment: &templateAttachment{
			Type:    "template",
			Payload: fbGenericTemplatePayload{TemplateType: "generic", Elements: elements},
		},
	}, nil
}

// --- Messenger Profile API ---
// https://developers.facebook.com/docs/messenger-platform/messenger-profile/persistent-menu

// messengerProfile is the payload for POST /me/messenger_profile.
type messengerProfile struct {
	PersistentMenu []persistentMenuLocale `json:"persistent_menu,omitempty"`
	GetStarted     *fbGetStarted          `json:"get_started,omitempty"`
}

type persistentMenuLocale struct {
	Locale                string         `json:"locale"` // "default"
	ComposerInputDisabled bool           `json:"composer_input_disabled"`
	CallToActions         []fbMenuAction `json:"call_to_actions"`
}

type fbMenuAction struct {
	Type          string         `json:"type"`                              // postback | web_url | nested
	Title         string         `json:"title"`
	Payload       string         `json:"payload,omitempty"`
	URL           string         `json:"url,omitempty"`
	CallToActions []fbMenuAction `json:"call_to_actions,omitempty"`
}

type fbGetStarted struct {
	Payload string `json:"payload"`
}

// SetMessengerProfile calls POST /me/messenger_profile to set persistent menu or get-started button.
func (c *apiClient) SetMessengerProfile(ctx context.Context, token string, profile any) error {
	raw, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshal messenger profile: %w", err)
	}

	endpoint := c.apiURL + "/me/messenger_profile"
	c.logger.DebugContext(ctx, "POST messenger_profile", "url", endpoint, "body", string(raw))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, "messenger_profile request failed", "error", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.logger.InfoContext(ctx, "messenger_profile response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		if isTokenInvalidError(body) {
			return ErrTokenInvalid
		}
		return fmt.Errorf("fb set messenger profile: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// DeleteMessengerProfile calls DELETE /me/messenger_profile to remove specific profile fields.
func (c *apiClient) DeleteMessengerProfile(ctx context.Context, token string, fields []string) error {
	raw, err := json.Marshal(struct {
		Fields []string `json:"fields"`
	}{Fields: fields})
	if err != nil {
		return fmt.Errorf("marshal delete messenger profile: %w", err)
	}

	endpoint := c.apiURL + "/me/messenger_profile"
	c.logger.DebugContext(ctx, "DELETE messenger_profile", "url", endpoint, "fields", fields)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, "delete messenger_profile request failed", "error", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.logger.InfoContext(ctx, "delete messenger_profile response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		if isTokenInvalidError(body) {
			return ErrTokenInvalid
		}
		return fmt.Errorf("fb delete messenger profile: status %d: %s", resp.StatusCode, body)
	}
	return nil
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

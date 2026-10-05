package viberbm

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// maxFileNameLen bounds the outbound file name attached to a FILE content.
const maxFileNameLen = 25

func (p *viberBMProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	res, err := p.api.SendText(ctx, g.BaseURL, g.APIKey, g.SenderName, to, req.Text, outboundMessageID(req.ID))

	return toResponse(res, to, err)
}

func (p *viberBMProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	url := firstURL(req.Images)
	if url == "" {
		return nil, vibbmmodel.ErrMediaMissing
	}

	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	res, err := p.api.SendImage(ctx, g.BaseURL, g.APIKey, g.SenderName, to, url, req.Text, outboundMessageID(req.ID))

	return toResponse(res, to, err)
}

func (p *viberBMProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	url := firstURL(req.Documents)
	if url == "" {
		return nil, vibbmmodel.ErrMediaMissing
	}

	doc := req.Documents[0]
	kind := classifyMedia(doc.MimeType, doc.FileName)

	var name string

	switch kind {
	case mediaAudio:
		return nil, vibbmmodel.ErrAudioUnsupported
	case mediaVideo:
		if !isSupportedVideo(doc.MimeType, doc.FileName) {
			return nil, vibbmmodel.ErrFileTypeUnsupported
		}
	case mediaFile:
		var err error
		if name, err = documentName(doc); err != nil {
			return nil, err
		}
	case mediaImage:
	}

	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	var res *sendResult

	// Callers without dedicated image/video RPCs (gateway) deliver all media as
	// documents; Infobip FILE content rejects image and video extensions.
	switch kind {
	case mediaImage:
		res, err = p.api.SendImage(ctx, g.BaseURL, g.APIKey, g.SenderName, to, url, req.Text, outboundMessageID(req.ID))
	case mediaVideo:
		res, err = p.sendVideo(ctx, g, to, url, req)
	case mediaFile, mediaAudio:
		res, err = p.api.SendFile(ctx, g.BaseURL, g.APIKey, g.SenderName, to, url, name, outboundMessageID(req.ID))
	}

	return toResponse(res, to, err)
}

func (p *viberBMProvider) sendVideo(ctx context.Context, g *vibbmmodel.ViberBMGate, to, url string, req *sharedmodel.Message) (*sendResult, error) {
	meta, err := p.video.Probe(ctx, url)
	if err != nil {
		return nil, err
	}

	if meta.duration > maxVideoDuration {
		return nil, vibbmmodel.ErrVideoTooLong
	}

	thumb, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: g.DomainID,
		Name:     "thumbnail.jpg",
		MimeType: "image/jpeg",
	}, bytes.NewReader(meta.thumbnail))
	if err != nil {
		return nil, fmt.Errorf("viber_bm video thumbnail upload: %w", err)
	}

	thumbURL, err := p.publicFileURL(ctx, g.DomainID, thumb.ID)
	if err != nil {
		return nil, fmt.Errorf("viber_bm video thumbnail link: %w", err)
	}

	return p.api.SendVideo(ctx, g.BaseURL, g.APIKey, g.SenderName, to, videoContent{
		MediaURL:      url,
		MediaDuration: isoDuration(meta.duration),
		ThumbnailURL:  thumbURL,
		Text:          req.Text,
	}, outboundMessageID(req.ID))
}

// publicFileURL mirrors thread-service link generation: storage answers with a
// path relative to base_url, and Infobip needs an absolute http(s) URL.
func (p *viberBMProvider) publicFileURL(ctx context.Context, domainID int64, fileID string) (string, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid storage file id %q: %w", fileID, err)
	}

	res, err := p.links.GenerateFileLink(ctx, &pbstorage.GenerateFileLinkRequest{
		DomainId: domainID,
		FileId:   id,
		Action:   "download",
		Source:   "file",
	})
	if err != nil {
		return "", err
	}

	base, err := url.Parse(res.GetBaseUrl())
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	rel, err := url.Parse(res.GetUrl())
	if err != nil {
		return "", fmt.Errorf("parse file url: %w", err)
	}

	full := base.ResolveReference(rel)
	if (full.Scheme != "https" && full.Scheme != "http") || full.Host == "" {
		return "", fmt.Errorf("storage link %q is not an absolute http(s) url", full.Redacted())
	}

	return full.String(), nil
}

// SendTemplate is the cold-start primitive (not part of provider.Sender):
// an approved, business-initiated template for outreach outside an open 24h
// session. templateID must already be approved.
func (p *viberBMProvider) SendTemplate(ctx context.Context, gateID, toSub, templateID, language string, params map[string]string) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, gateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, toSub)
	if err != nil {
		return nil, err
	}

	// Ensure the contact + via exist before a cold-start send so a later reply
	// resolves to the same contact identity. Idempotent; failures are non-fatal.
	if _, syncErr := p.syncContact(ctx, g, to, ""); syncErr != nil {
		p.logger.WarnContext(ctx, "viber_bm template: sync contact failed", "gate_id", gateID, "err", syncErr)
	}

	res, err := p.api.SendTemplate(ctx, g.BaseURL, g.APIKey, g.SenderName, to, templateID, language, params, "")

	return toResponse(res, to, err)
}

// resolveMSISDN returns the recipient MSISDN: a UUID sub is resolved to the
// contact subject via the gateway Search RPC, anything else is normalised in
// place (the UUID-shape check avoids misrouting a formatted number).
func (p *viberBMProvider) resolveMSISDN(ctx context.Context, gate *vibbmmodel.ViberBMGate, sub string) (string, error) {
	if _, err := uuid.Parse(sub); err != nil {
		return normalizeMSISDN(sub), nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{sub},
	})
	if err != nil {
		return "", fmt.Errorf("resolve msisdn for %s: %w", sub, err)
	}

	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve msisdn for %s: contact not found or has no subject", sub)
	}

	return normalizeMSISDN(items[0].GetSubject()), nil
}

// normalizeMSISDN strips a leading '+' and surrounding whitespace; InfoBip
// expects a bare international number.
func normalizeMSISDN(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "+")
}

// outboundMessageID echoes the internal message id to InfoBip as the dedup key,
// mapping the zero UUID to "" so omitempty drops it rather than sending a
// constant key for every message.
func outboundMessageID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}

	return id.String()
}

// toResponse stamps the recipient and provider message id into the response
// metadata for the outbound handler to persist for receipts.
func toResponse(res *sendResult, to string, err error) (*sharedmodel.MessageResponse, error) {
	if err != nil || res == nil {
		return nil, err
	}

	resp := &sharedmodel.MessageResponse{
		ID: res.MessageID,
		MD: map[string]any{
			"recipient_id": to,
		},
	}

	if res.BulkID != "" {
		resp.MD["bulk_id"] = res.BulkID
	}

	return resp, nil
}

// fileExtensions is the set Infobip accepts for FILE content fileName; the API
// rejects anything else with a 400 violation on messages[].content.fileName.
var fileExtensions = map[string]struct{}{
	".doc": {}, ".docx": {}, ".dot": {}, ".dotx": {},
	".xls": {}, ".xlsx": {}, ".xlsm": {}, ".xltx": {},
	".ods": {}, ".fods": {}, ".odt": {}, ".fodt": {}, ".odf": {},
	".rtf": {}, ".txt": {}, ".info": {}, ".pdf": {}, ".xps": {},
	".pdax": {}, ".eps": {}, ".csv": {},
}

// documentName builds a FILE fileName Infobip accepts: a supported, lowercased
// extension (taken from the mime type when the name lacks one) and a length
// cap that truncates the stem, never the extension.
func documentName(doc *sharedmodel.Document) (string, error) {
	var name, mimeType string
	if doc != nil {
		name, mimeType = strings.TrimSpace(doc.FileName), doc.MimeType
	}

	stem, ext := name, strings.ToLower(filepath.Ext(name))
	if _, ok := fileExtensions[ext]; ok {
		stem = strings.TrimSuffix(name, filepath.Ext(name))
	} else if ext = extensionByMime(mimeType); ext == "" {
		return "", vibbmmodel.ErrFileTypeUnsupported
	}

	if stem == "" {
		stem = "file"
	}

	if runes, limit := []rune(stem), maxFileNameLen-len(ext); len(runes) > limit {
		stem = string(runes[:limit])
	}

	return stem + ext, nil
}

func extensionByMime(mimeType string) string {
	if mimeType == "" {
		return ""
	}

	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return ""
	}

	for _, ext := range exts {
		if _, ok := fileExtensions[ext]; ok {
			return ext
		}
	}

	return ""
}

type urlGetter interface {
	GetURL() string
}

func firstURL[T urlGetter](items []T) string {
	if len(items) == 0 {
		return ""
	}

	return items[0].GetURL()
}

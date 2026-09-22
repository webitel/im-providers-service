package custom

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const maxInboundBytes = 25 << 20 // 25 MiB

type syncedMedia struct {
	id       string
	name     string
	mimeType string
	size     int64
}

func (p *customProvider) handleFile(ctx context.Context, in *inbound) {
	media, err := p.syncMedia(ctx, in.gate, in.msg.File)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to sync custom file", "url", in.msg.File.URL, "err", err)

		return
	}

	if _, err := p.messenger.SendDocument(ctx, &sharedmodel.SendDocumentRequest{
		DomainID: in.gate.DomainID,
		From:     in.peers.from,
		To:       in.peers.to,
		Document: sharedmodel.DocumentRequest{
			Body: in.msg.Text,
			Documents: []*sharedmodel.Document{{
				ID:       media.id,
				FileName: media.name,
				MimeType: media.mimeType,
				Size:     media.size,
			}},
		},
		ExternalID:        in.externalID,
		ReplyToExternalID: in.msg.ReplyTo,
		Variables:         in.variables,
	}); err != nil {
		p.logger.ErrorContext(ctx, "failed to send document", "file_name", media.name, "err", err)
	}
}

// syncMedia copies the file into Webitel storage instead of passing the
// external URL on to the core. The customer's link may be short-lived or
// reachable only from here, and an operator opening a chat months later still
// needs the attachment.
func (p *customProvider) syncMedia(ctx context.Context, gate *custommodel.CustomGate, file *wireFile) (*syncedMedia, error) {
	resp, err := p.fetch.get(ctx, gate, file.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	mimeType := resolveMime(file, resp.Header.Get("Content-Type"))
	name := fileName(file, mimeType)

	uploaded, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: gate.DomainID,
		Name:     name,
		MimeType: mimeType,
	}, io.LimitReader(resp.Body, maxInboundBytes))
	if err != nil {
		return nil, err
	}

	size := file.Size
	if size <= 0 {
		size = resp.ContentLength
	}

	if size <= 0 {
		size = 1
	}

	return &syncedMedia{id: uploaded.ID, name: name, mimeType: mimeType, size: size}, nil
}

func resolveMime(file *wireFile, headerContentType string) string {
	if m := normalizeMime(file.Mime); m != "" && m != "application/octet-stream" {
		return m
	}

	if m := normalizeMime(headerContentType); m != "" && m != "application/octet-stream" {
		return m
	}

	if ext := strings.ToLower(filepath.Ext(file.Name)); ext != "" {
		if m := normalizeMime(mime.TypeByExtension(ext)); m != "" {
			return m
		}
	}

	return "application/octet-stream"
}

func normalizeMime(value string) string {
	if value == "" {
		return ""
	}

	parsed, _, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}

	return strings.ToLower(parsed)
}

func fileName(file *wireFile, mimeType string) string {
	if file.Name != "" {
		return file.Name
	}

	if base := path.Base(file.URL); base != "" && base != "." && base != "/" && filepath.Ext(base) != "" {
		return base
	}

	ext := ".bin"
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		ext = exts[0]
	}

	return fmt.Sprintf("custom_file_%d%s", time.Now().Unix(), ext)
}

package facebook

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	client "github.com/webitel/im-providers-service/infra/client/grpc"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/provider/facebook/graph"
	"github.com/webitel/im-providers-service/internal/provider/facebook/payload"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gateIdentity string

func (g gateIdentity) Identity() string { return string(g) }

type facebookProvider struct {
	api       *Client
	logger    *slog.Logger
	messenger service.Messenger
	gateCache store.GateCache
	userCache store.ExternalUserCache
	repo      store.FacebookStore
	gatewayer *imgateway.Client
	media     service.MediaManager
}

func New(
	m service.Messenger,
	l *slog.Logger,
	gc store.GateCache,
	uc store.ExternalUserCache,
	repo store.FacebookStore,
	gatewayer *imgateway.Client,
	media service.MediaManager,
) provider.Provider {
	return &facebookProvider{
		api:       NewClient(l),
		logger:    l.With("provider", "facebook"),
		messenger: m,
		gateCache: gc,
		userCache: uc,
		repo:      repo,
		gatewayer: gatewayer,
		media:     media,
	}
}

func (p *facebookProvider) Type() string { return "facebook" }

func (p *facebookProvider) HandleWebhook(ctx context.Context, data []byte) error {
	uri := p.normalizeURI(ctx)
	req, err := p.api.ParseWebhook(data)
	if err != nil || req == nil || len(req.Entry) == 0 {
		return nil
	}

	gate, err := p.resolveGate(ctx, uri, req.Entry[0].ID)
	if err != nil || !gate.Enabled {
		return err
	}

	for _, m := range req.AllMessages() {
		psid := m.Sender.ID
		if psid == "" {
			continue
		}

		fbusr, err := p.api.GetUserProfile(ctx, psid, gate.PageToken)
		if err != nil {
			p.logger.Error("failed to fetch user profile", "psid", psid, "err", err)
			continue
		}

		contact, err := p.externalUserSync(ctx, gate, psid, fbusr)
		if err != nil {
			p.logger.Error("sync failed", "psid", psid, "err", err)
			continue
		}

		from := model.Peer{Sub: contact.Sub, Iss: gate.Peer.Iss}
		to := model.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss}

		if m.Message != nil {
			if m.Message.Text != "" {
				_, _ = p.messenger.SendText(ctx, &model.SendTextRequest{
					From:     from,
					To:       to,
					Body:     m.Message.Text,
					DomainID: gate.DomainID,
				})
			}

			if len(m.Message.Attachments) > 0 {
				p.handleAttachments(ctx, gate, from, to, m.Message.Attachments)
			}
		}
	}
	return nil
}

func (p *facebookProvider) handleAttachments(ctx context.Context, gate *model.FacebookGate, from, to model.Peer, attachments []payload.InboundAttachment) {
	for _, attach := range attachments {
		fbURL := attach.Payload.URL
		if fbURL == "" {
			continue
		}

		fileName := p.generateFileName(attach)
		res, err := p.syncMedia(ctx, gate, fbURL, fileName)
		if err != nil {
			p.logger.Error("failed to sync media", "url", fbURL, "err", err)
			continue
		}

		if res.Size <= 0 {
			res.Size = 1
		}

		switch attach.Type {
		case "image":
			_, _ = p.messenger.SendImage(ctx, &model.SendImageRequest{
				DomainID: gate.DomainID,
				From:     from,
				To:       to,
				Image: model.ImageRequest{
					Images: []*model.Image{{
						ID:       res.ID,
						FileName: fileName,
						MimeType: res.MimeType,
					}},
				},
			})
		case "video", "audio", "file":
			_, _ = p.messenger.SendDocument(ctx, &model.SendDocumentRequest{
				DomainID: gate.DomainID,
				From:     from,
				To:       to,
				Document: model.DocumentRequest{
					Documents: []*model.Document{{
						ID:       res.ID,
						FileName: fileName,
						MimeType: res.MimeType,
						Size:     res.Size,
					}},
				},
			})
		}
	}
}

type SyncMediaResponse struct {
	ID       string
	MimeType string
	Size     int64
}

func (p *facebookProvider) syncMedia(ctx context.Context, gate *model.FacebookGate, fbURL, fileName string) (*SyncMediaResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fbURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("access_token", gate.PageToken)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fb download failed with status: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	size := resp.ContentLength

	res, err := p.media.UploadFile(ctx, model.UploadRequest{
		DomainID: gate.DomainID,
		Name:     fileName,
		MimeType: contentType,
	}, resp.Body)
	if err != nil {
		return nil, err
	}

	return &SyncMediaResponse{
		ID:       res.ID,
		MimeType: contentType,
		Size:     size,
	}, nil
}

func (p *facebookProvider) externalUserSync(ctx context.Context, g *model.FacebookGate, psid string, prof *UserProfile) (*gatewayv1.Contact, error) {
	u := &model.ExternalUser{ID: psid, FirstName: prof.FirstName, LastName: prof.LastName}

	if ok, _ := p.userCache.IsKnown(ctx, u); ok {
		return &gatewayv1.Contact{Sub: psid}, nil
	}

	authCtx := client.WithIdentity(ctx, gateIdentity(fmt.Sprintf("%d.%s", g.DomainID, g.Peer.Sub)))

	internalUsr, err := p.gatewayer.Create(
		authCtx,
		&gatewayv1.CreateContactRequest{
			IssId:    g.Peer.Iss,
			Type:     p.Type(),
			Name:     u.FirstName,
			Username: u.LastName,
			Subject:  u.ID,
			IsBot:    false,
		})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			_ = p.userCache.MarkKnown(ctx, u)
			return &gatewayv1.Contact{Sub: psid}, nil
		}
		return nil, fmt.Errorf("gateway sync failed: %w", err)
	}

	_ = p.userCache.MarkKnown(ctx, u)
	return internalUsr, nil
}

func (p *facebookProvider) generateFileName(attach payload.InboundAttachment) string {
	name := attach.Payload.Title
	if name == "" {
		name = attach.Payload.Name
	}
	if name != "" {
		return name
	}

	ext := ".bin"
	switch attach.Type {
	case "image":
		ext = ".jpg"
	case "video":
		ext = ".mp4"
	case "audio":
		ext = ".mp3"
	}
	return fmt.Sprintf("fb_%s_%d%s", attach.Type, time.Now().Unix(), ext)
}

func (p *facebookProvider) Verify(ctx context.Context, query url.Values) (string, error) {
	req := graph.ParseVerify(query)
	if req.Mode != "subscribe" {
		return "", fmt.Errorf("unexpected hub.mode: %s", req.Mode)
	}
	return req.Challenge, nil
}

func (p *facebookProvider) resolveGate(ctx context.Context, uri, pageID string) (*model.FacebookGate, error) {
	k := uri + ":" + pageID
	if _, ok := p.gateCache.Get(k); ok {
		return p.repo.SelectByPageAndURI(ctx, pageID, uri)
	}
	g, err := p.repo.SelectByPageAndURI(ctx, pageID, uri)
	if err != nil {
		return nil, err
	}
	p.gateCache.Set(k, store.GateState{
		GateID: g.ID, Enabled: g.Enabled, Issuer: g.Peer.Iss, Sub: g.Peer.Sub, Domain: g.DomainID,
	})
	return g, nil
}

func (p *facebookProvider) normalizeURI(ctx context.Context) string {
	uri, _ := ctx.Value("webhook_uri").(string)
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}
	return uri
}

// Outbound methods
func (p *facebookProvider) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.PageToken, req.To.Sub, req.Text)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	u := ""
	if len(req.Images) > 0 {
		u = req.Images[0].URL
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaImage, u)
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	u := ""
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaFile, u)
}

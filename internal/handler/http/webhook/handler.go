package webhook

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/webitel/im-providers-service/internal/provider"
)

type Handler struct {
	logger    *slog.Logger
	providers map[string]provider.Provider
}

func NewHandler(logger *slog.Logger, providers []provider.Provider) *Handler {
	m := make(map[string]provider.Provider)
	for _, p := range providers {
		m[p.Type()] = p
	}
	return &Handler{
		logger:    logger,
		providers: m,
	}
}

// ServeHTTP acts as a generic entry point for all webhooks.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pType := chi.URLParam(r, "provider")
	uri := chi.URLParam(r, "uri")

	p, ok := h.providers[pType]
	if !ok {
		h.logger.Warn("webhook received for unknown provider", "type", pType)
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}

	// [GET] Verification (for Facebook/WhatsApp/etc.)
	if r.Method == http.MethodGet {
		if v, ok := p.(provider.Verifier); ok {
			gCtx := context.WithValue(r.Context(), provider.WebhookURIKey, uri)
			challenge, err := v.Verify(gCtx, r.URL.Query())
			if err != nil {
				h.logger.Error("verification failed", "provider", pType, "error", err)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(challenge))
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read body", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	h.logger.Debug("incoming webhook", "provider", pType, "uri", uri, "body", string(body))

	ctx := context.WithValue(r.Context(), provider.WebhookURIKey, uri)

	if sv, ok := p.(provider.SignatureValidator); ok {
		sig := r.Header.Get("X-Hub-Signature-256")
		if err := sv.ValidateSignature(ctx, sig, body); err != nil {
			h.logger.Warn("signature validation failed", "provider", pType, "uri", uri, "err", err)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	if err := p.HandleWebhook(ctx, body); err != nil {
		h.logger.Error("processing failed", "provider", pType, "uri", uri, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

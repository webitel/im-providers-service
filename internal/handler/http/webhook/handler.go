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

	// [CLEAN_VERIFICATION]: Handle GET requests using the Verifier interface.
	// This removes provider-specific hardcoding from the handler.
	if r.Method == http.MethodGet {
		if v, ok := p.(provider.Verifier); ok {
			challenge, err := v.Verify(r.Context(), r.URL.Query())
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

	// [POST_PROCESSING]: Standard webhook event handling.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read body", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Inject 'uri' into context for provider database lookups.
	ctx := context.WithValue(r.Context(), "webhook_uri", uri)
	if err := p.HandleWebhook(ctx, body); err != nil {
		h.logger.Error("processing failed", "provider", pType, "uri", uri, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

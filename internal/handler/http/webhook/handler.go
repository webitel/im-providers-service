package webhook

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/webitel/im-providers-service/internal/provider"
)

// Handler processes incoming webhooks from all registered messaging platforms.
type Handler struct {
	logger    *slog.Logger
	providers map[string]provider.Provider
}
	
	






// NewHandler initializes the handler with a registry of providers.
// [CONSTRUCTOR] Used by fx.Module.
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

// ServeHTTP handles the unified webhook entry point.
// [ROUTING] Pattern: POST /wh/{provider}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract provider type from URL using chi
	pType := chi.URLParam(r, "provider")

	p, ok := h.providers[pType]
	if !ok {
		h.logger.Warn("webhook received for unknown provider", "type", pType)
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// [DISPATCH] Route the raw payload to the specific adapter
	if err := p.HandleWebhook(r.Context(), body); err != nil {
		h.logger.Error("provider failed to process webhook",
			"provider", pType,
			"error", err,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

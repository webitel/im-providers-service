package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read body", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	h.prettyPrintJSON(pType, uri, body)

	ctx := context.WithValue(r.Context(), "webhook_uri", uri)
	if err := p.HandleWebhook(ctx, body); err != nil {
		h.logger.Error("processing failed", "provider", pType, "uri", uri, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) prettyPrintJSON(pType, uri string, data []byte) {
	if len(data) == 0 {
		return
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		h.logger.Debug("incoming webhook body (non-json)", "provider", pType, "body", string(data))
		return
	}

	const (
		reset  = "\033[0m"
		bold   = "\033[1m"
		cyan   = "\033[36m"
		green  = "\033[32m"
		yellow = "\033[33m"
		gray   = "\033[90m"
	)

	fmt.Printf("\n%s%s─── %s RECEIVED WEBHOOK [%s | %s] ───%s\n", bold, cyan, pType, pType, uri, reset)
	fmt.Printf("%s%s%s\n", green, prettyJSON.String(), reset)
	fmt.Printf("%s%s──────────────────────────────────────────────────%s\n\n", bold, gray, reset)
}

package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

// maxResponseBytes caps how much of the external system's reply we read: the
// contract's response is a two-field object, and an endpoint answering with a
// page of HTML must not be able to hold a buffer open.
const maxResponseBytes = 8 << 10 // 8 KiB

type client struct {
	logger *slog.Logger
	http   *http.Client
}

func newClient(logger *slog.Logger) *client {
	return &client{
		logger: logger.With("component", "custom.callback"),
		http:   &http.Client{},
	}
}

func (c *client) post(ctx context.Context, gate *custommodel.CustomGate, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, gate.RequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gate.CallbackURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%w: build request: %w", custommodel.ErrCallbackRejected, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signHeader, sign(payload, gate.AppSecret))

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", custommodel.ErrCallbackFailed, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%w: status %s", custommodel.ErrCallbackFailed, resp.Status)
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var decoded wireResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		c.logger.DebugContext(ctx, "callback response not understood, treating as accepted", "err", err)

		return nil
	}

	if !bool(decoded.Success) && decoded.Error != "" {
		return fmt.Errorf("%w: %s", custommodel.ErrCallbackRejected, decoded.Error)
	}

	return nil
}

func nowMilli() int64 {
	return time.Now().UnixMilli()
}

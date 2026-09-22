package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

// maxResponseBytes caps how much of the external system's reply we read: the
// contract's response is a two-field object, and an endpoint answering with a
// page of HTML must not be able to hold a buffer open.
const maxResponseBytes = 8 << 10 // 8 KiB

const postErrID = "custom.client.post"

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
		return errors.Append(custommodel.ErrCallbackRejected, "build request: "+err.Error(),
			errors.WithCause(err), errors.WithID(postErrID))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signHeader, sign(payload, gate.AppSecret))

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.Append(custommodel.ErrCallbackFailed, err.Error(),
			errors.WithCause(err), errors.WithID(postErrID))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.Append(custommodel.ErrCallbackFailed, "status "+resp.Status,
			errors.WithID(postErrID))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return errors.Append(custommodel.ErrCallbackFailed, "read response: "+err.Error(),
			errors.WithCause(err), errors.WithID(postErrID))
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
		return errors.Append(custommodel.ErrCallbackRejected, decoded.Error,
			errors.WithID(postErrID))
	}

	return nil
}

func nowMilli() int64 {
	return time.Now().UnixMilli()
}

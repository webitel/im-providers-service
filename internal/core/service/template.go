package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"text/template"

	lru "github.com/hashicorp/golang-lru/v2"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
)

// SystemEventType identifies a system event that can be rendered as a user-facing message.
type SystemEventType = string

const (
	EventMemberAdded   SystemEventType = "member_added"
	EventMemberRemoved SystemEventType = "member_removed"
	EventTransferred   SystemEventType = "transferred"
)

const contactIDSuffix = "_contact_id"

// TemplateRenderer resolves and renders system event templates.
//
// If no gate-specific override exists in the store, Render returns an empty string
// and the caller must skip sending the message entirely.
//
// Before rendering, any var key ending in _contact_id is resolved to a contact
// name and injected as <prefix>_name so templates can reference it.
type TemplateRenderer struct {
	store        sharedstore.TemplateStore
	contacts     *imcontact.Client
	contactNames *lru.Cache[string, string]
	logger       *slog.Logger
}

// NewTemplateRenderer creates a TemplateRenderer.
// store and contacts may be nil — useful in tests.
func NewTemplateRenderer(store sharedstore.TemplateStore, contacts *imcontact.Client, logger *slog.Logger) *TemplateRenderer {
	cache, _ := lru.New[string, string](512)
	return &TemplateRenderer{
		store:        store,
		contacts:     contacts,
		contactNames: cache,
		logger:       logger.With("component", "template_renderer"),
	}
}

// Render resolves the template for (gateID, eventType) and executes it with vars.
// Returns an empty string when no override is configured — the caller must skip sending.
func (r *TemplateRenderer) Render(ctx context.Context, gateID, eventType string, vars map[string]string) string {
	tpl := r.resolve(ctx, gateID, eventType)

	enriched := r.enrichWithNames(ctx, vars)

	result, err := executeTemplate(tpl, enriched)
	if err != nil {
		r.logger.WarnContext(ctx, "template execution failed, skipping message",
			"gate_id", gateID,
			"event_type", eventType,
			"err", err,
		)
		return ""
	}
	return result
}

func (r *TemplateRenderer) resolve(ctx context.Context, gateID, eventType string) string {
	if r.store != nil {
		tpl, err := r.store.GetTemplate(ctx, gateID, eventType)
		if err == nil {
			return tpl
		}
		if !errors.Is(err, sharedstore.ErrNotFound) {
			r.logger.WarnContext(ctx, "template store lookup failed, falling back to default",
				"gate_id", gateID,
				"event_type", eventType,
				"err", err,
			)
		}
	}

	return ""
}

// enrichWithNames scans vars for keys ending in _contact_id, resolves each to a
// display name, and adds a corresponding _name key. The original map is not mutated.
func (r *TemplateRenderer) enrichWithNames(ctx context.Context, vars map[string]string) map[string]string {
	if r.contacts == nil || len(vars) == 0 {
		return vars
	}

	type pair struct{ prefix, id string }
	var toResolve []pair

	for k, v := range vars {
		if strings.HasSuffix(k, contactIDSuffix) && v != "" {
			prefix := strings.TrimSuffix(k, contactIDSuffix)
			toResolve = append(toResolve, pair{prefix, v})
		}
	}
	if len(toResolve) == 0 {
		return vars
	}

	enriched := make(map[string]string, len(vars)+len(toResolve))
	for k, v := range vars {
		enriched[k] = v
	}

	for _, p := range toResolve {
		enriched[p.prefix+"_name"] = r.resolveName(ctx, p.id)
	}
	return enriched
}

// resolveName returns the display name for a contact ID, using LRU as a first layer.
func (r *TemplateRenderer) resolveName(ctx context.Context, contactID string) string {
	if name, ok := r.contactNames.Get(contactID); ok {
		return name
	}

	resp, err := r.contacts.SearchContact(ctx, &contactv1.SearchContactRequest{
		Ids:  []string{contactID},
		Size: 1,
	})
	if err != nil || resp == nil || len(resp.GetContacts()) == 0 {
		r.logger.WarnContext(ctx, "contact name resolution failed", "contact_id", contactID, "err", err)
		return contactID
	}

	name := resp.GetContacts()[0].GetName()
	if name == "" {
		name = resp.GetContacts()[0].GetUsername()
	}
	r.contactNames.Add(contactID, name)
	return name
}

// executeTemplate parses tpl as a Go text/template and executes it with vars as the data map.
// Map keys are accessed with {{.key_name}} syntax. Missing keys produce an empty string
// rather than an error (Option "missingkey=zero").
func executeTemplate(tpl string, vars map[string]string) (string, error) {
	t, err := template.New("").Option("missingkey=zero").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

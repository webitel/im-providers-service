package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	corestore "github.com/webitel/im-providers-service/internal/core/store"
)

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// -- mock TemplateStore --

type mockTemplateStore struct {
	tpl string
	err error
}

func (m *mockTemplateStore) GetTemplate(_ context.Context, _, _ string) (string, error) {
	return m.tpl, m.err
}
func (m *mockTemplateStore) SetTemplate(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}
func (m *mockTemplateStore) DeleteTemplate(_ context.Context, _, _ string) error { return nil }
func (m *mockTemplateStore) ListTemplates(_ context.Context, _ string) ([]corestore.TemplateRow, error) {
	return nil, nil
}

func renderer(store corestore.TemplateStore) *TemplateRenderer {
	return NewTemplateRenderer(store, nil, noopLogger)
}

// -- executeTemplate --

func TestExecuteTemplate_Basic(t *testing.T) {
	out, err := executeTemplate("Hello {{.name}}", map[string]string{"name": "World"})
	if err != nil || out != "Hello World" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestExecuteTemplate_MissingKeyIsEmpty(t *testing.T) {
	out, err := executeTemplate("Hello {{.missing}}", map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello " {
		t.Errorf("expected empty interpolation, got %q", out)
	}
}

func TestExecuteTemplate_InvalidSyntax(t *testing.T) {
	_, err := executeTemplate("{{.unclosed", nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// -- TemplateRenderer.Render --

func TestRender_NoStore_ReturnsEmpty(t *testing.T) {
	r := renderer(nil)
	got := r.Render(context.Background(), "gate1", EventMemberAdded, map[string]string{
		"new_member_name": "Alice",
	})
	if got != "" {
		t.Errorf("expected empty string when no store, got %q", got)
	}
}

func TestRender_StoreNotFound_ReturnsEmpty(t *testing.T) {
	r := renderer(&mockTemplateStore{err: corestore.ErrNotFound})
	got := r.Render(context.Background(), "gate1", EventMemberAdded, nil)
	if got != "" {
		t.Errorf("expected empty string on ErrNotFound, got %q", got)
	}
}

func TestRender_StoreError_ReturnsEmpty(t *testing.T) {
	r := renderer(&mockTemplateStore{err: errors.New("db down")})
	got := r.Render(context.Background(), "gate1", EventTransferred, nil)
	if got != "" {
		t.Errorf("expected empty string on store error, got %q", got)
	}
}

func TestRender_StoreOverride_Rendered(t *testing.T) {
	r := renderer(&mockTemplateStore{tpl: "{{.new_member_name}} joined"})
	got := r.Render(context.Background(), "gate1", EventMemberAdded, map[string]string{
		"new_member_name": "Bob",
	})
	if got != "Bob joined" {
		t.Errorf("got %q, want %q", got, "Bob joined")
	}
}

func TestRender_BlankStoreOverride_ReturnsEmpty(t *testing.T) {
	r := renderer(&mockTemplateStore{tpl: ""})
	got := r.Render(context.Background(), "gate1", EventMemberAdded, nil)
	if got != "" {
		t.Errorf("expected empty string for blank template, got %q", got)
	}
}

func TestRender_InvalidStoreTemplate_ReturnsEmpty(t *testing.T) {
	r := renderer(&mockTemplateStore{tpl: "{{.unclosed"})
	got := r.Render(context.Background(), "gate1", EventMemberAdded, nil)
	if got != "" {
		t.Errorf("expected empty string on invalid template, got %q", got)
	}
}

// -- enrichWithNames --

func TestEnrichWithNames_NilContacts_PassThrough(t *testing.T) {
	r := renderer(nil)
	vars := map[string]string{"new_member_contact_id": "123", "foo": "bar"}
	got := r.enrichWithNames(context.Background(), vars)
	if got["new_member_contact_id"] != "123" || got["foo"] != "bar" {
		t.Errorf("unexpected mutation: %v", got)
	}
	if _, ok := got["new_member_name"]; ok {
		t.Error("name key should not be injected when contacts client is nil")
	}
}

func TestEnrichWithNames_NoContactIDKeys_PassThrough(t *testing.T) {
	r := renderer(nil)
	got := r.enrichWithNames(context.Background(), map[string]string{"some_field": "value"})
	if len(got) != 1 {
		t.Errorf("unexpected keys: %v", got)
	}
}

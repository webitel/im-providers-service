package service

import (
	"context"
	"log/slog"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
)

var _ FacebookManager = (*FacebookService)(nil)

type FacebookManager interface {
	CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error)
	GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
	UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error)
	DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)

	SetPersistentMenu(ctx context.Context, gateID string, items []fbmodel.MenuItem, composerDisabled bool) error
	DeletePersistentMenu(ctx context.Context, gateID string) error
	SetGetStarted(ctx context.Context, gateID string, payload string) error
	DeleteGetStarted(ctx context.Context, gateID string) error
}

// MessengerProfileAPI is the subset of the Graph API used for Messenger Profile operations.
// Defined here (exported) so the parent facebook package can satisfy it without an import cycle.
// profile is passed as any so the concrete type can be injected without creating a cycle
// between this package and internal/facebook.
type MessengerProfileAPI interface {
	SetMessengerProfile(ctx context.Context, token string, profile any) error
	DeleteMessengerProfile(ctx context.Context, token string, fields []string) error
}

// messengerProfilePayload mirrors facebook.messengerProfile but lives in this package
// to break the import cycle.
type messengerProfilePayload struct {
	PersistentMenu []persistentMenuLocale `json:"persistent_menu,omitempty"`
	GetStarted     *getStartedPayload     `json:"get_started,omitempty"`
}

type persistentMenuLocale struct {
	Locale                string       `json:"locale"`
	ComposerInputDisabled bool         `json:"composer_input_disabled"`
	CallToActions         []menuAction `json:"call_to_actions"`
}

type menuAction struct {
	Type               string       `json:"type"`
	Title              string       `json:"title"`
	Payload            string       `json:"payload,omitempty"`
	URL                string       `json:"url,omitempty"`
	// webview_height_ratio is required by FB for web_url buttons in persistent menu.
	// https://developers.facebook.com/docs/messenger-platform/messenger-profile/persistent-menu
	WebviewHeightRatio string       `json:"webview_height_ratio,omitempty"`
	CallToActions      []menuAction `json:"call_to_actions,omitempty"`
}

type getStartedPayload struct {
	Payload string `json:"payload"`
}

type FacebookService struct {
	repo     fbstore.FacebookStore
	graphAPI MessengerProfileAPI
	log      *slog.Logger
}

func NewFacebookService(repo fbstore.FacebookStore, graphAPI MessengerProfileAPI, log *slog.Logger) *FacebookService {
	return &FacebookService{
		repo:     repo,
		graphAPI: graphAPI,
		log:      log.With("layer", "service", "domain", "facebook_gate"),
	}
}

func (f *FacebookService) CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	gate := &fbmodel.FacebookGate{
		Name:      req.Name,
		MetaAppID: req.MetaAppID,
		PageID:    req.PageID,
		PageToken: req.PageToken,
		Peer:      req.Peer,
		Enabled:   true,
	}

	if err := f.repo.Insert(ctx, req.Dc, gate); err != nil {
		f.log.Error("failed to create facebook gate", "page_id", req.PageID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate created", "id", gate.ID, "page_name", gate.Name)
	return gate, nil
}

func (f *FacebookService) GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	return f.repo.Select(ctx, id)
}

func (f *FacebookService) UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error) {
	gate, err := f.repo.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	req.ApplyTo(gate)

	if err := f.repo.Update(ctx, gate); err != nil {
		f.log.Error("failed to update facebook gate", "id", req.ID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate updated", "id", gate.ID)
	return gate, nil
}

func (f *FacebookService) DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	gate, err := f.repo.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	// Unbind only removes the Facebook-specific configuration (the "tab")
	if err := f.repo.Unbind(ctx, id); err != nil {
		f.log.Error("failed to unbind facebook gate", "id", id, "err", err)
		return nil, err
	}

	f.log.Warn("facebook gate configuration removed", "id", id, "page_id", gate.PageID)
	return gate, nil
}

// SetPersistentMenu pushes a persistent menu to the Messenger Profile for the given gate's page.
func (f *FacebookService) SetPersistentMenu(ctx context.Context, gateID string, items []fbmodel.MenuItem, composerDisabled bool) error {
	gate, err := f.repo.Select(ctx, gateID)
	if err != nil {
		f.log.ErrorContext(ctx, "failed to fetch gate", "gate_id", gateID, "error", err)
		return err
	}

	f.log.InfoContext(ctx, "pushing persistent menu to FB", "gate_id", gateID, "page_id", gate.PageID, "items", len(items))
	profile := messengerProfilePayload{
		PersistentMenu: []persistentMenuLocale{
			{
				Locale:                "default",
				ComposerInputDisabled: composerDisabled,
				CallToActions:         menuItemsToActions(items, false),
			},
		},
	}
	if err := f.graphAPI.SetMessengerProfile(ctx, gate.PageToken, profile); err != nil {
		f.log.ErrorContext(ctx, "FB API rejected set persistent menu", "gate_id", gateID, "page_id", gate.PageID, "error", err)
		return err
	}
	f.log.InfoContext(ctx, "persistent menu set on Facebook", "gate_id", gateID, "page_id", gate.PageID)
	return nil
}

// DeletePersistentMenu removes the persistent_menu field from the Messenger Profile.
func (f *FacebookService) DeletePersistentMenu(ctx context.Context, gateID string) error {
	gate, err := f.repo.Select(ctx, gateID)
	if err != nil {
		f.log.ErrorContext(ctx, "failed to fetch gate", "gate_id", gateID, "error", err)
		return err
	}

	f.log.InfoContext(ctx, "deleting persistent menu from FB", "gate_id", gateID, "page_id", gate.PageID)
	if err := f.graphAPI.DeleteMessengerProfile(ctx, gate.PageToken, []string{"persistent_menu"}); err != nil {
		f.log.ErrorContext(ctx, "FB API rejected delete persistent menu", "gate_id", gateID, "page_id", gate.PageID, "error", err)
		return err
	}
	f.log.InfoContext(ctx, "persistent menu deleted from Facebook", "gate_id", gateID, "page_id", gate.PageID)
	return nil
}

// SetGetStarted sets the Get Started button payload on the Messenger Profile.
func (f *FacebookService) SetGetStarted(ctx context.Context, gateID string, payload string) error {
	gate, err := f.repo.Select(ctx, gateID)
	if err != nil {
		f.log.ErrorContext(ctx, "failed to fetch gate", "gate_id", gateID, "error", err)
		return err
	}

	f.log.InfoContext(ctx, "pushing get started button to FB", "gate_id", gateID, "page_id", gate.PageID, "payload", payload)
	profile := messengerProfilePayload{
		GetStarted: &getStartedPayload{Payload: payload},
	}
	if err := f.graphAPI.SetMessengerProfile(ctx, gate.PageToken, profile); err != nil {
		f.log.ErrorContext(ctx, "FB API rejected set get started", "gate_id", gateID, "page_id", gate.PageID, "error", err)
		return err
	}
	f.log.InfoContext(ctx, "get started button set on Facebook", "gate_id", gateID, "page_id", gate.PageID)
	return nil
}

// DeleteGetStarted removes the get_started field from the Messenger Profile.
func (f *FacebookService) DeleteGetStarted(ctx context.Context, gateID string) error {
	gate, err := f.repo.Select(ctx, gateID)
	if err != nil {
		f.log.ErrorContext(ctx, "failed to fetch gate", "gate_id", gateID, "error", err)
		return err
	}

	f.log.InfoContext(ctx, "deleting get started button from FB", "gate_id", gateID, "page_id", gate.PageID)
	if err := f.graphAPI.DeleteMessengerProfile(ctx, gate.PageToken, []string{"get_started"}); err != nil {
		f.log.ErrorContext(ctx, "FB API rejected delete get started", "gate_id", gateID, "page_id", gate.PageID, "error", err)
		return err
	}
	f.log.InfoContext(ctx, "get started button deleted from Facebook", "gate_id", gateID, "page_id", gate.PageID)
	return nil
}

// menuItemsToActions converts domain menu items to the Graph API call-to-action structure.
// Facebook persistent menu only supports postback and web_url types (no nested).
// Items with nested children are flattened into the parent list.
func menuItemsToActions(items []fbmodel.MenuItem, _ bool) []menuAction {
	actions := make([]menuAction, 0, len(items))
	for _, item := range items {
		switch {
		case len(item.Nested) > 0:
			// FB persistent menu does not support nested type — flatten children into the list.
			actions = append(actions, menuItemsToActions(item.Nested, false)...)
		case item.URL != "":
			// webview_height_ratio is required by the FB API for web_url buttons.
			actions = append(actions, menuAction{
				Type:               "web_url",
				Title:              item.Title,
				URL:                item.URL,
				WebviewHeightRatio: "full",
			})
		default:
			actions = append(actions, menuAction{
				Type:    "postback",
				Title:   item.Title,
				Payload: item.Payload,
			})
		}
	}
	return actions
}

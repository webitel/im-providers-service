package model

import "testing"

func TestUpdateViberBM_ApplyTo_EmptyStringsKeepExisting(t *testing.T) {
	gate := &ViberBMGate{
		Name:       "IraViberBM",
		BaseURL:    "https://x18vwq.api.infobip.com",
		SenderName: "IBSelfServe",
		APIKey:     "old-key",
	}

	// Mirrors the gRPC handler, which always passes non-nil pointers.
	UpdateViberBM{
		Name:          new(""),
		BaseURL:       new(""),
		SenderName:    new(""),
		APIKey:        new("new-key"),
		WebhookSecret: new(""),
	}.ApplyTo(gate)

	if gate.Name != "IraViberBM" {
		t.Errorf("Name = %q, want unchanged", gate.Name)
	}

	if gate.BaseURL != "https://x18vwq.api.infobip.com" || gate.SenderName != "IBSelfServe" {
		t.Errorf("BaseURL/SenderName changed: %q / %q", gate.BaseURL, gate.SenderName)
	}

	if gate.APIKey != "new-key" {
		t.Errorf("APIKey = %q, want new-key", gate.APIKey)
	}
}

func TestUpdateViberBM_ApplyTo_NonEmptyApplied(t *testing.T) {
	gate := &ViberBMGate{Name: "old", Enabled: false}

	UpdateViberBM{Name: new("new"), Enabled: new(true)}.ApplyTo(gate)

	if gate.Name != "new" || !gate.Enabled {
		t.Errorf("gate = %+v, want name=new enabled=true", gate)
	}
}

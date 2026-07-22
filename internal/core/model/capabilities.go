package model

// ProviderCapabilities declares which delivery-status confirmations a
// channel can produce. Channels without receipts leave messages in SENT —
// an honest state, not a defect; the UI uses these flags to avoid drawing
// unreachable status marks.
type ProviderCapabilities struct {
	// SupportsDelivered is true when the provider emits delivery receipts.
	SupportsDelivered bool
	// SupportsRead is true when the provider emits read/seen receipts.
	SupportsRead bool
	// SupportsFailed is true when the provider reports failed deliveries.
	SupportsFailed bool
}

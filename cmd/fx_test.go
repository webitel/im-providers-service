package cmd

import (
	"testing"

	"go.uber.org/fx"

	"github.com/webitel/im-providers-service/config"
)

// TestAppOptions_GraphResolves type-checks the dependency graph without running
// any constructor. Registering a provider module means adding constructors on
// several layers at once, and a missing or mistyped dependency would otherwise
// only surface at boot.
func TestAppOptions_GraphResolves(t *testing.T) {
	if err := fx.ValidateApp(AppOptions(&config.Config{})); err != nil {
		t.Fatalf("dependency graph does not resolve: %v", err)
	}
}

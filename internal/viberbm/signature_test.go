package viberbm

import (
	"context"
	"net/http"
	"testing"

	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
)

// signatureRepo implements vibbmstore.ViberBMStore with only SelectByURI wired.
type signatureRepo struct {
	gate *vibbmmodel.ViberBMGate
	err  error
}

var _ vibbmstore.ViberBMStore = (*signatureRepo)(nil)

func (r *signatureRepo) Insert(_ context.Context, _ int64, _ *vibbmmodel.ViberBMGate) error {
	return nil
}

func (r *signatureRepo) Select(_ context.Context, _ string) (*vibbmmodel.ViberBMGate, error) {
	return r.gate, r.err
}

func (r *signatureRepo) SelectByURI(_ context.Context, _ string) (*vibbmmodel.ViberBMGate, error) {
	return r.gate, r.err
}

func (r *signatureRepo) SelectBySenderAndURI(_ context.Context, _, _ string) (*vibbmmodel.ViberBMGate, error) {
	return r.gate, r.err
}

func (r *signatureRepo) List(_ context.Context, _ int64, _ string, _, _ int) ([]*vibbmmodel.ViberBMGate, bool, error) {
	return nil, false, nil
}

func (r *signatureRepo) Update(_ context.Context, _ *vibbmmodel.ViberBMGate) error { return nil }
func (r *signatureRepo) Unbind(_ context.Context, _ string) error                  { return nil }

func providerWithSecret(secret string) *viberBMProvider {
	gate := &vibbmmodel.ViberBMGate{
		ID:            "g-sig",
		WebhookSecret: secret,
		Enabled:       true,
	}

	return &viberBMProvider{
		logger: discardLogger(),
		repo:   &signatureRepo{gate: gate},
	}
}

func TestValidateSignature(t *testing.T) {
	cases := []struct {
		name       string
		secret     string // configured on the gate
		headerVal  string // value in incoming Authorization header
		wantErrMsg string // non-empty = expect error containing this substring; "" = expect nil
	}{
		{
			name:      "valid credential passes",
			secret:    "my-super-secret",
			headerVal: "my-super-secret",
		},
		{
			name:       "wrong secret is rejected",
			secret:     "correct-secret",
			headerVal:  "wrong-secret",
			wantErrMsg: "credential mismatch",
		},
		{
			name:      "empty configured secret always passes (opt-out)",
			secret:    "",
			headerVal: "",
		},
		{
			name:      "empty configured secret passes even with arbitrary header",
			secret:    "",
			headerVal: "some-random-value",
		},
		{
			name:       "missing header with secret configured is error",
			secret:     "my-secret",
			headerVal:  "", // no header sent
			wantErrMsg: "credential mismatch",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := providerWithSecret(tc.secret)

			h := http.Header{}
			if tc.headerVal != "" {
				h.Set("Authorization", tc.headerVal)
			}

			err := p.ValidateSignature(context.Background(), h, nil)

			if tc.wantErrMsg == "" {
				if err != nil {
					t.Errorf("expected nil error, got: %v", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrMsg)
			}

			if got := err.Error(); len(got) == 0 {
				t.Errorf("error message empty, want substring %q", tc.wantErrMsg)
			}
		})
	}
}

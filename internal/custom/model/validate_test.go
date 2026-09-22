package model

import (
	"net"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestCreateCustom_Validate(t *testing.T) {
	valid := func() CreateCustom {
		return CreateCustom{
			Name:        "Partner middleware",
			CallbackURL: "https://partner.example.org/hook",
			Peer:        sharedmodel.Peer{Sub: "bot-sub", Iss: "schema"},
		}
	}

	cases := []struct {
		name    string
		mutate  func(*CreateCustom)
		wantErr bool
	}{
		{name: "complete", mutate: func(*CreateCustom) {}},
		{name: "http callback", mutate: func(c *CreateCustom) { c.CallbackURL = "http://on-premise.local/hook" }},
		{name: "allowlist of address and block", mutate: func(c *CreateCustom) {
			c.AllowedIPs = []string{"203.0.113.7", "10.0.0.0/8", " "}
		}},
		{name: "no name", mutate: func(c *CreateCustom) { c.Name = "" }, wantErr: true},
		{name: "no callback", mutate: func(c *CreateCustom) { c.CallbackURL = "" }, wantErr: true},
		{name: "no peer", mutate: func(c *CreateCustom) { c.Peer = sharedmodel.Peer{} }, wantErr: true},
		{name: "relative callback", mutate: func(c *CreateCustom) { c.CallbackURL = "/hook" }, wantErr: true},
		{name: "non-http callback", mutate: func(c *CreateCustom) { c.CallbackURL = "ftp://partner.example.org" }, wantErr: true},
		{name: "unparsable allowlist entry", mutate: func(c *CreateCustom) {
			c.AllowedIPs = []string{"not-an-address"}
		}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := valid()
			tc.mutate(&req)

			err := req.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("want a validation error, got none")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("want no error, got %v", err)
			}
		})
	}
}

func TestParseAllowedIP(t *testing.T) {
	cases := []struct {
		entry    string
		contains string
		excludes string
		wantErr  bool
	}{
		{entry: "203.0.113.7", contains: "203.0.113.7", excludes: "203.0.113.8"},
		{entry: "203.0.113.0/24", contains: "203.0.113.200", excludes: "198.51.100.1"},
		{entry: " 10.0.0.0/8 ", contains: "10.1.2.3", excludes: "11.0.0.1"},
		{entry: "2001:db8::/32", contains: "2001:db8::1", excludes: "2001:dba::1"},
		{entry: "not-an-address", wantErr: true},
		{entry: "203.0.113.0/99", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.entry, func(t *testing.T) {
			network, err := ParseAllowedIP(tc.entry)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want an error, got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseAllowedIP: %v", err)
			}

			if !network.Contains(parseIP(t, tc.contains)) {
				t.Errorf("%s should contain %s", tc.entry, tc.contains)
			}

			if network.Contains(parseIP(t, tc.excludes)) {
				t.Errorf("%s should not contain %s", tc.entry, tc.excludes)
			}
		})
	}
}

func TestUpdateCustom_Validate(t *testing.T) {
	str := func(v string) *string { return &v }
	num := func(v int32) *int32 { return &v }

	cases := []struct {
		name    string
		upd     UpdateCustom
		wantErr bool
	}{
		{name: "nothing to change", upd: UpdateCustom{}},
		{name: "rotated secret", upd: UpdateCustom{AppSecret: str("next")}},
		{name: "callback replaced", upd: UpdateCustom{CallbackURL: str("https://partner.example.org/hook")}},
		{name: "empty secret", upd: UpdateCustom{AppSecret: str("")}, wantErr: true},
		{name: "empty callback", upd: UpdateCustom{CallbackURL: str("")}, wantErr: true},
		{name: "relative callback", upd: UpdateCustom{CallbackURL: str("/hook")}, wantErr: true},
		{name: "timeout below the accepted range", upd: UpdateCustom{RequestTimeoutMS: num(1)}, wantErr: true},
		{name: "timeout above the accepted range", upd: UpdateCustom{RequestTimeoutMS: num(MaxRequestTimeoutMS + 1)}, wantErr: true},
		{name: "retries above the accepted range", upd: UpdateCustom{RetryAttempts: num(MaxRetryAttempts + 1)}, wantErr: true},
		{name: "retries disabled", upd: UpdateCustom{RetryAttempts: num(0)}},
		{name: "unparsable allowlist entry", upd: UpdateCustom{AllowedIPs: &[]string{"not-an-address"}}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.upd.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("want a validation error, got none")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("want no error, got %v", err)
			}
		})
	}
}

func TestValidate_ReportsInvalidArgument(t *testing.T) {
	cases := []struct {
		name string
		err  error
		id   string
	}{
		{
			name: "missing required fields",
			err:  CreateCustom{}.Validate(),
			id:   "custom.model.create_custom.validate",
		},
		{
			name: "bad field value",
			err: CreateCustom{
				Name:        "n",
				CallbackURL: "ftp://x",
				Peer:        sharedmodel.Peer{Sub: "s", Iss: "i"},
			}.Validate(),
			id: "custom.model.validate_callback_url",
		},
		{
			name: "bad allowlist entry",
			err:  UpdateCustom{AllowedIPs: &[]string{"not-an-address"}}.Validate(),
			id:   "custom.model.validate_allowed_ips",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Code(tc.err); got != codes.InvalidArgument {
				t.Errorf("grpc code = %s, want InvalidArgument", got)
			}

			if got := errors.ID(tc.err); got != tc.id {
				t.Errorf("error id = %q, want %q", got, tc.id)
			}
		})
	}
}

func parseIP(t *testing.T, raw string) net.IP {
	t.Helper()

	ip := net.ParseIP(raw)
	if ip == nil {
		t.Fatalf("test fixture %q is not an ip address", raw)
	}

	return ip
}

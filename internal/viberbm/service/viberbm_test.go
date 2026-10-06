package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
)

type fakeStore struct {
	vibbmstore.ViberBMStore

	gate    *vibbmmodel.ViberBMGate
	updated bool
	unbound bool
}

func (s *fakeStore) Select(_ context.Context, id string) (*vibbmmodel.ViberBMGate, error) {
	if s.gate == nil || s.gate.ID != id {
		return nil, sharedstore.ErrNotFound
	}

	g := *s.gate

	return &g, nil
}

func (s *fakeStore) Update(context.Context, *vibbmmodel.ViberBMGate) error {
	s.updated = true

	return nil
}

func (s *fakeStore) Unbind(context.Context, string) error {
	s.unbound = true

	return nil
}

type fakeSender struct {
	calls int
}

func (f *fakeSender) SendTemplate(context.Context, string, string, string, string, map[string]string) (*sharedmodel.MessageResponse, error) {
	f.calls++

	return &sharedmodel.MessageResponse{ID: "m1"}, nil
}

const (
	ownDomain   int64 = 1
	otherDomain int64 = 2
)

func newService() (*ViberBMService, *fakeStore, *fakeSender) {
	st := &fakeStore{gate: &vibbmmodel.ViberBMGate{ID: "g1", DomainID: ownDomain, BaseURL: "https://x.api.infobip.com"}}
	snd := &fakeSender{}

	return NewViberBMService(st, snd, slog.New(slog.NewTextHandler(io.Discard, nil))), st, snd
}

func TestGateAccessIsScopedToDomain(t *testing.T) {
	ctx := context.Background()

	t.Run("get", func(t *testing.T) {
		s, _, _ := newService()

		if _, err := s.GetGate(ctx, otherDomain, "g1"); !errors.Is(err, sharedstore.ErrNotFound) {
			t.Fatalf("foreign get err = %v, want ErrNotFound", err)
		}

		if g, err := s.GetGate(ctx, ownDomain, "g1"); err != nil || g.ID != "g1" {
			t.Fatalf("own get = %v, %v", g, err)
		}
	})

	t.Run("update", func(t *testing.T) {
		s, st, _ := newService()
		name := "renamed"

		if _, err := s.UpdateGate(ctx, otherDomain, vibbmmodel.UpdateViberBM{ID: "g1", Name: &name}); !errors.Is(err, sharedstore.ErrNotFound) || st.updated {
			t.Fatalf("foreign update err = %v, updated = %v", err, st.updated)
		}

		if _, err := s.UpdateGate(ctx, ownDomain, vibbmmodel.UpdateViberBM{ID: "g1", Name: &name}); err != nil || !st.updated {
			t.Fatalf("own update err = %v, updated = %v", err, st.updated)
		}
	})

	t.Run("delete", func(t *testing.T) {
		s, st, _ := newService()

		if _, err := s.DeleteGate(ctx, otherDomain, "g1"); !errors.Is(err, sharedstore.ErrNotFound) || st.unbound {
			t.Fatalf("foreign delete err = %v, unbound = %v", err, st.unbound)
		}

		if _, err := s.DeleteGate(ctx, ownDomain, "g1"); err != nil || !st.unbound {
			t.Fatalf("own delete err = %v, unbound = %v", err, st.unbound)
		}
	})

	t.Run("send template", func(t *testing.T) {
		s, _, snd := newService()

		if _, err := s.SendTemplate(ctx, otherDomain, "g1", "380", "t", "uk", nil); !errors.Is(err, sharedstore.ErrNotFound) || snd.calls != 0 {
			t.Fatalf("foreign send err = %v, calls = %d", err, snd.calls)
		}

		if _, err := s.SendTemplate(ctx, ownDomain, "g1", "380", "t", "uk", nil); err != nil || snd.calls != 1 {
			t.Fatalf("own send err = %v, calls = %d", err, snd.calls)
		}
	})
}

func TestUpdateRejectsNonHTTPSBaseURL(t *testing.T) {
	s, st, _ := newService()
	bad := "http://169.254.169.254"

	_, err := s.UpdateGate(context.Background(), ownDomain, vibbmmodel.UpdateViberBM{ID: "g1", BaseURL: &bad})

	var ve *vibbmmodel.ValidationError
	if !errors.As(err, &ve) || st.updated {
		t.Fatalf("err = %v, updated = %v; want validation error and no write", err, st.updated)
	}
}

package via

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestAppendToOutgoingContext(t *testing.T) {
	tests := []struct {
		name string
		gate Gate
		want metadata.MD
	}{
		{
			name: "full gate",
			gate: Gate{ID: "g1", Type: "facebook", Name: "Підтримка"},
			want: metadata.MD{
				HeaderID:   {"g1"},
				HeaderType: {"facebook"},
				HeaderName: {"Підтримка"},
			},
		},
		{
			name: "id only",
			gate: Gate{ID: "g1"},
			want: metadata.MD{HeaderID: {"g1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "x-webitel-type", "provider")
			md, _ := metadata.FromOutgoingContext(AppendToOutgoingContext(ctx, tt.gate))

			if got := md.Get("x-webitel-type"); len(got) != 1 || got[0] != "provider" {
				t.Fatalf("existing metadata lost: %v", md)
			}

			for k, v := range tt.want {
				if got := md.Get(k); len(got) != 1 || got[0] != v[0] {
					t.Errorf("%s = %v, want %v", k, got, v)
				}
			}

			if tt.gate.Name == "" && len(md.Get(HeaderName)) != 0 {
				t.Errorf("%s must be omitted for an empty name", HeaderName)
			}
		})
	}
}

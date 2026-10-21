package store

import (
	"reflect"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestBuildListQuery(t *testing.T) {
	tests := []struct {
		name      string
		filter    sharedmodel.ListFilter
		wantQuery string
		wantArgs  []any
	}{
		{
			name:      "domain only",
			filter:    sharedmodel.ListFilter{DomainID: 1},
			wantQuery: `SELECT * FROM im_provider.gate_summary WHERE dc = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			wantArgs:  []any{int64(1), 21, 0},
		},
		{
			name: "all filters",
			filter: sharedmodel.ListFilter{
				DomainID: 1,
				Types:    []sharedmodel.GateType{sharedmodel.TypeWhatsApp, sharedmodel.TypeViberBM},
				Status:   sharedmodel.StatusActive,
				Q:        "shop",
			},
			wantQuery: `SELECT * FROM im_provider.gate_summary WHERE dc = $1 AND type = ANY($2) AND status = $3 AND name ILIKE $4 ORDER BY created_at DESC LIMIT $5 OFFSET $6`,
			wantArgs:  []any{int64(1), []string{"whatsapp", "viber_bm"}, "active", "%shop%", 21, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildListQuery(tt.filter, 21, 0)
			if query != tt.wantQuery {
				t.Errorf("query =\n%s\nwant\n%s", query, tt.wantQuery)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %#v, want %#v", args, tt.wantArgs)
			}
		})
	}
}

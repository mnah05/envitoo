package service

import (
	"testing"
	"time"

	"github.com/mnah05/envitoo/internal/dto"
)

func TestHasDisallowedStaffUpdateFields(t *testing.T) {
	now := time.Now()
	cap := int32(10)
	price := int64(100)
	currency := "INR"
	reqApproval := true
	virtual := true
	category := "technical"

	tests := []struct {
		name string
		req  dto.UpdateEventRequest
		want bool
	}{
		{
			name: "allowed fields only",
			req: dto.UpdateEventRequest{
				Description:   new("updated"),
				MeetingURL:    new("https://meet.example.com"),
				CoverImageURL: new("https://img.example.com/a.png"),
				VenueName:     new("Hall A"),
			},
			want: false,
		},
		{
			name: "disallowed capacity",
			req:  dto.UpdateEventRequest{Capacity: &cap},
			want: true,
		},
		{
			name: "disallowed starts_at",
			req:  dto.UpdateEventRequest{StartsAt: &now},
			want: true,
		},
		{
			name: "disallowed requires_approval",
			req:  dto.UpdateEventRequest{RequiresApproval: &reqApproval},
			want: true,
		},
		{
			name: "disallowed is_virtual",
			req:  dto.UpdateEventRequest{IsVirtual: &virtual},
			want: true,
		},
		{
			name: "disallowed category",
			req:  dto.UpdateEventRequest{Category: &category},
			want: true,
		},
		{
			name: "disallowed price",
			req:  dto.UpdateEventRequest{PriceInPaise: &price, Currency: &currency},
			want: true,
		},
		{
			name: "disallowed visibility",
			req:  dto.UpdateEventRequest{Visibility: new("unlisted")},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasDisallowedStaffUpdateFields(tt.req); got != tt.want {
				t.Fatalf("HasDisallowedStaffUpdateFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

//go:fix inline
func strPtr(s string) *string { return new(s) }

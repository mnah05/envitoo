package service

import "testing"

func TestValidateCapacityForApproval(t *testing.T) {
	tests := []struct {
		name       string
		count      int32
		capacity   int32
		fromStatus string
		toStatus   string
		wantErr    bool
	}{
		{name: "pending to approved at capacity", count: 10, capacity: 10, fromStatus: "pending", toStatus: "approved"},
		{name: "pending to approved over capacity", count: 11, capacity: 10, fromStatus: "pending", toStatus: "approved", wantErr: true},
		{name: "rejected to approved at capacity", count: 10, capacity: 10, fromStatus: "rejected", toStatus: "approved", wantErr: true},
		{name: "cancelled to approved with room", count: 9, capacity: 10, fromStatus: "cancelled", toStatus: "approved"},
		{name: "approved to approved noop", count: 10, capacity: 10, fromStatus: "approved", toStatus: "approved"},
		{name: "pending to rejected", count: 10, capacity: 10, fromStatus: "pending", toStatus: "rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCapacityForApproval(tt.count, tt.capacity, tt.fromStatus, tt.toStatus)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateCapacityForApproval() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistrationStatusTransitionAllowed(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{"pending", "approved", true},
		{"pending", "rejected", true},
		{"pending", "cancelled", true},
		{"approved", "cancelled", true},
		{"approved", "rejected", true},
		{"approved", "pending", false},
		{"rejected", "approved", true},
		{"rejected", "pending", false},
		{"cancelled", "approved", false},
		{"cancelled", "pending", false},
		{"pending", "pending", true},
		{"unknown", "approved", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			got := registrationStatusTransitionAllowed(tt.from, tt.to)
			if got != tt.want {
				t.Fatalf("registrationStatusTransitionAllowed(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestRegistrationCreatedEmailTemplate(t *testing.T) {
	tests := []struct {
		status       string
		wantTemplate string
		wantTopic    string
	}{
		{"pending", EmailTemplateTicketStatusChange, EmailTopicTicketStatusChange},
		{"approved", EmailTemplateTicketConfirm, EmailTopicTicketConfirm},
		{"rejected", EmailTemplateTicketConfirm, EmailTopicTicketConfirm}, // non-pending create path uses confirm
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			gotTemplate, gotTopic := registrationCreatedEmailTemplate(tt.status)
			if gotTemplate != tt.wantTemplate || gotTopic != tt.wantTopic {
				t.Fatalf("registrationCreatedEmailTemplate(%q) = (%q, %q), want (%q, %q)",
					tt.status, gotTemplate, gotTopic, tt.wantTemplate, tt.wantTopic)
			}
		})
	}
}

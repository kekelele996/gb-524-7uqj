package service

import (
	"testing"

	"spectrum-interference-triangulation/backend/internal/constants"
)

func TestAuthorizeCaseTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    constants.CaseStatus
		to      constants.CaseStatus
		role    string
		allowed bool
	}{
		{"observer starts collection", constants.CaseDraft, constants.CaseCollecting, constants.RoleObserver, true},
		{"observer cannot analyze", constants.CaseCollecting, constants.CaseAnalyzing, constants.RoleObserver, false},
		{"analyst submits review", constants.CaseAnalyzing, constants.CasePendingReview, constants.RoleAnalyst, true},
		{"analyst cannot confirm", constants.CasePendingReview, constants.CaseConfirmed, constants.RoleAnalyst, false},
		{"reviewer confirms", constants.CasePendingReview, constants.CaseConfirmed, constants.RoleReviewer, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizeTransition(tt.from, tt.to, tt.role)
			if tt.allowed && err != nil {
				t.Fatalf("expected allowed, got %v", err)
			}
			if !tt.allowed && err == nil {
				t.Fatal("expected transition to be denied")
			}
		})
	}
}

package ghclient

import (
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestCIFromState(t *testing.T) {
	tests := []struct {
		in   string
		want model.CIState
	}{
		{"SUCCESS", model.CISuccess},
		{"FAILURE", model.CIFailure},
		{"ERROR", model.CIFailure},
		{"PENDING", model.CIPending},
		{"EXPECTED", model.CIPending},
		{"", model.CINone},
		{"WAT", model.CINone},
	}
	for _, tt := range tests {
		if got := ciFromState(tt.in); got != tt.want {
			t.Errorf("ciFromState(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCIFromRollup(t *testing.T) {
	if got := ciFromRollup(nil); got != model.CINone {
		t.Errorf("ciFromRollup(nil) = %v, want none", got)
	}
	r := &struct {
		State string `json:"state"`
	}{State: "SUCCESS"}
	if got := ciFromRollup(r); got != model.CISuccess {
		t.Errorf("ciFromRollup(SUCCESS) = %v, want success", got)
	}
}

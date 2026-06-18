package model

import "testing"

func TestComputeHealth(t *testing.T) {
	tests := []struct {
		name string
		repo Repo
		want Health
	}{
		{"archived is gray", Repo{Archived: true, CI: CIFailure}, HealthGray},
		{"ci failure is red", Repo{CI: CIFailure}, HealthRed},
		{"critical is red", Repo{CI: CISuccess, Dependabot: AlertCounts{Critical: 1}}, HealthRed},
		{"high is red", Repo{CI: CISuccess, Dependabot: AlertCounts{High: 2}}, HealthRed},
		{"ci pending is yellow", Repo{CI: CIPending}, HealthYellow},
		{"undeployed is yellow", Repo{CI: CISuccess, LatestTag: "v1", Undeployed: -1}, HealthYellow},
		{"moderate is yellow", Repo{CI: CISuccess, Dependabot: AlertCounts{Moderate: 1}}, HealthYellow},
		{"secret scan is yellow", Repo{CI: CISuccess, SecretScanning: 1}, HealthYellow},
		{"clean deployed is green", Repo{CI: CISuccess, LatestTag: "v1", Undeployed: 0}, HealthGreen},
		{"untagged clean is green", Repo{CI: CISuccess, Untagged: true}, HealthGreen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeHealth(tt.repo); got != tt.want {
				t.Errorf("ComputeHealth = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUndeployedLabel(t *testing.T) {
	cases := map[string]Repo{
		"untagged": {Untagged: true},
		">=1":      {Undeployed: -1},
		"-":        {Undeployed: 0},
		"3":        {Undeployed: 3},
	}
	for want, r := range cases {
		if got := r.UndeployedLabel(); got != want {
			t.Errorf("UndeployedLabel(%+v) = %q, want %q", r, got, want)
		}
	}
}

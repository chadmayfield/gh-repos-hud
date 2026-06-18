package ghclient

import "testing"

func TestSemverSelection(t *testing.T) {
	// Highest semver wins regardless of input order (the bug was picking the
	// newest-by-date tag, e.g. v0.1.2, over a higher v0.5.0).
	tags := []string{"v0.1.2", "v0.5.0", "mvp-1", "v0.1.10", "v0.5.0-rc1"}
	var best [3]int
	var bestName string
	for _, name := range tags {
		if k, ok := semverKey(name); ok {
			if bestName == "" || semverGreater(k, best) {
				best, bestName = k, name
			}
		}
	}
	if bestName != "v0.5.0" && bestName != "v0.5.0-rc1" {
		t.Errorf("highest = %q, want v0.5.0(-rc1)", bestName)
	}
	if best != [3]int{0, 5, 0} {
		t.Errorf("best key = %v, want [0 5 0]", best)
	}

	// v0.1.10 > v0.1.2 (numeric, not lexical)
	a, _ := semverKey("v0.1.10")
	b, _ := semverKey("v0.1.2")
	if !semverGreater(a, b) {
		t.Errorf("v0.1.10 should be > v0.1.2")
	}
	if _, ok := semverKey("mvp-1"); ok {
		t.Errorf("mvp-1 should not parse as semver")
	}
}

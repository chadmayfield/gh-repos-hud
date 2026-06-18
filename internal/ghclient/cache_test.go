package ghclient

import (
	"os"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestCacheRoundTrip(t *testing.T) {
	p, err := cachePath()
	if err != nil {
		t.Skip("no user cache dir")
	}
	_ = os.Remove(p)
	t.Cleanup(func() { _ = os.Remove(p) })

	// loadCache on a missing file -> miss.
	if _, ok := loadCache(time.Minute); ok {
		t.Fatal("expected cache miss on missing file")
	}

	st := &model.State{
		FetchedAt: time.Now(),
		Owners: []model.Owner{{Name: "org", Repos: []model.Repo{
			// CI/Health are json:"-"; gob must still preserve them.
			{Name: "r", CI: model.CISuccess, Health: model.HealthRed},
		}}},
	}
	saveCache(st)

	got, ok := loadCache(time.Minute)
	if !ok {
		t.Fatal("expected cache hit after save")
	}
	if !got.FromCache {
		t.Error("FromCache should be true on a cached load")
	}
	r := got.Owners[0].Repos[0]
	if r.CI != model.CISuccess || r.Health != model.HealthRed {
		t.Errorf("enums not preserved through gob: CI=%v Health=%v", r.CI, r.Health)
	}

	// Expired TTL -> miss.
	if _, ok := loadCache(time.Nanosecond); ok {
		t.Error("expected miss for an expired TTL")
	}
}

package ghclient

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// Cache is a single-file, TTL-bounded snapshot of the last fetch. It lets
// repeated runs (and --watch ticks) reuse data instead of re-querying the
// GraphQL API, which is point-metered and easily exhausted. gob is used (not
// JSON) so the computed CI/Health enums — which are json:"-" for the public
// --json output — still round-trip.

func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh-repos-hud", "state.gob"), nil
}

// cacheVersion is bumped whenever the cached model shape changes, so a binary
// never decodes an incompatible older cache (which could yield partial data).
const cacheVersion = 4

type cacheEnvelope struct {
	Version int
	State   model.State
}

// loadCache returns the cached state if it exists, is younger than ttl, and was
// written by a matching cache version.
func loadCache(ttl time.Duration) (*model.State, bool) {
	p, err := cachePath()
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(p)
	if err != nil || time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	var env cacheEnvelope
	if err := gob.NewDecoder(f).Decode(&env); err != nil || env.Version != cacheVersion {
		return nil, false
	}
	env.State.FromCache = true
	return &env.State, true
}

// saveCache best-effort writes the snapshot; errors are non-fatal.
func saveCache(st *model.State) {
	p, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	f, err := os.Create(p)
	if err != nil {
		return
	}
	defer f.Close()
	_ = gob.NewEncoder(f).Encode(cacheEnvelope{Version: cacheVersion, State: *st})
}

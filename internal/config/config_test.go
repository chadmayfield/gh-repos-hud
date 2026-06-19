package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
	p := filepath.ToSlash(Path())
	if !strings.HasSuffix(p, ".config/gh-repos-hud/config.yml") {
		t.Errorf("Path()=%q", p)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no config file present -> defaults apply
	c := Load()
	if !c.IncludePersonal || !c.ExcludeArchived {
		t.Errorf("default bools wrong: %+v", c)
	}
	if c.RefreshSeconds != 30 || c.CacheTTLSeconds != 300 || c.Port != 8787 {
		t.Errorf("default numbers wrong: %+v", c)
	}
}

func TestLoadFromFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "gh-repos-hud")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "include_personal: false\n" +
		"exclude_archived: false\n" +
		"refresh: 15\n" +
		"cache_ttl: 60\n" +
		"port: 9999\n" +
		"orgs:\n" +
		"  include: [foo, bar]\n" +
		"  exclude: [baz]\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Load()
	if c.IncludePersonal || c.ExcludeArchived {
		t.Errorf("bools not overridden by file: %+v", c)
	}
	if c.RefreshSeconds != 15 || c.CacheTTLSeconds != 60 || c.Port != 9999 {
		t.Errorf("numbers not overridden by file: %+v", c)
	}
	if len(c.IncludeOrgs) != 2 || len(c.ExcludeOrgs) != 1 {
		t.Errorf("orgs not parsed: %+v", c)
	}
}

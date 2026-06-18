// Package config loads optional user settings from
// ~/.config/gh-repos-hud/config.yml. Every field has a default; the file is
// optional. Command-line flags override these at the call site.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config is the merged settings (file + defaults).
type Config struct {
	IncludeOrgs     []string
	ExcludeOrgs     []string
	IncludePersonal bool
	ExcludeArchived bool
	RefreshSeconds  int
	CacheTTLSeconds int
	Port            int
}

// Path returns the config file location (whether or not it exists).
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gh-repos-hud", "config.yml")
}

// Load reads the config file if present, applying defaults. A missing file is
// not an error. Env vars prefixed HUD_ override (e.g. HUD_PORT).
func Load() Config {
	v := viper.New()
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "gh-repos-hud"))
	}
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetEnvPrefix("HUD")
	v.AutomaticEnv()

	v.SetDefault("include_personal", true)
	v.SetDefault("exclude_archived", true)
	v.SetDefault("refresh", 30)
	v.SetDefault("cache_ttl", 300)
	v.SetDefault("port", 8787)

	_ = v.ReadInConfig() // missing file is fine

	return Config{
		IncludeOrgs:     v.GetStringSlice("orgs.include"),
		ExcludeOrgs:     v.GetStringSlice("orgs.exclude"),
		IncludePersonal: v.GetBool("include_personal"),
		ExcludeArchived: v.GetBool("exclude_archived"),
		RefreshSeconds:  v.GetInt("refresh"),
		CacheTTLSeconds: v.GetInt("cache_ttl"),
		Port:            v.GetInt("port"),
	}
}

// Package model holds the front-end-agnostic data the HUD renders. The
// ghclient package populates it; the tui, web, and JSON outputs consume it.
package model

import "time"

// Health is the single rollup status per repo.
type Health int

const (
	HealthGray   Health = iota // unknown / degraded / empty
	HealthGreen                // all good
	HealthYellow               // needs a look (undeployed, moderate alerts, CI in-progress)
	HealthRed                  // CI failing or open critical/high alert
)

// Glyph returns an ASCII status marker (no emoji, per project convention).
func (h Health) Glyph() string {
	switch h {
	case HealthGreen:
		return "[OK]"
	case HealthYellow:
		return "[~]"
	case HealthRed:
		return "[!!]"
	default:
		return "[?]"
	}
}

func (h Health) String() string {
	switch h {
	case HealthGreen:
		return "green"
	case HealthYellow:
		return "yellow"
	case HealthRed:
		return "red"
	default:
		return "gray"
	}
}

// CIState is the status of the latest run on the default branch.
type CIState int

const (
	CINone CIState = iota // no checks / no commit
	CISuccess
	CIPending
	CIFailure
)

func (c CIState) String() string {
	switch c {
	case CISuccess:
		return "success"
	case CIPending:
		return "pending"
	case CIFailure:
		return "failure"
	default:
		return "none"
	}
}

// Short returns a fixed-width-ish label for tables.
func (c CIState) Short() string {
	switch c {
	case CISuccess:
		return "OK"
	case CIPending:
		return "..."
	case CIFailure:
		return "FAIL"
	default:
		return "-"
	}
}

// ScanState is the per-repo enablement of a scanning product (code-scanning or
// secret-scanning). It is a tri-state because "0 open alerts" is ambiguous on
// its own: a repo can have scanning ON and clean, or simply OFF. Only a
// per-repo API probe distinguishes them (200 = on, 403/404 = off).
type ScanState int

const (
	ScanUnknown ScanState = iota // couldn't determine (network/5xx/rate-limit) -> "?"
	ScanOff                      // not enabled / no analysis / no access      -> "off"
	ScanOn                       // enabled; the alert count is meaningful      -> "N"
)

func (s ScanState) String() string {
	switch s {
	case ScanOn:
		return "on"
	case ScanOff:
		return "off"
	default:
		return "unknown"
	}
}

// Cell renders the scan column: the open-alert count when on, "off" when not
// enabled, and "?" when undetermined. Shared by the TUI, web, and --plain
// renderers so the three never drift.
func (s ScanState) Cell(openAlerts int) string {
	switch s {
	case ScanOn:
		return itoa(openAlerts)
	case ScanOff:
		return "off"
	default:
		return "?"
	}
}

// AlertCounts is a severity breakdown of open alerts.
type AlertCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Moderate int `json:"moderate"`
	Low      int `json:"low"`
}

func (a AlertCounts) Total() int { return a.Critical + a.High + a.Moderate + a.Low }

// PRStats summarizes open pull requests.
type PRStats struct {
	Total     int `json:"total"`
	Bot       int `json:"bot"`
	Human     int `json:"human"`
	Mergeable int `json:"mergeable"`
	CIGreen   int `json:"ci_green"`
	Draft     int `json:"draft"`
}

// Repo is one repository's health snapshot.
type Repo struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
	ShortSHA      string `json:"short_sha"`

	CI     CIState `json:"-"`
	CIName string  `json:"ci"`

	Dependabot        AlertCounts `json:"dependabot"`
	DependabotEnabled bool        `json:"dependabot_enabled"`

	// Code/secret scanning: a tri-state enablement plus the open-alert count.
	// The count is only meaningful when the corresponding ScanState is ScanOn.
	CodeScanning   int       `json:"code_scanning"`
	CodeScan       ScanState `json:"-"`
	CodeScanName   string    `json:"code_scanning_status"` // on / off / unknown
	SecretScanning int       `json:"secret_scanning"`
	SecretScan     ScanState `json:"-"`
	SecretScanName string    `json:"secret_scanning_status"`

	LatestTag     string `json:"latest_tag"`
	LatestRelease string `json:"latest_release"`
	// Untagged: no version tag at all. Undeployed: commits on default branch
	// since the latest tag. -1 means ">=1 but exact count not yet resolved".
	Untagged   bool `json:"untagged"`
	Undeployed int  `json:"undeployed"`

	PRs PRStats `json:"prs"`

	// TagSHA/HeadSHA retained so the TUI can resolve exact ahead_by lazily.
	TagSHA  string `json:"-"`
	HeadSHA string `json:"-"`

	Health     Health   `json:"-"`
	HealthName string   `json:"health"`
	Warnings   []string `json:"warnings,omitempty"`
}

// AlertDetail is one open Dependabot alert, for the drill-in pane.
type AlertDetail struct {
	Package  string `json:"package"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
}

// PRDetail is one open pull request, for the drill-in pane.
type PRDetail struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Draft  bool   `json:"draft"`
}

// RepoDetail is lazily fetched when a repo is drilled into.
type RepoDetail struct {
	AheadBy    int  // exact commits since the latest tag
	AheadKnown bool // false until the compare call resolves
	Alerts     []AlertDetail
	PRs        []PRDetail
}

// Billing is org-level GHAS spend (the paid-features guard).
type Billing struct {
	Known                      bool `json:"known"`
	SecretProtectionCommitters int  `json:"secret_protection_committers"`
	CodeSecurityCommitters     int  `json:"code_security_committers"`
}

// Owner is an org (or the personal account) with its repos.
type Owner struct {
	Name    string  `json:"name"`
	IsUser  bool    `json:"is_user"`
	Billing Billing `json:"billing"`
	Repos   []Repo  `json:"repos"`
}

// RateLimit captures remaining API headroom for the footer.
type RateLimit struct {
	RESTRemaining    int       `json:"rest_remaining"`
	RESTLimit        int       `json:"rest_limit"`
	GraphQLRemaining int       `json:"graphql_remaining"`
	GraphQLLimit     int       `json:"graphql_limit"`
	GraphQLCost      int       `json:"graphql_cost"` // points this fetch spent
	ResetAt          time.Time `json:"reset_at"`
}

// Warning records a degraded feature (e.g. secret scanning disabled, or the
// token lacks a scope) without aborting the whole run.
type Warning struct {
	Owner   string `json:"owner"`
	Feature string `json:"feature"`
	Reason  string `json:"reason"`
}

// State is the full HUD snapshot.
type State struct {
	Owners    []Owner   `json:"owners"`
	FetchedAt time.Time `json:"fetched_at"`
	RateLimit RateLimit `json:"rate_limit"`
	Warnings  []Warning `json:"warnings,omitempty"`
	FromCache bool      `json:"from_cache,omitempty"`
}

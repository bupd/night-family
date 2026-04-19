// Package family defines the night-family member type, the canonical
// YAML serialisation, and an in-memory store loaded from a directory of
// YAML files.
//
// A member is a prompt-configurable persona (role + system prompt) with
// zero or more duties bound to it. The store is the daemon's
// authoritative view of the family at runtime; new files dropped into
// the config directory are picked up on SIGHUP (wired in a follow-up).
package family

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Risk classifies how much autonomy a member has.
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// Cost hints at how many tokens a member's run typically consumes.
type Cost string

const (
	CostLow    Cost = "low"
	CostMedium Cost = "medium"
	CostHigh   Cost = "high"
)

// Priority orders duties within a member's plan.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Provider identifies which LLM provider should power a member's runs.
type Provider struct {
	Name  string `yaml:"name"  json:"name"`
	Model string `yaml:"model,omitempty" json:"model,omitempty"`
}

// Duty is a binding between a member and a duty type, plus per-binding
// options like interval/priority/args.
type Duty struct {
	Type     string         `yaml:"type"     json:"type"`
	Interval string         `yaml:"interval" json:"interval"`
	Priority Priority       `yaml:"priority,omitempty" json:"priority,omitempty"`
	Args     map[string]any `yaml:"args,omitempty"     json:"args,omitempty"`
}

// Member is the canonical family-member shape.
type Member struct {
	Name           string    `yaml:"name"          json:"name"`
	Role           string    `yaml:"role"          json:"role"`
	SystemPrompt   string    `yaml:"system_prompt" json:"system_prompt"`
	Duties         []Duty    `yaml:"duties,omitempty"            json:"duties,omitempty"`
	RiskTolerance  Risk      `yaml:"risk_tolerance,omitempty"    json:"risk_tolerance,omitempty"`
	MaxPRsPerNight int       `yaml:"max_prs_per_night,omitempty" json:"max_prs_per_night,omitempty"`
	CostTier       Cost      `yaml:"cost_tier,omitempty"         json:"cost_tier,omitempty"`
	Reviewers      []string  `yaml:"reviewers,omitempty"         json:"reviewers,omitempty"`
	Provider       *Provider `yaml:"provider,omitempty"          json:"provider,omitempty"`

	// Timestamps are populated by the store, not the YAML file.
	CreatedAt time.Time `yaml:"-" json:"created_at,omitempty"`
	UpdatedAt time.Time `yaml:"-" json:"updated_at,omitempty"`
}

// defaultsFS bundles the seed roster so a fresh install has something to
// run with.
//
//go:embed all:defaults
var defaultsFS embed.FS

// DefaultsFS returns the embedded default roster filesystem rooted at
// "defaults/*.yaml". Callers use fs.ReadDir to enumerate.
func DefaultsFS() fs.FS {
	sub, _ := fs.Sub(defaultsFS, "defaults")
	return sub
}

// LoadDefaults parses the embedded seed roster. Returned slice is sorted
// by name for deterministic ordering.
func LoadDefaults() ([]Member, error) {
	return loadDir(DefaultsFS())
}

// LoadDir walks a directory of *.yaml files and returns each parsed as
// a Member. Invalid files are skipped; their errors are aggregated and
// returned alongside the valid set so callers can log them without
// failing the whole load.
func LoadDir(dir fs.FS) ([]Member, []error) {
	members, err := loadDir(dir)
	if err == nil {
		return members, nil
	}
	// loadDir returns a collected error only when there was a walk error.
	return members, []error{err}
}

// LoadDiskDir is LoadDir against a path on the real filesystem. If the
// path doesn't exist, it returns (nil, nil) — a missing config dir is
// a normal state, not an error.
func LoadDiskDir(path string) ([]Member, []error) {
	info, err := osStat(path)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	return LoadDir(osDirFS(path))
}

// loadDir is the internal helper shared by Load* variants.
func loadDir(dir fs.FS) ([]Member, error) {
	entries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !(strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
			continue
		}
		m, err := parseFile(dir, name)
		if err != nil {
			return out, fmt.Errorf("parse %s: %w", name, err)
		}
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseFile(dir fs.FS, name string) (Member, error) {
	raw, err := fs.ReadFile(dir, name)
	if err != nil {
		return Member{}, err
	}
	var m Member
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return Member{}, err
	}
	m.applyDefaults()
	if issues := Validate(m); len(issues) > 0 {
		return Member{}, issues
	}
	return m, nil
}

// ApplyDefaults fills in documented defaults for fields the caller
// omitted. Exported so API handlers can mirror the Store's validation
// contract before calling Validate.
func (m *Member) ApplyDefaults() { m.applyDefaults() }

// applyDefaults fills in documented defaults for fields the YAML omitted.
func (m *Member) applyDefaults() {
	if m.RiskTolerance == "" {
		m.RiskTolerance = RiskMedium
	}
	if m.CostTier == "" {
		m.CostTier = CostMedium
	}
	if m.MaxPRsPerNight == 0 {
		m.MaxPRsPerNight = 2
	}
	for i := range m.Duties {
		if m.Duties[i].Priority == "" {
			m.Duties[i].Priority = PriorityMedium
		}
	}
}

// ValidationError is a list of field-level problems with a member. It
// implements error so callers can decide to surface the whole list or
// short-circuit on first.
type ValidationError []FieldIssue

// FieldIssue points at a specific bad field inside a member.
type FieldIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Error returns a human-readable summary of all issues.
func (ve ValidationError) Error() string {
	if len(ve) == 0 {
		return "<no issues>"
	}
	parts := make([]string, 0, len(ve))
	for _, i := range ve {
		parts = append(parts, i.Path+": "+i.Message)
	}
	return strings.Join(parts, "; ")
}

// nameRe enforces the same pattern as openapi.yaml and
// family-member.schema.json.
var (
	nameRe     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	durationRe = regexp.MustCompile(`^[0-9]+(ns|us|µs|ms|s|m|h)$`)
)

// Validate returns every rule violation found in m. An empty slice means
// the member is valid.
func Validate(m Member) ValidationError {
	var issues ValidationError
	if !nameRe.MatchString(m.Name) {
		issues = append(issues, FieldIssue{
			Path:    "name",
			Message: `must match ^[a-z][a-z0-9-]{0,62}$`,
		})
	}
	if strings.TrimSpace(m.Role) == "" {
		issues = append(issues, FieldIssue{Path: "role", Message: "must not be empty"})
	}
	if strings.TrimSpace(m.SystemPrompt) == "" {
		issues = append(issues, FieldIssue{
			Path:    "system_prompt",
			Message: "must not be empty",
		})
	}
	switch m.RiskTolerance {
	case RiskLow, RiskMedium, RiskHigh:
	default:
		issues = append(issues, FieldIssue{
			Path:    "risk_tolerance",
			Message: `must be one of low|medium|high`,
		})
	}
	switch m.CostTier {
	case CostLow, CostMedium, CostHigh:
	default:
		issues = append(issues, FieldIssue{
			Path:    "cost_tier",
			Message: `must be one of low|medium|high`,
		})
	}
	if m.MaxPRsPerNight < 0 || m.MaxPRsPerNight > 50 {
		issues = append(issues, FieldIssue{
			Path:    "max_prs_per_night",
			Message: "must be between 0 and 50",
		})
	}
	for i, d := range m.Duties {
		base := fmt.Sprintf("duties[%d]", i)
		if strings.TrimSpace(d.Type) == "" {
			issues = append(issues, FieldIssue{Path: base + ".type", Message: "required"})
		}
		if !durationRe.MatchString(d.Interval) {
			issues = append(issues, FieldIssue{
				Path:    base + ".interval",
				Message: `must be a Go-style duration (e.g. "24h")`,
			})
		}
		switch d.Priority {
		case PriorityLow, PriorityMedium, PriorityHigh:
		default:
			issues = append(issues, FieldIssue{
				Path:    base + ".priority",
				Message: "must be one of low|medium|high",
			})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return issues
}

// Store is an in-memory, name-keyed collection of Members. It's safe
// for concurrent reads and writes.
type Store struct {
	mu      sync.RWMutex
	members map[string]Member
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{members: make(map[string]Member)}
}

// ErrNotFound is returned by Get/Remove when the named member doesn't exist.
var ErrNotFound = errors.New("family: member not found")

// ErrDuplicate is returned by Add when a name already exists.
var ErrDuplicate = errors.New("family: member with that name already exists")

// Seed loads members from a directory into the store, replacing any
// existing entries. Returns the number of members loaded.
func (s *Store) Seed(members []Member) int {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members = make(map[string]Member, len(members))
	for _, m := range members {
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		if m.UpdatedAt.IsZero() {
			m.UpdatedAt = now
		}
		s.members[m.Name] = m
	}
	return len(s.members)
}

// List returns all members sorted by name.
func (s *Store) List() []Member {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Member, 0, len(s.members))
	for _, m := range s.members {
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns the member with the given name or ErrNotFound.
func (s *Store) Get(name string) (Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.members[name]
	if !ok {
		return Member{}, ErrNotFound
	}
	return m, nil
}

// Add inserts a new member. Returns ErrDuplicate if one already exists
// with that name, or ValidationError if the member doesn't validate.
func (s *Store) Add(m Member) (Member, error) {
	m.applyDefaults()
	if issues := Validate(m); len(issues) > 0 {
		return Member{}, issues
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.members[m.Name]; ok {
		return Member{}, ErrDuplicate
	}
	m.CreatedAt = now
	m.UpdatedAt = now
	s.members[m.Name] = m
	return m, nil
}

// Put replaces an existing member, or creates one if it didn't exist.
// Returns the stored value and the validation result.
func (s *Store) Put(m Member) (Member, error) {
	m.applyDefaults()
	if issues := Validate(m); len(issues) > 0 {
		return Member{}, issues
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.members[m.Name]
	if ok {
		m.CreatedAt = existing.CreatedAt
	} else {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	s.members[m.Name] = m
	return m, nil
}

// Remove deletes a member. Returns ErrNotFound if the name isn't known.
func (s *Store) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.members[name]; !ok {
		return ErrNotFound
	}
	delete(s.members, name)
	return nil
}

// Len returns the number of members currently in the store.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.members)
}

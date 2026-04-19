// Package config loads the optional nfd config YAML from disk.
//
// Precedence (highest to lowest): explicit CLI flag → config file →
// built-in default. That way operators can set a reasonable baseline
// in ~/.config/night-family/config.yaml and override per-invocation
// with flags when debugging.
package config

import (
	"errors"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// Daemon mirrors the subset of nfd's flags that are sensible to put in
// a YAML file. Absent fields keep their built-in defaults.
type Daemon struct {
	Addr        string   `yaml:"addr"`
	LogLevel    string   `yaml:"log_level"`
	DB          string   `yaml:"db"`
	FamilyDir   string   `yaml:"family_dir"`
	Provider    string   `yaml:"provider"`
	ClaudeBin   string   `yaml:"claude_bin"`
	ClaudeArgs  []string `yaml:"claude_args"`
	Repo        string   `yaml:"repo"`
	BaseBranch  string   `yaml:"base_branch"`
	Reviewers   []string `yaml:"reviewers"`
	SignOff     *bool    `yaml:"signoff"`
	SkipPush    bool     `yaml:"skip_push"`
	SkipPR      bool     `yaml:"skip_pr"`
	AutoTrigger bool     `yaml:"auto_trigger"`
}

// Load reads path and returns the parsed Daemon. A missing file is
// not an error — returns (Daemon{}, nil) so callers can treat "no
// config" and "empty config" identically.
func Load(path string) (Daemon, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Daemon{}, nil
	}
	if err != nil {
		return Daemon{}, err
	}
	var d Daemon
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return Daemon{}, err
	}
	return d, nil
}

// DefaultPath returns the path nfd looks at when the operator doesn't
// pass --config. Honours XDG_CONFIG_HOME, falls back to
// ~/.config/night-family/config.yaml.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg + "/night-family/config.yaml"
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return home + "/.config/night-family/config.yaml"
}

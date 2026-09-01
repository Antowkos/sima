package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int                       `yaml:"version"`
	Project  Project                   `yaml:"project"`
	Policy   Policy                    `yaml:"policy"`
	Learn    Learn                     `yaml:"learn"`
	Brief    Brief                     `yaml:"brief,omitempty"`
	Team     Team                      `yaml:"team,omitempty"`
	Backends map[string]BackendProfile `yaml:"backends"`

	LearnConfigured bool `yaml:"-"`
}

type Project struct {
	Name string `yaml:"name"`
	Mode string `yaml:"mode"`
}

type Policy struct {
	PersonalAutoApply            bool `yaml:"personal_auto_apply"`
	TeamAutoApply                bool `yaml:"team_auto_apply"`
	RequireCleanArchivistSession bool `yaml:"require_clean_archivist_session"`
	RejectRewardHacking          bool `yaml:"reject_reward_hacking"`
}

type Learn struct {
	AutoApply           bool `yaml:"auto_apply"`
	AutoCleanupDeferred bool `yaml:"auto_cleanup_deferred"`
}

type Brief struct {
	Retrieval   string         `yaml:"retrieval,omitempty"`
	MaxSelected int            `yaml:"max_selected,omitempty"`
	Embedding   BriefEmbedding `yaml:"embedding,omitempty"`
	Query       BriefQuery     `yaml:"query,omitempty"`
}

type BriefEmbedding struct {
	Command  string  `yaml:"command,omitempty"`
	Model    string  `yaml:"model,omitempty"`
	MinScore float64 `yaml:"min_score,omitempty"`
}

type BriefQuery struct {
	Decomposition string `yaml:"decomposition,omitempty"`
	Command       string `yaml:"command,omitempty"`
	MaxParts      int    `yaml:"max_parts,omitempty"`
}

type Team struct {
	Repo      string `yaml:"repo,omitempty"`
	Ref       string `yaml:"ref,omitempty"`
	AutoApply bool   `yaml:"auto_apply"`
	SyncMode  string `yaml:"sync_mode,omitempty"`
}

func (t Team) IsZero() bool {
	return t.Repo == "" && t.Ref == "" && !t.AutoApply && t.SyncMode == ""
}

func DefaultLearn() Learn {
	return Learn{AutoApply: true, AutoCleanupDeferred: true}
}

type BackendProfile struct {
	Kind           string            `yaml:"kind"`
	Executable     string            `yaml:"executable"`
	ConfigPath     string            `yaml:"config_path,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	EnvFile        string            `yaml:"env_file,omitempty"`
	WorkingDir     string            `yaml:"working_dir,omitempty"`
	PermissionMode string            `yaml:"permission_mode,omitempty"`
	Metadata       map[string]string `yaml:"metadata,omitempty"`
}

func Path(projectRoot string) string {
	return filepath.Join(projectRoot, ".sima", "config.yaml")
}

func Load(projectRoot string) (Config, error) {
	path := Path(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Backends == nil {
		cfg.Backends = map[string]BackendProfile{}
	}
	cfg.LearnConfigured = hasTopLevelLearnBlock(string(data))
	if !cfg.LearnConfigured {
		cfg.Learn = DefaultLearn()
	}
	return cfg, nil
}

func Save(projectRoot string, cfg Config) error {
	path := Path(projectRoot)
	if cfg.Backends == nil {
		cfg.Backends = map[string]BackendProfile{}
	}
	if !cfg.LearnConfigured {
		cfg.Learn = DefaultLearn()
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func hasTopLevelLearnBlock(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, "learn:") {
			return true
		}
	}
	return false
}

func AddBackend(projectRoot, name string, profile BackendProfile, force bool) error {
	if name == "" {
		return errors.New("backend name is required")
	}
	if err := validateProfile(profile); err != nil {
		return err
	}
	cfg, err := Load(projectRoot)
	if err != nil {
		return err
	}
	if _, exists := cfg.Backends[name]; exists && !force {
		return fmt.Errorf("backend %q already exists (use --force to replace)", name)
	}
	cfg.Backends[name] = profile
	return Save(projectRoot, cfg)
}

func validateProfile(profile BackendProfile) error {
	switch profile.Kind {
	case "claude-code", "codex":
	case "":
		return errors.New("backend kind is required")
	default:
		return fmt.Errorf("unsupported backend kind %q", profile.Kind)
	}
	if profile.Executable == "" {
		return errors.New("backend executable is required")
	}
	return nil
}

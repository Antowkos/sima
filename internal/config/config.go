package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int                       `yaml:"version"`
	Project  Project                   `yaml:"project"`
	Policy   Policy                    `yaml:"policy"`
	Backends map[string]BackendProfile `yaml:"backends"`
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
	return cfg, nil
}

func Save(projectRoot string, cfg Config) error {
	path := Path(projectRoot)
	if cfg.Backends == nil {
		cfg.Backends = map[string]BackendProfile{}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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

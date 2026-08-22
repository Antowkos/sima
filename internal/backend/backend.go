package backend

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/antowkos/sima/internal/config"
)

type DoctorResult struct {
	Name   string
	OK     bool
	Detail string
}

func Doctor(name string, profile config.BackendProfile) DoctorResult {
	resolved, err := exec.LookPath(expandHome(profile.Executable))
	if err != nil {
		return DoctorResult{Name: name, OK: false, Detail: fmt.Sprintf("executable not found: %s", profile.Executable)}
	}
	if profile.ConfigPath != "" {
		if _, err := os.Stat(expandHome(profile.ConfigPath)); err != nil {
			return DoctorResult{Name: name, OK: false, Detail: fmt.Sprintf("config_path invalid: %v", err)}
		}
	}
	if profile.EnvFile != "" {
		if _, err := os.Stat(expandHome(profile.EnvFile)); err != nil {
			return DoctorResult{Name: name, OK: false, Detail: fmt.Sprintf("env_file invalid: %v", err)}
		}
	}
	return DoctorResult{Name: name, OK: true, Detail: resolved}
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return home + path[1:]
		}
	}
	return path
}

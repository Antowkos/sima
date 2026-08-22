package simafs

import (
	"os"
	"path/filepath"
)

type Check struct {
	Name   string
	OK     bool
	Detail string
}

type Report struct {
	Checks []Check
}

func (r Report) OK() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

func Doctor(projectRoot string) Report {
	simaRoot := filepath.Join(projectRoot, ".sima")
	checks := []Check{
		pathCheck(".sima exists", simaRoot),
		pathCheck("config.yaml exists", filepath.Join(simaRoot, "config.yaml")),
		pathCheck("schema.yaml exists", filepath.Join(simaRoot, "schema.yaml")),
		pathCheck("personal memory exists", filepath.Join(simaRoot, "personal", "memory", "cards")),
		pathCheck("personal skills exists", filepath.Join(simaRoot, "personal", "skills", "active")),
		pathCheck("team scaffold exists", filepath.Join(simaRoot, "team")),
		pathCheck("archivist prompt exists", filepath.Join(simaRoot, "system", "prompts", "archivist.md")),
	}
	return Report{Checks: checks}
}

func pathCheck(name, path string) Check {
	if _, err := os.Stat(path); err != nil {
		return Check{Name: name, OK: false, Detail: err.Error()}
	}
	return Check{Name: name, OK: true}
}

package team

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antowkos/sima/internal/config"
)

type InitOptions struct {
	Repo string
	Ref  string
}

type Status struct {
	Configured   bool
	Repo         string
	Ref          string
	SyncMode     string
	AutoApply    bool
	SourceExists bool
	MemoryCards  int
	Skills       int
}

type PullResult struct {
	Repo       string
	Ref        string
	SourcePath string
	Copied     []string
}

func Init(projectRoot string, opts InitOptions) error {
	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		return fmt.Errorf("team repo is required")
	}
	ref := strings.TrimSpace(opts.Ref)
	if ref == "" {
		ref = "main"
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return err
	}
	cfg.Team = config.Team{Repo: repo, Ref: ref, AutoApply: false, SyncMode: "mirror"}
	return config.Save(projectRoot, cfg)
}

func Inspect(projectRoot string) (Status, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return Status{}, err
	}
	status := Status{
		Configured: strings.TrimSpace(cfg.Team.Repo) != "",
		Repo:       cfg.Team.Repo,
		Ref:        cfg.Team.Ref,
		SyncMode:   cfg.Team.SyncMode,
		AutoApply:  cfg.Team.AutoApply,
	}
	if status.Ref == "" {
		status.Ref = "main"
	}
	if status.SyncMode == "" {
		status.SyncMode = "mirror"
	}
	_, err = os.Stat(sourcePath(projectRoot))
	status.SourceExists = err == nil
	status.MemoryCards = countFiles(filepath.Join(projectRoot, ".sima", "team", "memory", "cards"), []string{".yaml", ".yml", ".md"})
	status.Skills = countFiles(filepath.Join(projectRoot, ".sima", "team", "skills", "active"), []string{".md"})
	return status, nil
}

func Pull(projectRoot string) (PullResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return PullResult{}, err
	}
	if strings.TrimSpace(cfg.Team.Repo) == "" {
		return PullResult{}, fmt.Errorf("team repo is not configured; run sima team init --repo <git-url>")
	}
	ref := strings.TrimSpace(cfg.Team.Ref)
	if ref == "" {
		ref = "main"
	}
	if cfg.Team.SyncMode != "" && cfg.Team.SyncMode != "mirror" {
		return PullResult{}, fmt.Errorf("unsupported team sync_mode %q", cfg.Team.SyncMode)
	}

	source := sourcePath(projectRoot)
	if _, err := os.Stat(filepath.Join(source, ".git")); err == nil {
		if err := git(source, "fetch", "--prune", "origin"); err != nil {
			return PullResult{}, err
		}
		if err := git(source, "checkout", ref); err != nil {
			if err2 := git(source, "checkout", "-B", ref, "origin/"+ref); err2 != nil {
				return PullResult{}, fmt.Errorf("checkout %s: %w", ref, err)
			}
		}
		if err := git(source, "pull", "--ff-only", "origin", ref); err != nil {
			return PullResult{}, err
		}
	} else {
		if err := os.RemoveAll(source); err != nil {
			return PullResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
			return PullResult{}, err
		}
		if err := git("", "clone", "--depth", "1", "--branch", ref, cfg.Team.Repo, source); err != nil {
			return PullResult{}, err
		}
	}

	copied := []string{}
	pairs := []struct{ src, dst string }{
		{filepath.Join(source, "memory", "cards"), filepath.Join(projectRoot, ".sima", "team", "memory", "cards")},
		{filepath.Join(source, "skills", "active"), filepath.Join(projectRoot, ".sima", "team", "skills", "active")},
	}
	for _, pair := range pairs {
		rels, err := mirrorDir(pair.src, pair.dst, projectRoot)
		if err != nil {
			return PullResult{}, err
		}
		copied = append(copied, rels...)
	}
	return PullResult{Repo: cfg.Team.Repo, Ref: ref, SourcePath: source, Copied: copied}, nil
}

func sourcePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".sima", "team", "source")
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mirrorDir(src, dst, projectRoot string) ([]string, error) {
	if err := os.RemoveAll(dst); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil, nil
	}
	var copied []string
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relFromSrc, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relFromSrc)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		relToProject, err := filepath.Rel(projectRoot, target)
		if err != nil {
			return err
		}
		copied = append(copied, filepath.ToSlash(relToProject))
		return nil
	})
	sort.Strings(copied)
	return copied, err
}

func countFiles(root string, exts []string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		for _, ext := range exts {
			if strings.EqualFold(filepath.Ext(path), ext) {
				count++
				break
			}
		}
		return nil
	})
	return count
}

package team

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/simafs"
)

func TestTeamInitPullAndStatus(t *testing.T) {
	project := t.TempDir()
	if _, err := simafs.Init(project); err != nil {
		t.Fatalf("init project: %v", err)
	}
	teamRepo := createTeamRepo(t)

	if err := Init(project, InitOptions{Repo: teamRepo, Ref: "main"}); err != nil {
		t.Fatalf("team init: %v", err)
	}
	cfg, err := config.Load(project)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Team.Repo != teamRepo || cfg.Team.Ref != "main" || cfg.Team.SyncMode != "mirror" || cfg.Team.AutoApply {
		t.Fatalf("unexpected team config: %#v", cfg.Team)
	}

	result, err := Pull(project)
	if err != nil {
		t.Fatalf("team pull: %v", err)
	}
	if len(result.Copied) != 2 {
		t.Fatalf("expected two mirrored files, got %#v", result.Copied)
	}
	memoryData, err := os.ReadFile(filepath.Join(project, ".sima", "team", "memory", "cards", "api.yaml"))
	if err != nil {
		t.Fatalf("read mirrored memory: %v", err)
	}
	if !strings.Contains(string(memoryData), "Team API invariant") {
		t.Fatalf("unexpected memory mirror:\n%s", memoryData)
	}
	skillData, err := os.ReadFile(filepath.Join(project, ".sima", "team", "skills", "active", "api-review.md"))
	if err != nil {
		t.Fatalf("read mirrored skill: %v", err)
	}
	if !strings.Contains(string(skillData), "Team API review") {
		t.Fatalf("unexpected skill mirror:\n%s", skillData)
	}

	status, err := Inspect(project)
	if err != nil {
		t.Fatalf("team status: %v", err)
	}
	if !status.Configured || !status.SourceExists || status.MemoryCards != 1 || status.Skills != 1 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func createTeamRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, dir := range []string{"memory/cards", "skills/active"} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	memory := "id: team-api\nstatus: active\ntype: invariant\ntitle: Team API invariant\ntrigger: When editing team API handlers.\nsummary: Team API handlers use shared request validation.\n"
	if err := os.WriteFile(filepath.Join(repo, "memory", "cards", "api.yaml"), []byte(memory), 0o644); err != nil {
		t.Fatalf("write memory: %v", err)
	}
	skill := "---\nstatus: active\n---\n# Team API review\n\nUse when reviewing team API changes.\n"
	if err := os.WriteFile(filepath.Join(repo, "skills", "active", "api-review.md"), []byte(skill), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "-c", "user.name=SIMA Test", "-c", "user.email=sima@example.test", "commit", "-m", "seed team knowledge")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

# Backend Profiles and SDD Support Implementation Plan

> **For Hermes:** Use this plan as the current SIMA dogfood slice. Keep tasks small, verified, and committed.

**Goal:** Add backend profiles for multiple Claude Code/Codex configurations and lay down SDD as a first-class SIMA workflow.

**Architecture:** Store named backend profiles in `.sima/config.yaml`, expose `sima backend list/add/doctor`, and keep SDD support as docs/system prompt groundwork before full `sima brief/run` exists.

**Tech Stack:** Go CLI, YAML config, project-local `.sima/` state.

---

### Task 1: Backend config model

**Objective:** Load/save backend profiles from `.sima/config.yaml`.

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Verification:** `go test ./internal/config`

### Task 2: Backend doctor

**Objective:** Validate configured executables/config paths without assuming one global Claude/Codex.

**Files:**
- Create: `internal/backend/backend.go`

**Verification:** `go test ./...`

### Task 3: CLI backend commands

**Objective:** Add `sima backend list/add/doctor`.

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

**Verification:**

```bash
./sima backend list
./sima backend add test --kind codex --executable /bin/echo
./sima backend doctor test
```

### Task 4: SDD docs/system groundwork

**Objective:** Document SDD as a first-class workflow SIMA should preserve and summarize, not blindly ingest.

**Files:**
- Modify: `docs/technical-spec-v0.md`
- Modify: `internal/simafs/assets.go`

**Verification:** `go test ./...`

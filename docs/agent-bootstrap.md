# Agent Bootstrap from a Repository URL

Use this when a user gives Claude Code, Codex, or another coding agent the SIMA repository URL and asks it to install SIMA for the current project.

The expected modern flow is agent-assisted but explicit:

1. Clone or update the SIMA source checkout.
2. Run SIMA's binary-only installer.
3. Return to the user's project.
4. Run explicit project setup only if the user asked for setup/onboarding.
5. Verify the installation with real commands.

## Preconditions

- The agent has shell access in the user's development environment.
- `git` is installed.
- Go is installed and available on `PATH`.
- For private repositories, GitHub auth already works through `gh auth status` or git credentials. Agents must not ask the user to paste tokens into chat.
- The current directory is the project to initialize, or the user provides the project path.

If a prerequisite is missing, report the blocker and the exact command the user should run, rather than guessing credentials or silently skipping verification.

## Recommended agent prompt

```text
Install SIMA from <repo-url> for this project. Clone or update the source checkout, run its binary-only install, then run project setup in the current repo. Do not use native agent memory for project knowledge; after setup follow the generated CLAUDE.md/AGENTS.md instructions.
```

## Safe install commands

From the user's project directory:

```bash
PROJECT="$PWD"
SIMA_REPO_URL="https://github.com/Antowkos/sima.git"
SIMA_SOURCE_DIR="${SIMA_SOURCE_DIR:-$HOME/.local/share/sima}"

if [ -d "$SIMA_SOURCE_DIR/.git" ]; then
  git -C "$SIMA_SOURCE_DIR" pull --ff-only
else
  git clone "$SIMA_REPO_URL" "$SIMA_SOURCE_DIR"
fi

cd "$SIMA_SOURCE_DIR"
./install.sh

cd "$PROJECT"
sima setup
sima doctor .
```

For private repositories, prefer `gh repo clone` when available and authenticated:

```bash
PROJECT="$PWD"
SIMA_REPO="Antowkos/sima"
SIMA_SOURCE_DIR="${SIMA_SOURCE_DIR:-$HOME/.local/share/sima}"

gh auth status
if [ -d "$SIMA_SOURCE_DIR/.git" ]; then
  git -C "$SIMA_SOURCE_DIR" pull --ff-only
else
  gh repo clone "$SIMA_REPO" "$SIMA_SOURCE_DIR" -- --depth 1
fi

cd "$SIMA_SOURCE_DIR"
./install.sh

cd "$PROJECT"
sima setup
sima doctor .
```

## Install-only mode

If the user only asked to install the binary, stop after:

```bash
cd "$SIMA_SOURCE_DIR"
./install.sh
sima version
```

Do not mutate the user's project unless setup was requested explicitly.

## Backend selection

Project setup auto-detects Claude Code/Codex by default. Agents may pass explicit backend config when the user requested it:

```bash
sima setup --backend claude --claude-config-dir ~/.claude-work
sima setup --backend codex --executable "$(command -v codex)"
sima setup --backend none
```

For Codex, run `codex doctor` first. If auth fails, tell the user to run `codex login`; do not treat an unauthenticated Codex backend as ready for `sima learn`.

## Verification

A completed setup must show real output from:

```bash
sima version
sima doctor .
sima lint .
```

Expected healthy state:

```text
[ok] project: .sima exists
[ok] config: load: version loaded
[ok] lint: errors: 0 errors, 0 warnings
```

If no backend is configured because `--backend none` was used or no executable was found, `sima lint .` should still pass and the agent should report that `sima doctor .` will become fully green after `sima backend add ...`.

## Pitfalls

- Do not pipe remote scripts from the internet into `sh` for a private development tool. Clone first, inspect/use repo files, then run `./install.sh`.
- Do not use `./install.sh --setup` unless the user asked for one-command install+setup. Binary-only install is the default.
- Do not store tokens or credentials in SIMA memory, logs, or docs.
- Do not assume shell aliases are available to non-interactive agent processes; use explicit executable paths or SIMA backend env/config flags.

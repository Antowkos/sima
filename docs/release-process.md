# Release Process

SIMA releases are intentionally tag-driven. Normal pushes to `main` run CI only; GitHub Releases are created only for explicit semantic-version tags.

## Git flow

Use this flow so releases contain understandable versions rather than every commit:

1. Land normal work on `main` through focused commits or pull requests.
2. Keep `main` green: CI must pass before tagging.
3. Choose a semantic version:
   - stable: `v0.1.0`, `v1.2.3`
   - prerelease: `v0.2.0-alpha.1`, `v0.2.0-rc.1`
4. Create an annotated tag from the exact commit to release.
5. Push the tag. The release workflow builds artifacts and publishes the GitHub Release.

Do not tag arbitrary WIP commits. If a release needs a fix, land the fix on `main`, then create a new patch/prerelease tag.

## Commands

From a clean `main` checkout:

```bash
git switch main
git pull --ff-only
go test ./...
./sima lint .
./sima doctor .

git tag -a v0.1.0 -m "SIMA v0.1.0"
git push origin v0.1.0
```

The pushed tag triggers `.github/workflows/release.yml`.

## What CI does

`.github/workflows/ci.yml` runs on:

- pull requests;
- pushes to `main`.

It verifies:

```bash
go test ./...
go build -o sima ./cmd/sima
./sima version
```

## What release does

`.github/workflows/release.yml` runs only on:

- pushed tags matching `vMAJOR.MINOR.PATCH`;
- pushed prerelease tags matching `vMAJOR.MINOR.PATCH-PRERELEASE`;
- manual `workflow_dispatch` for an existing semver tag.

It verifies the tag format, runs tests, builds cross-platform archives, generates SHA-256 checksum files, and creates a GitHub Release.

Build targets:

- `darwin/arm64`
- `darwin/amd64`
- `linux/arm64`
- `linux/amd64`
- `windows/amd64`

The binary version is injected from the tag, so a `v0.1.0` release prints:

```bash
sima version
# sima 0.1.0
```

Prerelease tags containing `-` create GitHub prereleases automatically.

## Manual release dispatch

If the tag already exists but the workflow needs to be rerun:

1. Open GitHub Actions → Release.
2. Run workflow manually.
3. Enter an existing semver tag, for example `v0.1.0`.

## Safety rules

- Never release from a dirty working tree.
- Never create release tags for transient dogfood/WIP commits.
- Never put tokens or credentials in release notes, artifacts, or `.sima/` evidence.
- If a release is bad, do not rewrite published tags by default. Create a new patch tag such as `v0.1.1`.

## Suggested branch protection

For team alpha, configure GitHub branch protection for `main`:

- require PR or at least require status checks before merging;
- require the `CI / Test` workflow to pass;
- restrict force pushes;
- keep tag creation limited to maintainers.

This keeps the release workflow simple: **only humans/maintainers decide when a version exists by pushing a semver tag**.

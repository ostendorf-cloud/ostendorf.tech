# govue's two halves and the pnpm pin

## govue's two halves move together

govue releases a Go module and a Vue package under one tag. `go.mod`'s
`require github.com/ostendorf-cloud/govue` and `web/package.json`'s
`@ostendorf-cloud/govue` pnpm git dependency must always name the same tag: a mismatch
means the repository takes half of a govue release and not the other half, silently.

`task check:govue-version` (`internal/depsync`) enforces this. It asks `go list -m` for
the version go.mod actually resolves to, the same resolution `go build` itself uses, and
pulls the tag out of package.json's `ostendorf-cloud/govue#<tag>&path:/web` specifier. It
then fails naming both versions and both files when they differ.

`web/pnpm-lock.yaml`'s own specifier field already has to match package.json's for
`pnpm install --frozen-lockfile` to pass. Its resolved entry pins a commit hash, not a
tag, so it cannot be compared to a version directly either.

That match is not enough on its own. pnpm's own git resolver has written a resolution
object for this dependency with no `path` field at all, even from a fully fresh resolve
against a cold store. That makes `pnpm install --frozen-lockfile` install the whole govue
repository instead of its `web/` subdirectory while every specifier still agrees
(pnpm/pnpm#12304, fixed by pnpm/pnpm#12344, not backported to the pnpm 10 release line
as far as this repository could establish). `check:govue-version` therefore also reads
`web/pnpm-lock.yaml` directly and fails when that dependency's resolution carries no
`path: /web`.

## Renovate

- Renovate raises both halves in one pull request, group `govue` in `renovate.json`.
- Its own npm manager skips the `&path:/web` specifier (`unversioned-reference`), so a
  regex custom manager reads the tag from `web/package.json`.
- Renovate regenerates no lock file for a custom manager, so the shared preset
  `govue-lockfile` (extended in `renovate.json`) runs one fixed post upgrade command,
  `pnpm --dir web install --lockfile-only --ignore-scripts --ignore-pnpmfile`, and
  commits `web/pnpm-lock.yaml`.
- The `constraints.pnpm` value of the govue rule equals the pin in `mise.toml`: move both
  together.
- Check on every govue pull request that the lock file entry still carries `path: /web`.
  `task check:govue-version` fails when it does not.
- The command and its limits are described in the workflows repository,
  `docs/concepts/renovate-post-upgrade.md`.
- `internal/depsync/renovate_test.go` fails when the group or the manager goes missing.

## pnpm is pinned three times and moves once

- `mise.toml` pins pnpm for a local run.
- The Dockerfile's Node stage installs the same patch through corepack from its
  `ARG PNPM_VERSION`, so the image reads the lockfile with the pnpm that wrote it.
- The govue rule's `constraints.pnpm` in `renovate.json` names it for the post upgrade
  lock file command.

Renovate's mise manager reads the first pin. A regex custom manager reads the argument
under its `# renovate: datasource=npm depName=pnpm` comment. Renovate's own
`renovate-config` manager reads the constraint as a tool constraint. The package rule
with `groupName` `pnpm` matches all three managers and puts the three edits into one
pull request with no manual step.

- `internal/envsync`'s `TestDockerfilePnpmMatchesMise` fails when the Dockerfile and
  `mise.toml` differ or the install line names a literal version.
- `internal/depsync/renovate_test.go` fails when the argument is no longer extracted, the
  group misses a manager, or the constraint differs from `mise.toml`.

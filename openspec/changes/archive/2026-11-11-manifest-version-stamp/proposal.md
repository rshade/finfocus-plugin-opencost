# Proposal

## Why

`finfocus plugin install opencost` installs release v0.1.2 into
`<home>/plugins/opencost/v0.1.2/`. `finfocus plugin validate --plugin opencost`
then fails with `manifest version mismatch: expected v0.1.2, got 0.1.0`. Core
compares the manifest `version` with the install directory name exactly, and a
registry install names that directory after the release tag. GoReleaser
archived the checked-in `plugin.manifest.json`, whose `version` stayed `0.1.0`,
so no released build could pass `plugin validate`. `make install` copied no
manifest and used a bare `0.1.0` directory that no release matches.

## What Changes

- `hack/manifestversion` writes a copy of `plugin.manifest.json` with the
  top-level `version` set to a `v`-prefixed semver and every other byte kept.
- A GoReleaser `before` hook writes `build/release/plugin.manifest.json` with
  `v{{ .Version }}`. The archives ship that file at their root instead of the
  checked-in manifest.
- `make install` builds first, installs to
  `$FINFOCUS_HOME` (or `~/.finfocus`) `/plugins/opencost/v$(VERSION)/`, and
  writes the stamped manifest there. `VERSION` defaults to the
  `.release-please-manifest.json` version.

## Non-goals

- No change to core. Core keeps its exact comparison.
- The checked-in manifest `version` is not bumped by Release Please. Its JSON
  updater writes a bare version, which core would still reject.

## Capabilities

### New Capabilities

- `release-manifest`: shipped manifests carry the release version.

### Modified Capabilities

- None.

## Impact

- `.goreleaser.yaml`, `Makefile`, `.gitignore`, `hack/manifestversion`.
- `README.md` and `CLAUDE.md`.

# release-manifest Specification

## Purpose

Ship a `plugin.manifest.json` whose `version` matches the directory FinFocus core installs the plugin into, so `finfocus plugin validate` passes on release and local installs.

## Requirements

### Requirement: Release archives carry the release version

Every release archive SHALL contain `plugin.manifest.json` at its root with
`version` equal to the release tag: `v` followed by the semver (`v0.1.2`).
Core's `plugin validate` compares that value with the install directory, which
a registry install names after the tag.

#### Scenario: Registry install validates

- **WHEN** release `v0.1.2` is installed into `plugins/opencost/v0.1.2/`
- **THEN** its `plugin.manifest.json` `version` is `v0.1.2`
- **AND** `finfocus plugin validate --plugin opencost` reports it valid

Tested by: TestGoreleaserArchivesStampedManifest, TestStampCheckedInManifest

### Requirement: Local install stamps the same version

`make install` SHALL write the binary and a stamped manifest into
`plugins/opencost/v<version>/`. The stamp SHALL change only the top-level
`version` and SHALL reject a version that lacks the `v` prefix.

#### Scenario: Local install validates

- **WHEN** `make install` runs with `VERSION` 0.1.2
- **THEN** the binary and a manifest with `version` `v0.1.2` are in `plugins/opencost/v0.1.2/`

#### Scenario: Bare version is refused

- **WHEN** the stamp is asked for version `0.1.2`
- **THEN** it fails and writes no manifest

Tested by: TestStampRejectsBadInput, TestRunWritesStampedFile

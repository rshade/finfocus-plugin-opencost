# Tasks

## 1. Stamp the manifest version

- [x] 1.1 `hack/manifestversion` replaces only the top-level `version` and rejects a version without the `v` prefix. Verify: `go test -count=1 ./hack/manifestversion/` fails before, passes after.
- [x] 1.2 GoReleaser stamps `v{{ .Version }}` and archives the stamped file. Verify: `TestGoreleaserArchivesStampedManifest` fails before, passes after; `goreleaser check` exits 0; a local tagged build's archive manifest reads `v0.1.2` and core's `plugin validate` passes on it.
- [x] 1.3 `make install` writes the stamped manifest into `plugins/opencost/v$(VERSION)/`. Verify: `FINFOCUS_HOME=<scratch> make install`, then core's `plugin validate --plugin opencost` exits 0.

## 2. Docs and close out

- [x] 2.1 README install section and CLAUDE.md. Verify: `npx markdownlint-cli2` on the changed files exits 0.
- [x] 2.2 Full gates: `make build vet lint`, `go test -count=1 ./...`, `mise exec -- openspec validate --all` and `mise exec -- openspec validate release-manifest --type spec --strict`, then archive.

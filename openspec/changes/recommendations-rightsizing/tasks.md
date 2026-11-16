# Tasks

## 1. Config

- [x] 1.1 Write `TestRecommendationsConfig` (defaults, YAML, env overrides, invalid env names the variable, validation without headroom, validation with profile `kubecost`), then add `Recommendations` and `RecommendationHeadroom` to `allocation.Config` with their env overrides and validation; document both keys in the README and `config.example.yaml`. Verify: `go test -count=1 ./internal/allocation/ -run 'Recommendations|Readme|ConfigValidate'` exits 0. Break check: drop the headroom check from `Validate` and the test fails.

## 2. Capability

- [x] 2.1 Extend `TestGetPluginInfoCapabilitiesFollowProfile` with a recommendations case and the manifest pin in `naming_test.go`, then report `RECOMMENDATIONS` from `Info` when enabled and add it to `plugin.manifest.json`. Verify: `go test -count=1 ./internal/server/ -run 'PluginInfo|Kubecost'` exits 0. Break check: always add the capability and the default-profile case fails.

## 3. RPC

- [x] 3.1 Write `TestRecommendationsOffAnswersEmpty`, `TestRecommendationsFromRecordedControllers`, `TestRecommendationsSkipRowsWithNothingToLower`, and `TestRecommendationsHonourRequestScope` against `testdata/opencost-real/allocation-controller-60m.json`, then add `internal/server/recommendations.go` and move `cpuCoreHours` and `ramByteHours` to `ConsumedFields`. Document the RPC in the README. Verify: `go test -count=1 ./internal/server/ -run Recommendations` and `go test ./internal/allocation/ -run WireFields` exit 0. Break check: price savings from the configured rate instead of the row and the recorded-savings assertion fails.

## 4. Integration

- [x] 4.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, `make drift`, `npx markdownlint-cli2 "**/*.md"`, and `mise exec -- openspec validate --all --strict`; each exits 0.

## Workflow follow-up

- Archive the change in the same commit as the last code task. Profile `kubecost` stays open on #138.

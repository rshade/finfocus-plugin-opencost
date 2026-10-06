# Tasks

## 1. Parse errors

- [x] 1.1 Return an error naming the file path when the config file fails to parse. Verify: `go test -count=1 ./internal/allocation/ -run TestConfigFileParseErrorNamesPath` fails before the change, passes after. Break check: restore `_ = yaml.Unmarshal` and the test fails.
- [x] 1.2 Return an error naming the variable for an unparseable `KUBECOST_TIMEOUT` and for a `KUBECOST_TLS_SKIP_VERIFY` value other than `true` or `false`. Verify: `go test -count=1 ./internal/allocation/ -run TestEnvParseErrorNamesVariable` table test, one row per variable. Break check: restore the silent fallback and each row fails.

## 2. Startup validation

- [x] 2.1 Add `Config.Validate`: `baseUrl` is an `http(s)` URL with a host, `profile` is empty or `opencost` or `kubecost`, a set `currency` is three upper-case letters, and `timeout`, `requestsPerSecond`, `rateBurst` are not negative. Verify: `go test -count=1 ./internal/allocation/ -run TestConfigValidate`.
- [x] 2.2 Call `Validate` in `main.go` before `pluginsdk.Serve` and exit 1 with a one-line reason. Verify: `go test -count=1 ./cmd/finfocus-plugin-opencost/ -run Validate`; the built binary with an empty `KUBECOST_BASE_URL` exits 1. Break check: skip the `Validate` call and the binary test fails.

## 3. Token safety and docs

- [x] 3.1 Test that `KUBECOST_API_TOKEN` set to a sentinel with each invalid config never appears in the error. Verify: `go test -count=1 ./internal/allocation/ -run TestConfigErrorsNeverContainToken`.
- [x] 3.2 State the real precedence in the README: defaults, then environment, then the file for keys it sets, then `OPENCOST_PROFILE`, `OPENCOST_CURRENCY`, and `KUBECOST_API_TOKEN`, which always win. Verify: `markdownlint README.md` exit 0.

## 4. Close out

- [x] 4.1 Existing valid-config tests pass unchanged: `go test -count=1 ./internal/allocation/ ./cmd/finfocus-plugin-opencost/`.
- [x] 4.2 `mise exec -- openspec validate --all --strict` exit 0, then archive the change.

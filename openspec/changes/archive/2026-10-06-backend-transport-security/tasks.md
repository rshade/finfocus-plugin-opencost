# Tasks

## 1. Plain HTTP

- [x] 1.1 Reject a non-loopback `http` base URL in `Validate` unless `allowInsecureHttp` is set. Accept `localhost`, `127.0.0.0/8`, and `::1`. Verify: `go test -count=1 ./internal/allocation/ -run TestInsecure`. Break check: accepting `http://example.com` fails that test.

## 2. Skip-verify warning

- [x] 2.1 Log one warning from startup when `tlsSkipVerify` is on, and none when it is off. The line contains no token and no URL userinfo. Verify: `go test -count=1 ./internal/allocation/ -run TestTLS`. Break check: warning while the flag is off fails the test.

## 3. Private CA

- [x] 3.1 Add `caCertFile` and trust its PEM on the backend client. A missing or unparsable file fails startup. Verify: `go test -count=1 ./internal/allocation/ -run TestCA`. Break check: ignoring the file fails the success case.

## 4. Docs

- [x] 4.1 Document `allowInsecureHttp`, `caCertFile`, their env names, and the gRPC authentication non-goal. Verify: `go test -count=1 ./internal/allocation/ -run TestReadmeDocumentsEveryConfigKey`.

## 5. Close out

- [x] 5.1 `mise exec -- openspec validate backend-transport-security --strict` exits 0, then archive the change in this commit.

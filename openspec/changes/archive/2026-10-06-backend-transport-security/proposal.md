# Proposal

## Why

A non-loopback `http` base URL sends the Kubecost bearer token in the clear, and a private CA today forces `tlsSkipVerify`. Startup should reject that URL, warn when verification is off, and trust a named CA file.

## What Changes

- Reject a non-loopback `http` base URL unless `allowInsecureHttp` is set. `http://localhost`, `127.0.0.0/8`, and `::1` still start.
- Log one warning at startup when `tlsSkipVerify` is on, and none when it is off. The line names no token and no URL userinfo.
- Add `caCertFile`. A PEM bundle is added to the system trust pool. A missing or unparsable file stops startup.
- Document both keys in the README and `config.example.yaml`. gRPC authentication stays a non-goal.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `config-loading`: startup validation gains the plain-HTTP rule, the skip-verify warning, and the private CA file.

## Impact

- `internal/allocation/config.go` and `client.go` load and enforce the new keys.
- `cmd/finfocus-plugin-opencost/main.go` logs the warning before serve.
- README and `config.example.yaml` list the keys. No proto change and no gRPC authentication.

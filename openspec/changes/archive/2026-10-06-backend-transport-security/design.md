# Design

## Context

See proposal.md. `Config.Validate` already requires an `http` or `https` URL with a host. `httpClient` sets `InsecureSkipVerify` from `tlsSkipVerify` and uses the default root pool. `main` logs config failures and then serves.

## Goals / Non-Goals

**Goals:**

- Fail startup on a non-loopback `http` URL unless the operator opts in.
- One warning when certificate checks are off, and none when they are on.
- Trust a PEM file without turning verification off.

**Non-Goals:**

- gRPC authentication. The host dials the plugin on `127.0.0.1`.
- Client certificates, token rotation, and refusing `allowInsecureHttp` when a token is set.

## Decisions

- The plain-HTTP rule lives in `Validate`, next to the existing URL check, so the process exits before `Serve`. The transport does not silently rewrite the URL.
- Loopback is `localhost` (any case) plus `net.IP.IsLoopback`, which covers `127.0.0.0/8` and `::1`. Other names such as `opencost.local` are not loopback.
- `allowInsecureHttp` is a bool seeded from `KUBECOST_ALLOW_INSECURE_HTTP` and overridable by the file, the same way `tlsSkipVerify` works. A token on a public `http` URL stays the operator's opt-in. Refusing that pair would block a deliberate lab setup the issue leaves open.
- `caCertFile` is appended to `x509.SystemCertPool`. If the system pool cannot be loaded, the pool is only the file. Zero parsed certificates is an error naming `caCertFile`. `Validate` and `NewClient` both load the file, so a caller that skips `Validate` still fails.
- `tlsSkipVerify` still disables verification when a CA file is also set. The warning is the signal. The warning text is fixed and does not include the URL.
- `main` calls `Config.WarnIfTLSSkipped` once after validation. The test builds the plugin binary so a removed call fails.

## Risks / Trade-offs

- [Existing tests treat `http://h:1` as a valid URL for other fields] → Those cases move to `https` so they still assert profile, currency, and limits.
- [A copied `config.example.yaml` with a missing CA path would refuse to start] → The example leaves `caCertFile` empty.
- [System pool plus a private CA trusts both] → That is the point of appending. Replacing the system pool would drop public CAs.

## Context

`NewClient` built a bare `http.Client`, so the TLS transport inside
`GetDetailedAllocation` never ran. YAML unmarshalling overwrote
`KUBECOST_API_TOKEN`. Filter values are interpolated as `key:"value"` and
joined with `+`.

## Goals / Non-Goals

**Goals:**

- Certificate verification is on unless `tlsSkipVerify` is true.
- The token is the environment variable, including when a YAML file sets `apiToken`.
- A kubecost request log names the URL and omits the token.
- An allocation error string omits the token.
- A hostile namespace never reaches the backend and is `InvalidArgument`.

**Non-Goals:**

- Certificate pinning, custom CAs, or gRPC authentication.
- Rate limiting. That is OC-4.2.
- Closing issue #13. Its other acceptance criteria are not this change.

## Decisions

- Reserved characters are `"`, `+`, `\`, whitespace, `(`, and `)`.
- The status message does not echo the rejected value.
- Request logs use the redacted URL and do not include headers.
- The OpenCost profile still does not send the token. Prediction uses that same rule.

## Risks / Trade-offs

- A namespace that legitimately contains one of those characters cannot be queried.
  Kubernetes names do not allow them.

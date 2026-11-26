# internal/openbao

`internal/openbao` is a small, standard library only HTTP client against OpenBao's KV v2
API, ported from old_simon-ostendorf.de's own `internal/openbao`.

- It logs in through a `Login` interface that returns the token and its lease.
- `Kubernetes` is the method for a cluster. It reads the projected service account token
  fresh on every login rather than caching it, because the kubelet rotates that file.
- The client re-logs in at two thirds of the lease instead of waiting for `renew-self`
  to fail, and retries a login once on a 403.
- `StaticToken` is the method for local tests: a token handed to the process, sent as
  `X-Vault-Token`, never renewed. Its 403 is the fixed `ErrForbidden` with no retry and
  no response body.
- Another method is one more `Login` implementation.
- Every token is an `openbao.Token`, which prints as `[redacted]` under every `fmt` verb,
  in `slog` and in JSON. The one plain string conversion is the request header.
- `Write` is a full-document overwrite guarded by the version `Read` last saw, so two
  replicas writing at once cannot silently clobber each other.
- `cmd/feed` is the only binary that constructs one of these. The frontend and the API
  have no OpenBao client at all.
- `internal/secrettest` holds the test helpers that prove a secret stays out of every
  error in an unwrap tree and every formatted value.

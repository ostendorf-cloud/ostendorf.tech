# internal/instagram

`internal/instagram` is the domain logic behind the home page's Impressionen section.

- `Client` talks to Meta's Graph API.
- `Feed` caches the posts and their picture bytes.
- `Syncer` keeps `Feed` current.
- `TokenStore` reads and writes the access token and its refresh time under two
  configured key names. It has no built-in defaults; `internal/feedconfig` supplies
  them. It keeps every other key of the secret.
- It uses `internal/openbao`'s KV v2 API under optimistic concurrency (CAS). It
  refreshes the token against Meta once it passes `TokenRefreshAge` and treats a CAS
  mismatch as the other side's write winning rather than an error.

## Where it runs

- `cmd/feed` is the only binary that imports this package, the only one with an OpenBao
  client, and the only one that ever holds the access token.
- The token stays in memory, is never logged and never returned in an error.
- A refreshed token is written back to OpenBao, so it survives a restart. The manual
  rotation this needed before Meta's 60-day expiry is gone, not merely less frequent.
- Ported from old_simon-ostendorf.de's `internal/instagram`, with one change: where a
  refreshed token is stored, OpenBao rather than nowhere.
- `internal/config` and `internal/frontendconfig` hold no secret manager client and no
  Instagram or Meta credential. They moved here and to `internal/feedconfig`.

## Paging

- `Client.Media` pages through every post of the account. It asks for `POSTS_PER_REQUEST`
  posts per request and sends the `after` cursor of each answer back as a query
  parameter on its own `BaseURL`, until Meta's answer has no `paging.next` or the
  `POST_COUNT` cap is reached.
- Only the presence of `paging.next` is read, never its URL, so an answer cannot send
  the client or the token to another host.
- Redirects are not followed. Every Graph API and picture request goes through
  `Client.do`, which returns a 3xx as `ErrRedirect`, so the picture host list holds for
  the one URL requested.
- `POST_COUNT` 0 means `MaxPosts`, 10000. The page bound is also 10000, enough for 10000
  posts at one post per request.
- The cursor is checked for length and characters and never logged.
- A repeated cursor, a 429, any other 4xx and a cycle past the page bound end the cycle
  with an error kind (`ErrCursorLoop`, `ErrRateLimited`, `ErrRefused`,
  `ErrTooManyPages`).
- Pages are fetched one after the other with `DefaultPagePause` (500 ms) between them.
- `Syncer` replaces the `Feed` only once every page was read. It logs posts, pages,
  failed images and the duration per cycle.
- `paging_test.go` and `bounds_test.go` prove all of it against a fake media edge,
  including a second server in `paging.next` or behind a redirect that must receive no
  request.

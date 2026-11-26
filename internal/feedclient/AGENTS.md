# internal/feedclient

`internal/feedclient` is the API's own client against the feed service (`cmd/feed`).

- `Client` covers the feed service's two routes.
- `Cache` holds the posts and picture bytes it last got back. It is the same caching
  `internal/instagram` itself did before the split.
- `Syncer` polls the feed service on an interval (`FEED_REFRESH_INTERVAL`) and keeps
  the `Cache` current.
- A failed sync is retried after 5 s, doubling up to 5 minutes or the interval, until
  one succeeds (`Syncer.RetryDelay`).
- Every call is bounded by a timeout, so an unreachable or slow feed service can never
  hang a page.
- An empty `Cache` renders no Impressionen section at all, exactly as an absent token
  did before the split. The causes: an unreachable feed service, an unset
  `FEED_SERVICE_URL`, or a feed service with nothing to show.
- It takes at most 10000 posts and 16 MiB of post list per cycle, the same ceiling as
  `POST_COUNT`, whatever the feed service sends.
- `internal/webapp/feed.go` is the one place this package is read from. Its
  `GET /api/v1/instagram` answers every cached post at once, and `InstagramFeed.vue`
  shows them in batches of 12 per click without a further request.

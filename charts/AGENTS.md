# charts

`charts/ostendorf.tech/` is the Helm chart for all three parts. See
`charts/ostendorf.tech/README.md`.

- A Deployment, Service and ServiceAccount each for the frontend, the API and the feed.
- The Ingress or HTTPRoute: `/api` to the API, `/` to the frontend.
- One NetworkPolicy per part.
- `scripts/validate-chart.sh` renders it for its default values and every documented
  non-default combination. It also runs a must-fail set proving every render-time `fail`
  and `required` guard rejects the values it is meant to reject. `task lint:helm` and
  `task validate:kubeconform` share it.
- The chart enforces a single API replica at render time and runs the API with
  `strategy.type: Recreate`. See `cmd/api/AGENTS.md`.
- Image and compose details are in `docs/images-and-compose.md`.

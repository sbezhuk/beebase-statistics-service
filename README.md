# beebase-statistics-service

Dashboard/statistics aggregation service for [BeeBase](https://github.com/sbezhuk/beebase-auth-service#trust-model),
an open-source backend for a beekeeper management application split into
microservices. See [CLAUDE.md](https://github.com/sbezhuk/beebase-auth-service/blob/main/CLAUDE.md)
for the architectural rules this service follows.

This service holds no data of its own. On every request it fetches the
caller's apiaries, hives, inspections, and harvest records from
`beebase-apiary-service`, `beebase-hive-service`,
`beebase-inspection-service`, and `beebase-harvest-service` (forwarding
the caller's own access token to each, so every downstream ownership
check still runs), computes the Dashboard's statistics from that
snapshot, and returns them. There is no cache and no local copy of
anyone else's data, so what it returns is always current - never stale,
never mocked.

Related services: `beebase-auth-service` (users, refresh tokens, JWT
issuing), `beebase-apiary-service`, `beebase-hive-service`,
`beebase-inspection-service`, `beebase-harvest-service`,
`beebase-gateway` (single entry point for clients).

## Quick start

```bash
cp .env.example .env
# point AUTH_JWKS_URL, APIARY_SERVICE_URL, HIVE_SERVICE_URL,
# INSPECTION_SERVICE_URL, and HARVEST_SERVICE_URL at those services, e.g.
#   http://localhost:8081/.well-known/jwks.json
#   http://localhost:8082 / :8083 / :8084 / :8087

make run
```

Verify it's up:

```bash
curl http://localhost:8080/health    # liveness — always 200 while the process is up
curl http://localhost:8080/ready     # readiness — this service has no dependency of its own to probe, so it's the same check as /health

TOKEN=...  # an access_token from auth-service's /api/v1/auth/register or /login

curl http://localhost:8080/api/v1/statistics/overview        -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/statistics/apiaries        -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/statistics/inspections     -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/statistics/activity        -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/statistics/harvest         -H "Authorization: Bearer $TOKEN"
curl http://localhost:8080/api/v1/statistics/needs-attention -H "Authorization: Bearer $TOKEN"
```

The full API surface is documented in [api/openapi.yaml](api/openapi.yaml).

Unlike most other BeeBase services, this repo has no standalone
`docker-compose.yml` — there's nothing local to provision (no database).
To run it alongside the rest of the stack, use `beebase-gateway`'s
docker-compose, which builds every service from sibling checkouts and
routes between them.

### Building with beebase-common

This service depends on the private `github.com/sbezhuk/beebase-common`
module. A local `go build`/`go test` resolves it through your own git
credentials, but building the Docker image needs a GitHub PAT with
`contents:read` on that repo, supplied as a BuildKit secret so it never
ends up in an image layer:

```bash
export BEEBASE_COMMON_GH_TOKEN=$(gh auth token)   # or any read-scoped PAT
docker build --secret id=github_token,env=BEEBASE_COMMON_GH_TOKEN .
```

`beebase-gateway`'s docker-compose already wires this secret through
for you (set `BEEBASE_COMMON_GH_TOKEN` before `docker compose build`).
CI needs the same token as a `BEEBASE_COMMON_GH_TOKEN` GitHub Actions
secret on this repo.

## Configuration

All configuration is via environment variables (see
[.env.example](.env.example) for the full list — it is a template only,
never read by the app, Docker Compose, or deployment tooling; copy it
once to create your real `.env`, which is what actually gets loaded).
Production configuration is generated at deploy time from AWS SSM
Parameter Store (see `beebase-gateway/deploy/deploy.sh`) — `.env.example`
is never used as a fallback, in development or in production.

| Variable                  | Default       | Description                                                       |
| -------------------------- | -------------- | ------------------------------------------------------------------- |
| `APP_ENV`                 | `development`  | `development` or `production`                                     |
| `LOG_LEVEL`                | `info`         | `debug`, `info`, `warn`, `error`                                   |
| `HTTP_PORT`                | `8080`         | Port the HTTP server listens on                                   |
| `HTTP_READ_TIMEOUT`        | `5s`           | Request read timeout                                               |
| `HTTP_WRITE_TIMEOUT`       | `10s`          | Response write timeout                                             |
| `HTTP_IDLE_TIMEOUT`        | `60s`          | Keep-alive idle timeout                                            |
| `HTTP_SHUTDOWN_TIMEOUT`    | `15s`          | Max time to wait for graceful shutdown                             |
| `AUTH_JWKS_URL`            | *(required)*   | auth-service's public key endpoint, used to verify access tokens   |
| `APIARY_SERVICE_URL`       | *(required)*   | apiary-service's base URL                                          |
| `HIVE_SERVICE_URL`         | *(required)*   | hive-service's base URL                                            |
| `INSPECTION_SERVICE_URL`   | *(required)*   | inspection-service's base URL                                      |
| `HARVEST_SERVICE_URL`      | *(required)*   | harvest-service's base URL                                         |

## Project structure

```
cmd/server/                       entry point: wires config, logger, clients, server
api/openapi.yaml                    API contract
internal/
  domain/statistics/                 pure calculation functions; no context, no I/O
  application/statistics/             use cases: Overview, ApiaryStats, InspectionStats, RecentActivity, HarvestStats, NeedsAttention
  platform/
    apiaryclient/                       ApiaryLister implemented against apiary-service
    hiveclient/                         HiveLister implemented against hive-service
    inspectionclient/                   InspectionLister implemented against inspection-service
    harvestclient/                      HarvestLister implemented against harvest-service
  transport/http/                    chi router, health/ready handlers
    statistics/                         Dashboard HTTP handlers, responses
```

logger, JSON response/error helpers, the graceful-shutdown server wrapper,
and JWKS-based access-token verification (`RequireAuth` middleware) all
come from [beebase-common](https://github.com/sbezhuk/beebase-common),
shared by every BeeBase service.

## Endpoints

Six endpoints, one per Dashboard section, so each can be loaded and
retried independently by a client without a bespoke partial-response
envelope:

- `GET /api/v1/statistics/overview` — total apiaries/hives/inspections,
  inspections in the last 7 days/this month/this year, apiaries without
  hives, latest inspection date.
- `GET /api/v1/statistics/apiaries` — per-apiary hive counts (ready to
  render as a distribution chart), the apiary with the most hives.
- `GET /api/v1/statistics/inspections` — inspection counts and windows,
  the hive with the most inspections, a zero-filled 30-day activity chart.
- `GET /api/v1/statistics/activity?limit=` — the caller's most recent
  inspections, newest first (default 10, max 50).
- `GET /api/v1/statistics/harvest` — total harvest records, total
  harvested amount per unit, the latest harvest date, and its product.
  A caller with no harvest records yet gets a valid "200" with all-zero/
  null fields, not an error.
- `GET /api/v1/statistics/needs-attention` — actionable counts:
  apiaries without hives, hives needing inspection (never inspected, or
  last inspected longer ago than the configured warning threshold), and
  the threshold itself, in days - the client reads it from here rather
  than hardcoding it. Backed by inspection-service's `GET
  /api/v1/inspections/hive-status` (see beebase-common/
  inspectionwarning), the same single source of truth hive-service's own
  `?needs_inspection=true` filter reads, so the two are always
  consistent.

## Known tradeoff

`/overview` and `/inspections` each page through the caller's *entire*
inspection history (no caching, no date-range filtering) to compute
counts, windows, and the 30-day chart in one pass. For a single
beekeeper's personal data this is fine; it isn't designed to scale past
that without revisiting.

`/activity` is the exception: inspection-service's default list order is
InspectedAt ascending with no descending variant, so instead of paging
through everything and sorting client-side, it fetches only the tail of
that order - a 1-row probe for the total, then the last `limit`-sized
page (and, when the boundary doesn't land on a whole page, the page
before it too) - at most 3 requests, each of size `limit`, regardless of
how many inspections the caller has.

harvest-service has no endpoint listing every harvest a caller owns in
one call, only `GET /hives/{hiveID}/harvest`, scoped to a single hive -
so `/statistics/harvest` pages through that endpoint once per hive the
caller owns. Same tradeoff, same justification: fine for a beekeeper's
handful of hives, not designed to scale past that.

## Development

```bash
make run    # go run ./cmd/server
make fmt    # go fmt ./...
make vet    # go vet ./...
make test   # unit tests: go test ./...
make lint   # golangci-lint run
make build  # build binary into bin/
```

There's no `test-integration` target — this service has no database or
other stateful dependency to integration-test against; `platform/*client_test.go`
already covers the HTTP-client logic against an `httptest.Server`.

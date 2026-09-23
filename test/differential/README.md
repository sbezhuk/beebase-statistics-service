# Health History HTTP differential harness

This test-only harness compares the current worktree versions of the OLD
inspection-service endpoint and the NEW statistics-service endpoint directly
inside the existing `beebase-dev_default` Compose network. It does not involve
gateway routing and contains no health calculation.

Start the existing development stack, then apply the test-only Compose
override while rebuilding the current statistics worktree:

```sh
cd ../beebase-gateway
BEEBASE_COMMON_GH_TOKEN="$(gh auth token)" \
  docker compose --env-file .env.dev \
  -f docker-compose.dev.yml \
  -f ../beebase-statistics-service/test/differential/docker-compose.override.yml \
  build statistics-service

DIFFERENTIAL_PRO_USER_ID=<local-test-user-uuid> \
  docker compose --env-file .env.dev \
  -f docker-compose.dev.yml \
  -f ../beebase-statistics-service/test/differential/docker-compose.override.yml \
  up -d --no-deps --force-recreate subscription-service statistics-service
```

The fixture is created through the existing authenticated apiary, hive, and
inspection APIs. The comparator receives a JSON array of scenarios. Each
scenario contains `name`, `path`, `token`, and expected `status` fields:

```sh
cat scenarios.json | docker run --rm -i --network beebase-dev_default \
  -v "$PWD":/src:ro -w /src golang:1.27-alpine \
  go run ./test/differential -scenarios /dev/stdin
```

For every scenario it performs the same GET against:

* `http://inspection-service:8080` (OLD)
* `http://statistics-service:8080` (NEW)

It compares HTTP status and the complete decoded JSON value. Object key order
is therefore irrelevant, while arrays, IDs, dates, markers, dimensions,
provenance, and error structures remain part of the comparison.

# Performance Testing

This directory contains Docker-first load testing scaffolding for gateway concurrency, TTFT, and failover validation.

## Usage

Start the application stack first, then run k6 via Docker Compose from this directory:

```bash
docker compose run --rm \
  -e TARGET_URL=http://host.docker.internal:8080 \
  -e API_KEY=your_api_key \
  k6 run /scripts/scenarios/smoke.js
```

For Linux hosts, replace `host.docker.internal` with the reachable gateway address.

## Scenarios

- `smoke.js`: lightweight HTTP sanity check for CI and local verification
- extend with:
  - balanced account load
  - single slow account tail latency
  - proxy failover
  - sticky session hot account
  - queue saturation
  - OpenAI WS prewarm + reuse

## Notes

- Keep the target environment isolated from production.
- Export raw k6 metrics and compare TTFT / latency percentiles before and after scheduler changes.
- Prefer Docker-only execution so contributors do not need local `go`, `k6`, or Redis/Postgres client binaries.

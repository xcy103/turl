[![codecov](https://codecov.io/gh/beihai0xff/turl/graph/badge.svg?token=DPVOTT6MIU)](https://codecov.io/gh/beihai0xff/turl)
[![GitHub Action](https://github.com/beihai0xff/turl/actions/workflows/ci.yml/badge.svg)](https://github.com/beihai0xff/turl/actions/)
[![Go Report Card](https://goreportcard.com/badge/github.com/beihai0xff/turl)](https://goreportcard.com/report/github.com/beihai0xff/turl)

# turl

> 🌐 中文版: [README.zh-CN.md](README.zh-CN.md)

A tiny-URL (short link) service written in Go.

Long-to-short URL conversion is a common need in social media, growth, and ad
campaigns: short links raise click-through rates and help work around keyword or
domain blocking. On platforms with strict length limits (e.g. a 140-character
post), a long URL eats into the budget, so it is shortened to save space.

Beyond being clean and compact, every visit to a short link is routed through the
backend before redirecting. That indirection is a natural place to do asynchronous
tracking for analytics. Typical use cases:

* Conversion tracking for sign-up, favorite, add-to-cart, order, and payment flows;
* Attribution for user shares;
* Reducing character usage.

# Features

## Roadmap

- [x] Distributed ID generator: unique IDs based on TDDL;
- [x] Distributed cache: Redis;
- [x] Local cache: bigcache;
- [x] Storage: MySQL;
- [x] URL 302 redirect;
- [x] URL encoding: Base58;
- [x] Rate limiting: Redis-based and standalone token-bucket limiters;
- [x] Read/write split: run in read-only / write-only / read-write modes;
- [x] Idempotency: repeated generation of the same URL yields the same short link;
- [x] Expiration: short links accept an optional TTL; expired links return 410 Gone
  and are reaped by a background janitor;
- [x] Observability: Prometheus metrics, Grafana dashboards, and OpenTelemetry
  distributed tracing (exported to Jaeger).

# Quick Start

## Run locally

Make sure Docker and Docker Compose are installed, then run:

```shell
make deploy
```

When the terminal prints `turl service containers start successfully`, the service
is up. This mode deploys MySQL and Redis as local storage and cache, and starts two
service nodes — one for read/write and one for read-only.

- Read/write service: [http://localhost:8080](http://localhost:8080) — generates short
  links, updates the remote cache, writes to the database, etc.
- Read-only service: [http://localhost:80](http://localhost:80) — only serves short-link
  redirects; it does not generate short links. In production you can run multiple
  read-only nodes to spread read traffic.
- Swagger: [http://localhost:8080/v1/management/swagger/index.html#/](http://localhost:8080/v1/management/swagger/index.html#/)

### Observability

Each node runs an admin server on a separate port (`:9090`), kept off the public
API so internal endpoints are never exposed to end users:

- `GET /healthz` — liveness (process is up);
- `GET /readyz` — readiness (pings MySQL and Redis), `503` if a dependency is down;
- `GET /metrics` — Prometheus metrics (RED HTTP metrics, cache hit/miss ratio, …);
- `GET /debug/pprof/` — profiling.

`make deploy` also starts a metrics stack:

- Prometheus: [http://localhost:9092](http://localhost:9092) — scrapes both nodes;
- Grafana: [http://localhost:3000](http://localhost:3000) — anonymous access, with a
  pre-provisioned "turl service overview" dashboard (request rate, p95 latency,
  cache hit ratio, short links created).
- Jaeger: [http://localhost:16686](http://localhost:16686) — distributed traces;
  each request links its HTTP span to the downstream cache and database spans, and
  logs are correlated via an injected `trace_id`.

## API

### Create a short link

```shell
curl -X POST http://localhost:8080/v1/management/shorten -H 'Content-Type: application/json' -d '{"long_url": "https://google.com"}'
```

Response:

```json
{"short_url":"http://localhost/24rgcX","long_url":"https://google.com","created_at":"2024-07-08T15:06:26.434Z","deleted_at":null,"expires_at":null,"error":""}
```

Pass an optional `ttl_seconds` to make the link expire. After it expires, visiting
it returns `410 Gone`, and a background janitor removes the row.

```shell
curl -X POST http://localhost:8080/v1/management/shorten -H 'Content-Type: application/json' -d '{"long_url": "https://google.com", "ttl_seconds": 3600}'
```

### Visit a short link

Visiting the short link `http://localhost/24rgcX` redirects to the original long URL
`https://google.com`.

```shell
curl -L http://localhost/24rgcX
```

### Look up a long link

```shell
curl -X GET http://localhost:8080/v1/management/shorten\?long_url\=https://google.com
```

Response:

```json
{"short_url":"http://localhost/24rgcX","long_url":"https://google.com","created_at":"2024-07-08T15:06:26.434Z","deleted_at":null,"error":""}
```

# System Design

## Functional requirements

* **Short-link generation:** given a long URL, produce a unique short URL. Generating
  the same long URL multiple times must always yield the same short URL.
* **Redirection:** a short link redirects to the original long URL via a 302 temporary
  redirect. A temporary redirect keeps search engines crawling the original URL (not the
  short link) and makes it easy to count visits.
* **Rate limiting:** access to short links can be rate-limited per unit time.
* **Expiration:** a short link can carry a TTL; once expired it is no longer resolvable.
* **Deletion:** a short link can be deleted, after which it is no longer resolvable.

## Non-functional requirements

* **High availability:** the service stays up even if a node goes down.
* **High performance:** capable of handling on the order of 100k requests per second.
* **Low latency:** redirects resolve quickly.
* **Scalability:** supports large volumes of short-link generation and access.
* **Reliability:** generation and resolution are correct.

## Capacity estimate

* Assume 100M daily active users.
* Each user writes ~0.1 posts/day, one short link per post → ~10M new short links/day:
  * ~500 bytes per long↔short mapping → ~5GB of new storage per day;
  * 10,000,000 writes/day → 10M / 86400s ≈ 116 → ~116 write QPS on average;
  * assume peak write ≈ 10× average ≈ 1160 QPS, rounded to ~1k write QPS.
* Each user reads ~10 posts/day → ~1B short-link visits/day, read:write ≈ 100:1:
  * ~11,600 reads/second on average, i.e. ~10k/s;
  * assume peak read ≈ 10× average ≈ ~100k/s.
* Cache budget:
  * Short-link traffic is heavily hot-skewed; assume 10% of data drives 99% of traffic.
  * Of 1B daily visits, assume 99% hit a fixed 10% of data → cache ~100M entries → ~50GB.
  * Further, use local cache for the hottest 1% → cache ~1M entries → ~500MB per node.
  * At 99% cache hit ratio, ~1% of 1B visits reach the database → ~10M/day → ~116 DB
    reads/second → ~100 QPS.

In summary:
  * **DB storage:** ~5GB/day, ~5TB over three years.
  * **DB throughput:** ~116 QPS average read+write, ~1k QPS peak.
  * **Cache memory:** ~50GB total, caching ~100M entries.
  * **Local cache:** ~500MB per node, caching ~1M entries.

## More design docs

* [Short-link system design](docs/system-design.md)
* [Base58 encoding](docs/base58-design.md)
* [Distributed ID generator (TDDL)](docs/tddl-design.md)
* [Rate limiter design](docs/rate-limiter-design.md)
* [API benchmark](docs/api-benchmark.md)
* [Database schema](docs/ddl)

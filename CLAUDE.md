# CLAUDE.md

Guidance for AI assistants (and humans) working in this repository.

## Project

`turl` is a tiny-URL (short link) service written in Go. It features a distributed
ID generator (TDDL), two-tier caching (local bigcache + Redis), MySQL storage,
Base58 encoding, rate limiting, and a read/write/readonly split deployment model.

Active work: adding **link expiration** and **observability** (OpenTelemetry tracing +
Prometheus metrics + Grafana + Jaeger). See the project memory for the phased plan.

## Language conventions

- **All code, comments, identifiers, and documentation: English.** The original author
  wrote some comments and docs in Chinese; new and modified code must be English. When
  editing a file with Chinese comments, translate the lines you touch.
- **README and docs: English.** `README.md` is the English source of truth.
  `README.zh-CN.md` keeps the original Chinese version, cross-linked.
- **Commit messages, PR titles/bodies: English.**
- **Reporting to the user at the end of a phase/step (summary + reflections): Chinese.**
  Everything that lands in the repo is English; only the conversational wrap-up is Chinese.

## Commit conventions

- **One commit per completed phase or self-contained step.** Keep history logical and
  readable — each commit should build and pass tests on its own.
- Conventional-commit style prefixes, matching existing history:
  `feat:`, `fix:`, `improve:`, `docs:`, `test:`, `refactor:`, `chore:`.
- Do not bundle unrelated changes into one commit.
- Commit only when a step is genuinely complete (compiles + relevant tests pass).

## Architecture boundaries (rules)

- **Layering:** HTTP handler (`app/turl`) → service (`commandService`/`queryService`)
  → storage (`pkg/storage`) + cache (`pkg/cache`). Do not let an outer concern leak
  inward (e.g. no `gin` types below the handler, no SQL above storage).
- **Read/write split:** write logic lives in `commandService`, read logic in
  `queryService`. `Readonly` servers must never construct write dependencies.
- **Cache is an interface (`cache.Interface`).** Extend behavior with decorators
  (e.g. metrics, expiry checks) rather than editing the proxy/redis/local impls.
- **Cache TTL must never outlive a link's lifetime.** When expiration lands, cache TTL
  must be `min(configured TTL, remaining link lifetime)`.
- **Config is centralized** in `configs/`. New tunables go through `ServerConfig` with
  `validate`/`yaml`/`mapstructure`/`json` tags, and are validated in `Validate()`.
- **Observability runs on a separate admin port**, not the public listener. Keep
  `/metrics`, `/healthz`, `/readyz`, `/debug/pprof` off the user-facing server.
- **Errors propagate; they are not swallowed.** Cache write failures are logged and
  tolerated (best-effort); correctness-critical failures return an error.

## Build, test, lint

```sh
go build ./...                 # compile everything
make test                      # spins up MySQL+Redis via docker compose, runs race tests
golangci-lint run              # lint (config in .golangci.yaml)
make gen/swagger               # regenerate swagger docs after handler/annotation changes
make gen/mock                  # regenerate mocks after changing an interface
```

- Tests use testify + mockery-generated mocks in `internal/tests/mocks/`.
- Keep coverage from regressing (project tracks codecov).
- After changing any exported interface, regenerate mocks; after changing handler
  swagger annotations, regenerate swagger docs.

## Tooling notes

- Go toolchain and `golangci-lint` live in `/opt/homebrew/bin`; `go install` tools
  (`swag`, `mockery`, `gomodifytags`) live in `~/go/bin` (on PATH via `~/.zshrc`).

# Contributing to Phoenix-Logs-RMQ

Thanks for your interest in contributing! Bug reports, fixes, docs and new ideas are all welcome.

By participating you agree to follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Table of contents

- [Ways to contribute](#ways-to-contribute)
- [Development setup](#development-setup)
- [Project layout](#project-layout)
- [Making a change](#making-a-change)
- [Coding guidelines](#coding-guidelines)
- [Commit messages](#commit-messages)
- [Pull request process](#pull-request-process)
- [Releasing (maintainers)](#releasing-maintainers)
- [Getting help](#getting-help)

## Ways to contribute

- **Report a bug.** Use the [bug report form](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/issues/new?template=bug_report.yml).
- **Suggest a feature.** Use the [feature request form](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/issues/new?template=feature_request.yml).
- **Fix an issue.** Issues labelled `good first issue` or `help wanted` are good places to start.
- **Improve docs.** Typos, unclear sections and missing examples all count.

> **Security issues must not be reported publicly.** See [SECURITY.md](SECURITY.md).

## Development setup

**Prerequisites:** Go (version in [`go.mod`](go.mod)), Docker with Compose v2, and `make`.

```bash
git clone https://github.com/<you>/Phoenix-Logs-RMQ.git
cd Phoenix-Logs-RMQ
cp .env.example .env               # local defaults; never commit .env

docker compose up -d rabbitmq clickhouse   # local broker + database
make run                                   # run the worker against them
```

Useful targets (run `make` to list them all):

| Target | What it does |
|---|---|
| `make check` | Everything CI runs: gofmt, vet, race tests, build |
| `make test` | `go test ./... -race` |
| `make cover` | Coverage report in `coverage.html` |
| `make lint` | `golangci-lint run` (optional, needs [golangci-lint](https://golangci-lint.run/usage/install/)) |
| `make docker-up` | Full stack, including the worker, via Docker Compose |

To send test messages, use [`scripts/rabbitmq.http`](scripts/rabbitmq.http) with the VS Code REST Client extension, or the RabbitMQ management UI at <http://localhost:15672> (`phoenix` / `phoenix`).

## Project layout

```text
cmd/phoenix-logs-rmq/   main package: wiring, signals, shutdown order
internal/config/        environment → Config
internal/logger/        zerolog + rotating file
internal/model/         message payloads and table column order
internal/logs/          validation: JSON body → ClickHouse row
internal/rmq/           RabbitMQ connection, consumer and LogsHandler
internal/sink/          size/interval batcher and ClickHouse connection
```

All packages live under `internal/`, so they are not a public Go API and may change between releases.

## Making a change

1. **Open or find an issue first** for anything bigger than a small fix, so we can agree on the approach.
2. Fork the repo and create a branch from `main`, for example `fix/batcher-timeout` or `feat/dead-letter`.
3. Make your change, with tests.
4. Run `make check` and make sure it passes.
5. Add a line under **Unreleased** in [CHANGELOG.md](CHANGELOG.md).
6. Open a pull request.

## Coding guidelines

- **Format.** `gofmt` is enforced by CI.
- **Document everything.** Every package has a package comment (in `doc.go` for multi-file packages). Every exported *and* unexported type, function and method has a doc comment that starts with its name, in the [Go doc comment](https://go.dev/doc/comment) style.
- **License header.** New `.go` files start with the GPL-3.0 header used in the existing files.
- **Errors.** Wrap with context (`fmt.Errorf("prepare batch: %w", err)`). Use `errors.Is` and sentinel errors such as `logs.ErrInvalid` for decisions.
- **Logging.** Use `internal/logger` with structured fields (`.Str("table", t)`), not string formatting. Never log credentials.
- **Delivery semantics are a contract.** Changes to when a message is acked, requeued or dropped must be called out in the PR and the CHANGELOG.
- **Tests.** Use table-driven tests where they fit. Tests must pass with `-race`. Prefer fakes over network calls; see `internal/sink/batcher_test.go`.
- **Dependencies.** Keep them minimal. Justify any new module in the PR description.
- **Config.** A new setting needs a default in `internal/config`, an entry in `.env.example`, and a row in the README configuration table.

## Commit messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat(sink): retry failed batches row by row
fix(rmq): requeue on context deadline
docs: clarify RB_PROCESS_TIMEOUT rule
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`. Mark breaking changes with `!` (for example `feat!:`) and explain them in the commit body.

## Pull request process

1. Fill in the [pull request template](.github/PULL_REQUEST_TEMPLATE.md).
2. CI must be green (gofmt, `go mod tidy`, vet, race tests, Docker build).
3. At least one maintainer approval is required.
4. PRs are squash-merged, so the PR title becomes the commit message. Use the Conventional Commits format for it.

## Releasing (maintainers)

1. Move the **Unreleased** entries in `CHANGELOG.md` under a new version heading.
2. Tag and push: `git tag v1.2.3 && git push origin v1.2.3`.
3. The [Docker workflow](.github/workflows/docker.yml) builds a `linux/amd64` image and pushes `rjyspl/phoenix-logs-rmq:v1.2.3` and `:latest`. Every push to `main` also pushes `:main` and `:latest`. It also syncs [DOCKERHUB.md](DOCKERHUB.md) to the Docker Hub overview.

The workflow needs the `DOCKER_USERNAME` and `DOCKER_PASSWORD` repository secrets.

## Getting help

- 💬 <a href="https://discord.com/invite/2BS7Z4FhJ" target="_blank" rel="noopener noreferrer">Discord</a>
- ✉️ <a href="mailto:connect@yukthi.com">connect@yukthi.com</a>

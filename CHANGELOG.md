# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- First open-source release of Phoenix-Logs-RMQ.
- RabbitMQ consumer for audit (`type: audit`) and mail-flow (`type: fuglu`) log messages on a durable quorum queue.
- Payload validation with drop-on-invalid semantics.
- Size- and interval-based ClickHouse batching. A message is acked only after its row is inserted (at-least-once delivery).
- Ordered graceful shutdown that writes pending rows before exiting.
- `-version` flag, with the version injected at build time.
- Multi-arch Docker image `rjyspl/phoenix-logs-rmq`.
- Docker Compose stack with RabbitMQ and ClickHouse.
- CI (gofmt, `go mod tidy`, vet, race tests, Docker build) and a tag-driven release workflow.

[Unreleased]: https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/commits/main

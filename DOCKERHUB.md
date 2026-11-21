# Phoenix-Logs-RMQ

**Consumes audit and mail-flow logs from RabbitMQ and bulk-inserts them into ClickHouse, with no message loss.**

[GitHub](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ) ·
[Documentation](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ#readme) ·
[Issues](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/issues) ·
<a href="https://discord.com/invite/2BS7Z4FhJ" target="_blank" rel="noopener noreferrer">Discord</a> ·
License: GPL-3.0

---

## Quick reference

| | |
|---|---|
| **Maintained by** | [Yukthi Systems](https://github.com/Yukthi-Systems) |
| **Source** | [github.com/Yukthi-Systems/Phoenix-Logs-RMQ](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ) |
| **Architectures** | `linux/amd64`, `linux/arm64` |
| **Base image** | `alpine:3.22` (static Go binary) |
| **Runs as** | `root` |
| **Where to get help** | <a href="https://discord.com/invite/2BS7Z4FhJ" target="_blank" rel="noopener noreferrer">Discord</a> · [connect@yukthi.com](mailto:connect@yukthi.com) |
| **Where to file issues** | [GitHub Issues](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/issues) |

## Supported tags

| Tag | Meaning |
|---|---|
| `vX.Y.Z` | An exact release. **Recommended for production.** |
| `vX.Y` | The latest patch of a minor version |
| `vX` | The latest release of a major version |
| `latest` | The most recent release |

All tags are listed on the [Tags](https://hub.docker.com/r/rjyspl/phoenix-logs-rmq/tags) page. Release notes are in the [CHANGELOG](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/blob/main/CHANGELOG.md).

## What it does

1. Consumes a durable **quorum** queue (default `v3_logs`) with high concurrency.
2. Routes each message by its AMQP `type` header: `audit` goes to `audit_logs`, and `fuglu` (the default) goes to `mail_flow_logs`.
3. Validates the JSON body. Invalid messages are logged and dropped.
4. Batches rows and writes them to ClickHouse when a batch reaches **500 rows or 2 s**, whichever comes first.
5. **Acks only after the insert succeeds.** Failed inserts and timeouts are requeued, giving at-least-once delivery.
6. On `SIGTERM`, writes the remaining rows before exiting.

## How to use this image

### docker run

```bash
docker run -d --name phoenix-logs-rmq \
  --restart unless-stopped \
  -e RB_HOST=rabbitmq.example.internal \
  -e RB_USERNAME=phoenix \
  -e RB_PASSWD=change-me \
  -e CLICK_HOUSE_HOST=clickhouse.example.internal \
  -e CLICK_HOUSE_PASSWORD=change-me \
  -v phoenix-logs:/app/logs \
  rjyspl/phoenix-logs-rmq:vX.Y.Z
```

Or keep your settings in an env file:

```bash
docker run -d --env-file ./phoenix.env rjyspl/phoenix-logs-rmq:vX.Y.Z
```

### Docker Compose

```yaml
services:
  phoenix-logs-rmq:
    image: rjyspl/phoenix-logs-rmq:vX.Y.Z
    restart: unless-stopped
    environment:
      RB_HOST: rabbitmq
      RB_USERNAME: phoenix
      RB_PASSWD: ${RB_PASSWD}
      CLICK_HOUSE_HOST: clickhouse
      CLICK_HOUSE_PASSWORD: ${CLICK_HOUSE_PASSWORD}
      LOG_CONSOLE: "true"
    volumes:
      - phoenix-logs:/app/logs

volumes:
  phoenix-logs:
```

A complete local stack (RabbitMQ, ClickHouse with an example schema, and the worker) is available in the [repository](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/blob/main/docker-compose.yaml):

```bash
git clone https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ.git && cd Phoenix-Logs-RMQ
docker compose up --build
```

### Check the version

```bash
docker run --rm rjyspl/phoenix-logs-rmq:vX.Y.Z -version
```

## Environment variables

### Required

| Variable | Description |
|---|---|
| `RB_HOST` | RabbitMQ host |
| `RB_USERNAME` | RabbitMQ user |
| `RB_PASSWD` | RabbitMQ password |
| `CLICK_HOUSE_HOST` | ClickHouse host |
| `CLICK_HOUSE_PASSWORD` | ClickHouse password |

### Optional

| Variable | Default | Description |
|---|---|---|
| `RB_PORT` | `5672` | AMQP port |
| `RB_VHOST` | `/` | Virtual host |
| `RB_CONSUME_QUEUE` | `v3_logs` | Queue to consume (durable, quorum) |
| `RB_CONSUMER_TAG` | broker-assigned | Consumer name |
| `RB_WORKER_COUNT` | `2 × BATCH_SIZE` | Concurrent handlers |
| `RB_PREFETCH` | `2 × BATCH_SIZE` | QoS prefetch |
| `AMQPR_CON_DELAY` | `5s` | Reconnect delay |
| `RB_PROCESS_TIMEOUT` | `30s` | Per-message timeout. Must be greater than flush interval + insert timeout. |
| `MSG_TYPE_AUDIT` | `audit` | `type` header value for audit messages |
| `MSG_TYPE_MAIL_FLOW` | `fuglu` | `type` header value for mail-flow messages (default when missing) |
| `BATCH_SIZE` | `500` | Maximum rows per insert |
| `BATCH_FLUSH_INTERVAL` | `2s` | Longest a partial batch waits |
| `CLICK_HOUSE_PORT` | `8123` | `8123` for HTTP, `9000` for native |
| `CLICK_HOUSE_HTTP` | `true` | `false` selects the native protocol |
| `CLICK_HOUSE_USERNAME` | `default` | ClickHouse user |
| `CLICK_HOUSE_DATABASE` | `default` | Database |
| `CLICK_HOUSE_AUDIT_LOGS_TABLE` | `audit_logs` | Audit table |
| `CLICK_HOUSE_MAIL_FLOW_LOGS_TABLE` | `mail_flow_logs` | Mail-flow table |
| `CLICK_HOUSE_INSERT_TIMEOUT` | `15s` | Timeout per insert |
| `LOG_LEVEL` | `info` | `trace`, `debug`, `info`, `warn`, `error` |
| `LOG_CONSOLE` | `false` | Set to `true` to also log to stderr (`docker logs`) |
| `LOG_FILE` | `/app/logs/app.log` | Rotating JSON log file |
| `LOG_MAX_SIZE_MB` / `LOG_MAX_BACKUPS` / `LOG_MAX_AGE_DAYS` / `LOG_COMPRESS` | `50` / `5` / `14` / `true` | Log rotation |

The full reference and the message formats are in the [README](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ#configuration).

## Volumes

| Path | Purpose |
|---|---|
| `/app/logs` | Rotating JSON log files. Mount a named volume or a host directory. |

## Health check

The image includes a `HEALTHCHECK` that verifies the worker process is running. The worker exits with a non-zero status (and the container restarts, given a restart policy) if it cannot reach ClickHouse at startup or a required variable is missing.

## Security

- No ports are exposed.
- Pass credentials with your orchestrator's secret store. Never bake them into an image.
- Report vulnerabilities privately. See [SECURITY.md](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/blob/main/SECURITY.md).

## License

Phoenix-Logs-RMQ is licensed under the [GNU General Public License v3.0](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/blob/main/LICENSE).

As with all Docker images, these also contain other software (such as the Alpine base image and its packages) that may be under other licenses. Users of this image are responsible for making sure their use complies with all relevant licenses.

---

© 2026 Yukthi Systems Private Limited · <a href="https://discord.com/invite/2BS7Z4FhJ" target="_blank" rel="noopener noreferrer">Discord</a> · [connect@yukthi.com](mailto:connect@yukthi.com)

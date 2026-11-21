# Security Policy

## Supported versions

Security fixes are released for the latest minor version only.

| Version | Supported |
|---|---|
| Latest `v1.x` release | ✅ |
| Older releases | ❌ |

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues, pull requests, or Discord.**

Report them privately through either channel:

1. **GitHub private vulnerability reporting** (preferred): [open a security advisory](https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ/security/advisories/new).
2. **Email:** <a href="mailto:connect@yukthi.com">connect@yukthi.com</a> with the subject `[SECURITY] Phoenix-Logs-RMQ`.

Please include:

- the affected version or image tag,
- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- any suggested fix or mitigation.

### What to expect

- We acknowledge your report within **3 business days**.
- We send an initial assessment within **10 business days**.
- We work with you on a fix and a coordinated disclosure date, and credit you in the release notes unless you prefer otherwise.

## Deployment hardening

Phoenix-Logs-RMQ handles credentials for RabbitMQ and ClickHouse. When you deploy it:

- Provide credentials through your orchestrator's secret store (Kubernetes Secrets, Docker secrets, Vault, and so on). Do not bake a `.env` file into an image.
- Give the RabbitMQ user only `read` permission on the consumed queue, and the ClickHouse user only `INSERT` on the two log tables.
- Message bodies, including email addresses and subjects, are written to the log when a message is dropped as invalid. Treat log files as sensitive data.

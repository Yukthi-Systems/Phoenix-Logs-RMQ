// Copyright (C) 2026 Yukthi Systems Private Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License version 3
// as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// version 3 along with this program. If not, see
// <https://www.gnu.org/licenses/>.

// Command phoenix-logs-rmq consumes audit and mail-flow log messages from a
// RabbitMQ queue, validates them, and bulk-inserts them into ClickHouse.
//
// All configuration is read from environment variables (or a .env file in
// the working directory); see the README for the full list.
//
// Usage:
//
//	phoenix-logs-rmq            # run the worker
//	phoenix-logs-rmq -version   # print the version and exit
//
// The process stops on SIGINT or SIGTERM. Shutdown is ordered so that no
// acknowledged message is lost: the consumer stops taking deliveries and
// waits for in-flight handlers, the batchers write any rows they still hold,
// and only then are the RabbitMQ and ClickHouse connections closed.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os/signal"
	"syscall"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/config"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logger"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/model"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/rmq"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/sink"
)

// version is the release version of the binary. It is "dev" for local
// builds and is set at link time for releases:
//
//	go build -ldflags "-X main.version=v1.2.3" ./cmd/phoenix-logs-rmq
var version = "dev"

// main wires configuration, logging, ClickHouse, the batchers and the
// RabbitMQ consumer together, then blocks until a shutdown signal arrives.
func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	config.Load()
	cfg := config.Cfg
	logger.Init(logger.Options{
		File:       cfg.LogFile,
		MaxSizeMB:  cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAgeDays: cfg.LogMaxAgeDays,
		Compress:   cfg.LogCompress,
		Level:      cfg.LogLevel,
		Console:    cfg.LogShowConsole,
	})
	logger.Info().Str("version", version).Msg("starting phoenix-logs-rmq")

	if missing := cfg.MissingRequired(); len(missing) > 0 {
		logger.Fatal().Strs("env", missing).Msg("required environment variables are not set")
	}
	if cfg.AuditType == cfg.MailFlowType {
		logger.Fatal().Str("type", cfg.AuditType).
			Msg("MSG_TYPE_AUDIT and MSG_TYPE_MAIL_FLOW must be different")
	}
	logger.Info().Str("audit_type", cfg.AuditType).Str("mail_flow_type", cfg.MailFlowType).
		Msg("message type header values")
	if cfg.ProcessTimeout <= cfg.FlushInterval+cfg.InsertTimeout {
		logger.Warn().
			Dur("process_timeout", cfg.ProcessTimeout).
			Dur("flush_interval", cfg.FlushInterval).
			Dur("insert_timeout", cfg.InsertTimeout).
			Msg("RB_PROCESS_TIMEOUT <= BATCH_FLUSH_INTERVAL + CLICK_HOUSE_INSERT_TIMEOUT; expect requeues and duplicates")
	}

	if err := run(cfg); err != nil {
		// Fatal exits with status 1 after the event is written.
		logger.Fatal().Err(err).Msg("worker stopped with error")
	}
	logger.Info().Msg("shutdown complete")
}

// run starts the worker and blocks until it has shut down cleanly. It
// returns an error only for failures during startup; errors while consuming
// are logged and handled by requeueing.
func run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ch, err := sink.Connect(ctx, sink.Options{
		Host:     cfg.ClickHouseHost,
		Port:     cfg.ClickHousePort,
		Username: cfg.ClickHouseUsername,
		Password: cfg.ClickHousePassword,
		Database: cfg.ClickHouseDatabase,
		HTTP:     cfg.ClickHouseHTTP,
	})
	if err != nil {
		return fmt.Errorf("clickhouse connect: %w", err)
	}
	defer func() { _ = ch.Close() }()

	auditB := sink.New(ch, cfg.ClickHouseAuditTable, model.AuditLogsColumns,
		cfg.BatchSize, cfg.FlushInterval, cfg.InsertTimeout)
	mailB := sink.New(ch, cfg.ClickHouseMailFlowTable, model.MailFlowLogsColumns,
		cfg.BatchSize, cfg.FlushInterval, cfg.InsertTimeout)
	// Closing twice is a no-op, so these only matter on an early return.
	defer auditB.Close()
	defer mailB.Close()

	rcfg := rmq.Config{
		URL:               amqpURL(cfg.RabbitUsername, cfg.RabbitPassword, cfg.RabbitHost, cfg.RabbitPort, cfg.RabbitVHost),
		ConsumeQueue:      cfg.RabbitConsumeQueue,
		ConsumerTag:       cfg.ConsumerTag,
		Prefetch:          cfg.RabbitPrefetch,
		Concurrency:       cfg.WorkerCount,
		ReconnectInterval: cfg.AMQPReconnectDelay,
		ProcessTimeout:    cfg.ProcessTimeout,
	}
	conn, err := rmq.NewConnection(rcfg)
	if err != nil {
		return fmt.Errorf("rabbitmq connect: %w", err)
	}
	defer conn.Close()

	consumer, err := rmq.NewConsumer(conn, &rmq.LogsHandler{
		Audit:        auditB,
		MailFlow:     mailB,
		AuditType:    cfg.AuditType,
		MailFlowType: cfg.MailFlowType,
	}, rcfg)
	if err != nil {
		return fmt.Errorf("rabbitmq consumer setup: %w", err)
	}

	go func() {
		<-ctx.Done()
		logger.Info().Msg("received signal to stop the process")
		// Close waits for running handlers. They are blocked in
		// Batcher.Add, and the batchers keep flushing on their interval,
		// so this returns once the in-flight rows are written.
		consumer.Close()
	}()

	logger.Info().
		Str("queue", rcfg.ConsumeQueue).
		Int("batch_size", cfg.BatchSize).
		Int("concurrency", rcfg.Concurrency).
		Int("prefetch", rcfg.Prefetch).
		Msg("starting to consume messages")
	if err := consumer.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("consumer stopped with error")
	}

	// Order matters: write the remaining rows before the deferred calls
	// close the RabbitMQ and ClickHouse connections.
	auditB.Close()
	mailB.Close()
	return nil
}

// amqpURL builds an AMQP connection URL from its parts. Credentials and the
// virtual host are escaped, so a vhost of "/" becomes "%2F" as RabbitMQ
// expects.
func amqpURL(user, pass, host string, port int, vhost string) string {
	return (&url.URL{
		Scheme:  "amqp",
		User:    url.UserPassword(user, pass),
		Host:    fmt.Sprintf("%s:%d", host, port),
		Path:    "/" + vhost,
		RawPath: "/" + url.PathEscape(vhost),
	}).String()
}

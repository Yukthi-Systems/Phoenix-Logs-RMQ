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

package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/model"

	"github.com/joho/godotenv"
)

// Config holds every tunable the worker needs at runtime: RabbitMQ
// connection and consumer settings, batching, ClickHouse, and logging.
// It is populated by Load; the environment variable for each field is
// given in parentheses.
type Config struct {
	// RabbitHost is the broker hostname or IP (RB_HOST, required).
	RabbitHost string
	// RabbitPort is the broker AMQP port (RB_PORT, default 5672).
	RabbitPort int
	// RabbitVHost is the broker virtual host (RB_VHOST, default "/").
	RabbitVHost string
	// RabbitUsername is the broker login username (RB_USERNAME, required).
	RabbitUsername string
	// RabbitPassword is the broker login password (RB_PASSWD, required).
	RabbitPassword string

	// RabbitConsumeQueue is the queue to consume from (RB_CONSUME_QUEUE,
	// default "v3_logs"). It is declared durable and quorum.
	RabbitConsumeQueue string
	// ConsumerTag identifies this consumer to the broker (RB_CONSUMER_TAG).
	// Empty lets the broker assign one.
	ConsumerTag string
	// WorkerCount is the number of concurrent message handlers
	// (RB_WORKER_COUNT, default 2 x BatchSize).
	WorkerCount int
	// RabbitPrefetch is the per-consumer QoS prefetch count (RB_PREFETCH,
	// default 2 x BatchSize).
	RabbitPrefetch int
	// AuditType is the "type" header value that marks an audit message
	// (MSG_TYPE_AUDIT, default "audit"). It is lower-cased, because the
	// header is matched case-insensitively.
	AuditType string
	// MailFlowType is the "type" header value that marks a mail-flow
	// message, and the type assumed when the header is missing
	// (MSG_TYPE_MAIL_FLOW, default "fuglu"). It is lower-cased.
	MailFlowType string
	// AMQPReconnectDelay is the delay between reconnect attempts
	// (AMQPR_CON_DELAY, default 5s).
	AMQPReconnectDelay time.Duration
	// ProcessTimeout bounds one message's handling, including waiting for
	// its batch to be inserted (RB_PROCESS_TIMEOUT, default 30s). It must
	// be greater than FlushInterval + InsertTimeout.
	ProcessTimeout time.Duration

	// BatchSize is the maximum number of rows per ClickHouse insert
	// (BATCH_SIZE, default 500).
	BatchSize int
	// FlushInterval is the longest a partial batch waits before it is
	// written (BATCH_FLUSH_INTERVAL, default 2s).
	FlushInterval time.Duration

	// ClickHouseHost is the ClickHouse hostname or IP (CLICK_HOUSE_HOST, required).
	ClickHouseHost string
	// ClickHousePort is the ClickHouse port (CLICK_HOUSE_PORT, default 8123).
	// Use 8123 with HTTP and 9000 with the native protocol.
	ClickHousePort int
	// ClickHouseHTTP selects the HTTP interface instead of the native
	// protocol (CLICK_HOUSE_HTTP, default true).
	ClickHouseHTTP bool
	// ClickHouseUsername is the ClickHouse user (CLICK_HOUSE_USERNAME, default "default").
	ClickHouseUsername string
	// ClickHousePassword is the ClickHouse password (CLICK_HOUSE_PASSWORD, required).
	ClickHousePassword string
	// ClickHouseDatabase is the target database (CLICK_HOUSE_DATABASE, default "default").
	ClickHouseDatabase string
	// ClickHouseAuditTable is the audit log table
	// (CLICK_HOUSE_AUDIT_LOGS_TABLE, default "audit_logs").
	ClickHouseAuditTable string
	// ClickHouseMailFlowTable is the mail-flow log table
	// (CLICK_HOUSE_MAIL_FLOW_LOGS_TABLE, default "mail_flow_logs").
	ClickHouseMailFlowTable string
	// InsertTimeout bounds a single ClickHouse insert
	// (CLICK_HOUSE_INSERT_TIMEOUT, default 15s).
	InsertTimeout time.Duration

	// LogFile is the path of the rotating log file (LOG_FILE, default "./logs/app.log").
	LogFile string
	// LogMaxSizeMB is the size in megabytes at which the log file is
	// rotated (LOG_MAX_SIZE_MB, default 50).
	LogMaxSizeMB int
	// LogMaxBackups is the number of rotated log files to keep
	// (LOG_MAX_BACKUPS, default 5).
	LogMaxBackups int
	// LogMaxAgeDays is the number of days to keep rotated log files
	// (LOG_MAX_AGE_DAYS, default 14).
	LogMaxAgeDays int
	// LogCompress gzip-compresses rotated log files (LOG_COMPRESS, default true).
	LogCompress bool
	// LogLevel is the minimum level to emit, such as "debug" or "info"
	// (LOG_LEVEL, default "debug").
	LogLevel string
	// LogShowConsole also writes logs to stderr (LOG_CONSOLE, default false).
	LogShowConsole bool
}

// Cfg is the process-wide configuration. It is nil until Load is called
// and must not be read before then.
var Cfg *Config

// Load reads a .env file from the working directory if one exists (a
// missing file is ignored), then populates Cfg from the environment,
// applying the defaults documented on each Config field. Variables that are
// already set in the environment take precedence over the .env file.
//
// Load must be called once during startup, before any reads of Cfg. It is
// not safe to call concurrently with reads of Cfg.
func Load() {
	_ = godotenv.Load()
	Cfg = FromEnv()
}

// FromEnv builds a Config from the current environment only, without
// reading a .env file or touching Cfg. It is useful in tests.
func FromEnv() *Config {
	batch := getint("BATCH_SIZE", 500)
	return &Config{
		RabbitHost:     getenv("RB_HOST", ""),
		RabbitPort:     getint("RB_PORT", 5672),
		RabbitVHost:    getenv("RB_VHOST", "/"),
		RabbitUsername: getenv("RB_USERNAME", ""),
		RabbitPassword: getenv("RB_PASSWD", ""),

		RabbitConsumeQueue: getenv("RB_CONSUME_QUEUE", "v3_logs"),
		ConsumerTag:        getenv("RB_CONSUMER_TAG", ""),
		// Each handler goroutine blocks until its batch is inserted, so
		// filling a batch needs at least BatchSize handlers and BatchSize
		// unacked messages. Two tables share them, hence 2x.
		WorkerCount:        getint("RB_WORKER_COUNT", 2*batch),
		RabbitPrefetch:     getint("RB_PREFETCH", 2*batch),
		AuditType:          strings.ToLower(strings.TrimSpace(getenv("MSG_TYPE_AUDIT", model.TypeAudit))),
		MailFlowType:       strings.ToLower(strings.TrimSpace(getenv("MSG_TYPE_MAIL_FLOW", model.TypeMailFlow))),
		AMQPReconnectDelay: getdur("AMQPR_CON_DELAY", 5*time.Second),
		ProcessTimeout:     getdur("RB_PROCESS_TIMEOUT", 30*time.Second),

		BatchSize:     batch,
		FlushInterval: getdur("BATCH_FLUSH_INTERVAL", 2*time.Second),

		ClickHouseHost:          getenv("CLICK_HOUSE_HOST", ""),
		ClickHousePort:          getint("CLICK_HOUSE_PORT", 8123),
		ClickHouseHTTP:          getbool("CLICK_HOUSE_HTTP", true),
		ClickHouseUsername:      getenv("CLICK_HOUSE_USERNAME", "default"),
		ClickHousePassword:      getenv("CLICK_HOUSE_PASSWORD", ""),
		ClickHouseDatabase:      getenv("CLICK_HOUSE_DATABASE", "default"),
		ClickHouseAuditTable:    getenv("CLICK_HOUSE_AUDIT_LOGS_TABLE", "audit_logs"),
		ClickHouseMailFlowTable: getenv("CLICK_HOUSE_MAIL_FLOW_LOGS_TABLE", "mail_flow_logs"),
		InsertTimeout:           getdur("CLICK_HOUSE_INSERT_TIMEOUT", 15*time.Second),

		LogFile:        getenv("LOG_FILE", "./logs/app.log"),
		LogMaxSizeMB:   getint("LOG_MAX_SIZE_MB", 50),
		LogMaxBackups:  getint("LOG_MAX_BACKUPS", 5),
		LogMaxAgeDays:  getint("LOG_MAX_AGE_DAYS", 14),
		LogCompress:    getbool("LOG_COMPRESS", true),
		LogLevel:       getenv("LOG_LEVEL", "debug"),
		LogShowConsole: getbool("LOG_CONSOLE", false),
	}
}

// MissingRequired returns the names of required environment variables that
// have no value in c, in a stable order. An empty result means c is complete.
func (c *Config) MissingRequired() []string {
	required := []struct {
		env   string
		value string
	}{
		{"RB_HOST", c.RabbitHost},
		{"RB_USERNAME", c.RabbitUsername},
		{"RB_PASSWD", c.RabbitPassword},
		{"CLICK_HOUSE_HOST", c.ClickHouseHost},
		{"CLICK_HOUSE_PASSWORD", c.ClickHousePassword},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.env)
		}
	}
	return missing
}

// getenv returns the environment variable named key, or def if it is unset
// or empty.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getint returns the environment variable named key parsed as an int, or
// def if it is unset, empty, or not a valid integer.
func getint(key string, def int) int {
	i, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return i
}

// getbool returns the environment variable named key parsed with
// strconv.ParseBool, or def if it is unset, empty, or not a valid boolean.
func getbool(key string, def bool) bool {
	b, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return def
	}
	return b
}

// getdur returns the environment variable named key parsed with
// time.ParseDuration (for example "5s" or "1m"), or def if it is unset,
// empty, or not a valid duration.
func getdur(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return def
	}
	return d
}

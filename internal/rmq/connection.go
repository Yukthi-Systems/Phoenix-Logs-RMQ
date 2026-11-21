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

package rmq

import (
	"time"

	rabbitmq "github.com/wagslane/go-rabbitmq"
)

// Config holds the settings shared by NewConnection and NewConsumer: where
// to connect, which queue to consume, and how much work to do concurrently.
type Config struct {
	// URL is the AMQP connection string, e.g. "amqp://user:pass@host:5672/vhost".
	URL string
	// ConsumeQueue is the name of the queue to consume messages from.
	ConsumeQueue string
	// ConsumerTag identifies this consumer to the broker. An empty string
	// lets the broker generate one.
	ConsumerTag string
	// Prefetch is the per-consumer QoS prefetch count.
	Prefetch int
	// Concurrency is the number of goroutines processing deliveries concurrently.
	Concurrency int
	// ReconnectInterval is the base delay between connection retry attempts.
	ReconnectInterval time.Duration
	// ProcessTimeout bounds each Handler.Process call.
	ProcessTimeout time.Duration
}

// Connection wraps a single AMQP connection to a RabbitMQ broker.
type Connection struct {
	// Conn is the underlying go-rabbitmq connection.
	Conn *rabbitmq.Conn
}

// NewConnection dials the broker at cfg.URL and returns an open Connection.
// If the connection drops later, it is re-established automatically,
// waiting cfg.ReconnectInterval between attempts. The caller must call
// Close when done with it.
func NewConnection(cfg Config) (*Connection, error) {
	conn, err := rabbitmq.NewConn(
		cfg.URL,
		rabbitmq.WithConnectionOptionsLogger(rabbitLogger{}),
		rabbitmq.WithConnectionOptionsBaseReconnectInterval(cfg.ReconnectInterval),
	)
	if err != nil {
		return nil, err
	}

	return &Connection{Conn: conn}, nil
}

// Close closes the underlying AMQP connection. Any error from closing is
// discarded, since it is only called during shutdown.
func (c *Connection) Close() {
	_ = c.Conn.Close()
}

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
	"context"

	rabbitmq "github.com/wagslane/go-rabbitmq"
)

// Consumer drives a durable queue subscription, dispatching each delivery
// to a Handler and acknowledging, rejecting, or requeueing it based on the
// Action the Handler returns.
type Consumer struct {
	consumer *rabbitmq.Consumer
	handler  Handler
	cfg      Config
}

// NewConsumer declares cfg.ConsumeQueue as a durable quorum queue on conn
// and sets up a consumer with cfg.Prefetch QoS and cfg.Concurrency
// parallel handlers. Deliveries are not processed until Start is called.
//
// Declaring an existing queue with different arguments (for example a
// classic queue) fails with PRECONDITION_FAILED.
func NewConsumer(
	conn *Connection,
	handler Handler,
	cfg Config,
) (*Consumer, error) {

	consumer, err := rabbitmq.NewConsumer(
		conn.Conn,
		cfg.ConsumeQueue,
		rabbitmq.WithConsumerOptionsQueueDurable,
		rabbitmq.WithConsumerOptionsQueueQuorum,
		rabbitmq.WithConsumerOptionsQOSPrefetch(cfg.Prefetch),
		rabbitmq.WithConsumerOptionsConcurrency(cfg.Concurrency),
		rabbitmq.WithConsumerOptionsConsumerName(cfg.ConsumerTag),
		rabbitmq.WithConsumerOptionsLogger(rabbitLogger{}),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		consumer: consumer,
		handler:  handler,
		cfg:      cfg,
	}, nil
}

// Start begins consuming deliveries and blocks until Close is called. Each
// delivery is passed to the Handler with a context derived from ctx and
// bounded by cfg.ProcessTimeout, so cancelling ctx also cancels in-flight
// handlers.
func (c *Consumer) Start(ctx context.Context) error {

	return c.consumer.Run(func(d rabbitmq.Delivery) rabbitmq.Action {
		processCtx, cancel := context.WithTimeout(ctx, c.cfg.ProcessTimeout)
		defer cancel()
		return c.handler.Process(processCtx, d)
	})
}

// Close stops taking new deliveries, waits for running handlers to
// return, and causes a blocked Start call to return.
func (c *Consumer) Close() {
	c.consumer.Close()
}

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

// Package rmq connects to RabbitMQ and consumes a queue, using
// github.com/wagslane/go-rabbitmq for automatic reconnection.
//
// A Consumer passes each delivery to a Handler. The Action the Handler
// returns (Ack, NackRequeue or NackDiscard) is sent to the broker by the
// library, so handlers never acknowledge messages themselves.
//
// LogsHandler is the Handler used by the worker. It routes each message by
// its "type" header to the matching parser and ClickHouse batcher, and acks
// it only after the batch holding its row has been inserted.
package rmq

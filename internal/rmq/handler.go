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
	"errors"
	"fmt"
	"strings"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logger"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logs"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/model"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/sink"
	rabbitmq "github.com/wagslane/go-rabbitmq"
)

// Handler processes a single delivery from the consumed queue and reports
// how the Consumer should acknowledge it back to the broker. Implementations
// must be safe for concurrent use.
type Handler interface {
	Process(ctx context.Context, d rabbitmq.Delivery) rabbitmq.Action
}

// LogsHandler routes audit and mail-flow messages to their ClickHouse
// batchers. It is safe for concurrent use.
type LogsHandler struct {
	// Audit receives rows from messages whose type is AuditType.
	Audit *sink.Batcher
	// MailFlow receives rows from messages whose type is MailFlowType.
	MailFlow *sink.Batcher
	// AuditType is the lower-cased "type" header value of audit messages.
	// Empty means model.TypeAudit.
	AuditType string
	// MailFlowType is the lower-cased "type" header value of mail-flow
	// messages, also used when the header is missing. Empty means
	// model.TypeMailFlow.
	MailFlowType string
}

// Process handles one delivery:
//
//  1. Read the "type" header, lower-cased. A missing header defaults to
//     MailFlowType.
//  2. Parse and validate the body for that type.
//  3. Hand the row to that table's batcher and wait for the bulk insert.
//  4. Ack only after the insert succeeds.
//
// Invalid messages and unknown types are logged and acked, so they are
// dropped rather than redelivered forever. A failed insert or a timeout
// returns NackRequeue, so the message is redelivered instead of lost.
func (h *LogsHandler) Process(ctx context.Context, d rabbitmq.Delivery) rabbitmq.Action {
	auditType := h.AuditType
	if auditType == "" {
		auditType = model.TypeAudit
	}
	mailFlowType := h.MailFlowType
	if mailFlowType == "" {
		mailFlowType = model.TypeMailFlow
	}
	msgType := headerString(d.Headers, "type", mailFlowType)

	var (
		row     []any
		err     error
		batcher *sink.Batcher
	)
	switch msgType {
	case auditType:
		row, err = logs.ParseAudit(d.Body)
		batcher = h.Audit
	case mailFlowType:
		row, err = logs.ParseMailFlow(d.Body)
		batcher = h.MailFlow
	default:
		logger.Error().Str("type", msgType).Interface("headers", d.Headers).
			Bytes("body", d.Body).Msg("unknown message type; dropping")
		return rabbitmq.Ack
	}
	if err != nil {
		if errors.Is(err, logs.ErrInvalid) {
			logger.Error().Err(err).Str("type", msgType).Bytes("body", d.Body).
				Msg("invalid log message; dropping")
			return rabbitmq.Ack
		}
		logger.Error().Err(err).Msg("unexpected parse error; requeueing")
		return rabbitmq.NackRequeue
	}

	if err := batcher.Add(ctx, row); err != nil {
		logger.Warn().Err(err).Str("type", msgType).Msg("insert not confirmed; requeueing")
		return rabbitmq.NackRequeue
	}
	return rabbitmq.Ack
}

// headerString reads an AMQP header as a lower-cased string. amqp091 gives
// a header as string or []byte depending on the publisher, so both are
// handled. Any other type is formatted with fmt.Sprint. A missing or nil
// header returns def.
func headerString(h map[string]any, key, def string) string {
	v, ok := h[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case string:
		return strings.ToLower(t)
	case []byte:
		return strings.ToLower(string(t))
	default:
		return strings.ToLower(fmt.Sprint(t))
	}
}

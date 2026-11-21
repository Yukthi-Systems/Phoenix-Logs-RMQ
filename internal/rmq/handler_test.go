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
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wagslane/go-rabbitmq"
)

func TestHeaderString(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]any
		want    string
	}{
		{"nil headers", nil, "def"},
		{"missing key", map[string]any{"other": "x"}, "def"},
		{"nil value", map[string]any{"type": nil}, "def"},
		{"bytes", map[string]any{"type": []byte("AUDIT")}, "audit"},
		{"string", map[string]any{"type": "Fuglu"}, "fuglu"},
		{"other type", map[string]any{"type": int32(7)}, "7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerString(tt.headers, "type", "def"); got != tt.want {
				t.Errorf("headerString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestProcessDropsWithoutInserting covers the paths that must ack (drop) a
// message without touching a batcher. The batchers are nil, so reaching
// one would panic.
func TestProcessDropsWithoutInserting(t *testing.T) {
	h := &LogsHandler{}
	tests := []struct {
		name    string
		headers amqp.Table
		body    string
	}{
		{"unknown type", amqp.Table{"type": "something-else"}, `{}`},
		{"invalid audit body", amqp.Table{"type": "audit"}, `{"user_id":1}`},
		{"invalid mail-flow body (default type)", nil, `not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := rabbitmq.Delivery{Delivery: amqp.Delivery{Headers: tt.headers, Body: []byte(tt.body)}}
			if got := h.Process(context.Background(), d); got != rabbitmq.Ack {
				t.Errorf("Process() = %v, want Ack", got)
			}
		})
	}
}

// TestProcessCustomTypes checks that configured type values replace the
// defaults: the old "audit" value becomes unknown, and the custom values
// route to their parsers (invalid bodies, so no batcher is reached).
func TestProcessCustomTypes(t *testing.T) {
	h := &LogsHandler{AuditType: "aud", MailFlowType: "mail"}
	tests := []struct {
		name    string
		headers amqp.Table
		body    string
	}{
		{"default audit value is now unknown", amqp.Table{"type": "audit"}, `{}`},
		{"custom audit type", amqp.Table{"type": "AUD"}, `{"user_id":1}`},
		{"custom mail-flow type", amqp.Table{"type": "mail"}, `not json`},
		{"missing header uses custom mail-flow type", nil, `not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := rabbitmq.Delivery{Delivery: amqp.Delivery{Headers: tt.headers, Body: []byte(tt.body)}}
			if got := h.Process(context.Background(), d); got != rabbitmq.Ack {
				t.Errorf("Process() = %v, want Ack", got)
			}
		})
	}
}

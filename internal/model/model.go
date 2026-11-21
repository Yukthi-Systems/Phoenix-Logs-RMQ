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

// Package model defines the message payloads consumed from RabbitMQ and the
// ClickHouse column layouts they are written to.
package model

import "encoding/json"

// Default message types, matched against the lower-cased AMQP "type"
// header. They can be overridden with MSG_TYPE_AUDIT and MSG_TYPE_MAIL_FLOW
// (see config.Config).
const (
	// TypeAudit marks an audit log message, stored in the audit table.
	TypeAudit = "audit"
	// TypeMailFlow marks a mail-flow log message, stored in the mail-flow
	// table. Publishers send it as "fuglu", after the mail filter that
	// produces these events. It is also the default when the header is
	// missing.
	TypeMailFlow = "fuglu"
)

// AuditLogsColumns is the column order of the audit logs table. Rows from
// logs.ParseAudit are in this order.
var AuditLogsColumns = []string{
	"organization_id", "user_id", "message", "action_type", "action_timestamp", "details",
}

// MailFlowLogsColumns is the column order of the mail-flow logs table. Rows
// from logs.ParseMailFlow are in this order.
var MailFlowLogsColumns = []string{
	"euid", "from_email_id", "to_email_ids", "email_domains", "subject", "type",
	"status", "status_description", "log_timestamp", "email_timestamp", "meta_data",
}

// AuditLogIn is the JSON body of an audit message. Pointer and RawMessage
// fields distinguish a missing or null field from an empty value, so
// validation can reject them.
type AuditLogIn struct {
	OrganizationID  *string         `json:"organization_id"`
	UserID          *string         `json:"user_id"`
	Message         *string         `json:"message"`
	ActionType      *string         `json:"action_type"`
	ActionTimestamp *string         `json:"action_timestamp"`
	Details         json.RawMessage `json:"details"`
}

// MailFlowLogIn is the JSON body of a mail-flow message. Unknown fields are
// ignored.
type MailFlowLogIn struct {
	EUID              *string         `json:"euid"`
	FromEmailID       *string         `json:"from_email_id"`
	ToEmailIDs        []*string       `json:"to_email_ids"`
	EmailDomains      []string        `json:"email_domains"`
	Subject           *string         `json:"subject"`
	Type              *string         `json:"type"`
	Status            *string         `json:"status"`
	StatusDescription *string         `json:"status_description"`
	LogTimestamp      *string         `json:"log_timestamp"`
	EmailTimestamp    *string         `json:"email_timestamp"`
	MetaData          json.RawMessage `json:"meta_data"`
}

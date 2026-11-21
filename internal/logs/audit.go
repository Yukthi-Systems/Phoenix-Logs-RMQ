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

package logs

import (
	"encoding/json"
	"time"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logger"
	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/model"
)

// ParseAudit validates an audit log payload and returns a row in
// model.AuditLogsColumns order.
//
// Every field is required and must not be null; details must be a JSON
// object and is returned as JSON text. Unknown fields are ignored. If
// action_timestamp cannot be parsed, it is set to the current time in IST
// and the row is still returned.
//
// The returned error wraps ErrInvalid when the message should be dropped.
func ParseAudit(body []byte) ([]any, error) {
	var in model.AuditLogIn
	if err := json.Unmarshal(body, &in); err != nil {
		// This covers bad JSON, a body that is not an object, and wrong
		// field types (for example a number where a string is expected).
		return nil, invalid("audit decode: %v", err)
	}

	for _, f := range []struct {
		name string
		v    *string
	}{
		{"organization_id", in.OrganizationID},
		{"user_id", in.UserID},
		{"message", in.Message},
		{"action_type", in.ActionType},
		{"action_timestamp", in.ActionTimestamp},
	} {
		if err := requireStr(f.name, f.v); err != nil {
			return nil, err
		}
	}
	details, err := requireObject("details", in.Details)
	if err != nil {
		return nil, err
	}

	ts, err := parseISO(*in.ActionTimestamp)
	if err != nil {
		logger.Error().Err(err).Str("value", *in.ActionTimestamp).
			Msg("error parsing action_timestamp, setting to current time")
		ts = time.Now().In(IST)
	}

	return []any{
		*in.OrganizationID,
		*in.UserID,
		*in.Message,
		*in.ActionType,
		ts,
		details,
	}, nil
}

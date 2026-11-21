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

// ParseMailFlow validates a mail-flow log payload and returns a row in
// model.MailFlowLogsColumns order.
//
// Every field is required and must not be null. to_email_ids and
// email_domains must be JSON arrays of strings (a null inside to_email_ids
// is rejected), and meta_data must be a JSON object, returned as JSON text.
// Unknown fields are ignored. If either log_timestamp or email_timestamp
// cannot be parsed, both are set to the same current time in IST and the
// row is still returned.
//
// The returned error wraps ErrInvalid when the message should be dropped.
func ParseMailFlow(body []byte) ([]any, error) {
	var in model.MailFlowLogIn
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, invalid("mail_flow decode: %v", err)
	}

	for _, f := range []struct {
		name string
		v    *string
	}{
		{"euid", in.EUID},
		{"from_email_id", in.FromEmailID},
		{"subject", in.Subject},
		{"type", in.Type},
		{"status", in.Status},
		{"status_description", in.StatusDescription},
		{"log_timestamp", in.LogTimestamp},
		{"email_timestamp", in.EmailTimestamp},
	} {
		if err := requireStr(f.name, f.v); err != nil {
			return nil, err
		}
	}
	if in.ToEmailIDs == nil {
		return nil, invalid("missing or non-list field %q", "to_email_ids")
	}
	if in.EmailDomains == nil {
		return nil, invalid("missing or non-list field %q", "email_domains")
	}
	toIDs := make([]string, len(in.ToEmailIDs))
	for i, e := range in.ToEmailIDs {
		if e == nil {
			return nil, invalid("invalid email address in to_email_ids")
		}
		toIDs[i] = *e
	}
	meta, err := requireObject("meta_data", in.MetaData)
	if err != nil {
		return nil, err
	}

	// If either timestamp fails to parse, both are set to the same current
	// time so the pair stays consistent.
	logTS, err1 := parseISO(*in.LogTimestamp)
	emailTS, err2 := parseISO(*in.EmailTimestamp)
	if err1 != nil || err2 != nil {
		logger.Error().AnErr("log_timestamp", err1).AnErr("email_timestamp", err2).
			Msg("error parsing date, setting to current time")
		now := time.Now().In(IST)
		logTS, emailTS = now, now
	}

	return []any{
		*in.EUID,
		*in.FromEmailID,
		toIDs,
		in.EmailDomains,
		*in.Subject,
		*in.Type,
		*in.Status,
		*in.StatusDescription,
		logTS,
		emailTS,
		meta,
	}, nil
}

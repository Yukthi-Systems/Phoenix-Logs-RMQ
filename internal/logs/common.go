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
	"bytes"
	"errors"
	"fmt"
	"time"
)

// IST (Asia/Kolkata, UTC+05:30) is the zone used for timestamps that carry
// no offset and for the "now" fallback. India has no daylight saving time,
// so a fixed zone is exact and needs no tzdata.
var IST = time.FixedZone("IST", 5*3600+30*60)

// ErrInvalid marks a payload that will never be valid. The RabbitMQ handler
// drops (acks) such messages instead of requeueing them.
var ErrInvalid = errors.New("invalid log message")

// invalid returns an error that wraps ErrInvalid with a formatted reason.
func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

// requireStr returns an ErrInvalid error if the string field name is
// missing, null, or was not a JSON string (in which case decoding left v
// nil).
func requireStr(name string, v *string) error {
	if v == nil {
		return invalid("missing or non-string field %q", name)
	}
	return nil
}

// requireObject checks that the field name is present and is a JSON object,
// and returns its raw JSON text, which is stored unchanged.
func requireObject(name string, raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return "", invalid("missing or non-object field %q", name)
	}
	return string(raw), nil
}

// isoLayouts are the ISO 8601 shapes accepted by parseISO. When parsing, Go
// accepts fractional seconds after the seconds field even though the
// layouts do not show them.
var isoLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05Z0700",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05Z0700",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15:04",
	"2006-01-02",
}

// parseISO parses an ISO 8601 timestamp in any of the isoLayouts. It accepts
// "T" or a space as the separator, optional fractional seconds, an optional
// zone ("Z", "+05:30", "+0530"), and date-only values. A timestamp with no
// zone is read as IST. On failure it returns the error from the last layout
// tried.
func parseISO(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range isoLayouts {
		t, err := time.ParseInLocation(layout, s, IST)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

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
	"errors"
	"testing"
	"time"
)

const validMail = `{"euid":"e1","from_email_id":"a@x.com","to_email_ids":["b@y.com"],
"email_domains":["y.com"],"subject":"s","type":"outbound","status":"sent",
"status_description":"ok","log_timestamp":"2025-07-01T10:00:00+05:30",
"email_timestamp":"2025-07-01 10:00:00.123456","meta_data":{"k":1},"extra":"ignored"}`

func TestMailFlowValid(t *testing.T) {
	row, err := ParseMailFlow([]byte(validMail))
	if err != nil {
		t.Fatal(err)
	}
	if len(row) != 11 || row[0] != "e1" || row[10] != `{"k":1}` {
		t.Fatalf("bad row %#v", row)
	}
	naive := row[9].(time.Time)
	if _, off := naive.Zone(); off != 19800 || naive.Nanosecond() != 123456000 {
		t.Fatalf("naive ts should be IST with micros, got %v", naive)
	}
}

func TestMailFlowInvalid(t *testing.T) {
	cases := map[string]string{
		"missing field":   `{"euid":"e1"}`,
		"wrong type":      `{"euid":1}`,
		"null to item":    `{"euid":"e1","from_email_id":"a","to_email_ids":[null],"email_domains":[],"subject":"s","type":"t","status":"s","status_description":"d","log_timestamp":"x","email_timestamp":"x","meta_data":{}}`,
		"meta not object": `{"euid":"e1","from_email_id":"a","to_email_ids":[],"email_domains":[],"subject":"s","type":"t","status":"s","status_description":"d","log_timestamp":"x","email_timestamp":"x","meta_data":[]}`,
		"not json":        `nope`,
		"array body":      `[1,2]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseMailFlow([]byte(body)); !errors.Is(err, ErrInvalid) {
				t.Errorf("want ErrInvalid, got %v", err)
			}
		})
	}
}

func TestMailFlowBadTimestampResetsBoth(t *testing.T) {
	body := `{"euid":"e1","from_email_id":"a","to_email_ids":[],"email_domains":[],"subject":"s","type":"t","status":"s","status_description":"d","log_timestamp":"2025-07-01T10:00:00Z","email_timestamp":"garbage","meta_data":{}}`
	row, err := ParseMailFlow([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	logTS, emailTS := row[8].(time.Time), row[9].(time.Time)
	if !logTS.Equal(emailTS) || time.Since(logTS) > time.Minute {
		t.Fatalf("both timestamps should be the same 'now', got %v and %v", logTS, emailTS)
	}
}

func TestBadTimestampFallsBackToNow(t *testing.T) {
	body := `{"organization_id":"o","user_id":"u","message":"m","action_type":"a","action_timestamp":"garbage","details":{}}`
	row, err := ParseAudit([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(row[4].(time.Time)) > time.Minute {
		t.Fatal("expected now fallback")
	}
}

func TestAuditNullDetails(t *testing.T) {
	body := `{"organization_id":"o","user_id":"u","message":"m","action_type":"a","action_timestamp":"2025-01-01","details":null}`
	if _, err := ParseAudit([]byte(body)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestParseISO(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Time
		wantErr bool
	}{
		{in: "2025-07-01T10:00:00Z", want: time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC)},
		{in: "2025-07-01T10:00:00+05:30", want: time.Date(2025, 7, 1, 10, 0, 0, 0, IST)},
		{in: "2025-07-01T10:00:00+0530", want: time.Date(2025, 7, 1, 10, 0, 0, 0, IST)},
		{in: "2025-07-01 10:00:00.5", want: time.Date(2025, 7, 1, 10, 0, 0, 5e8, IST)},
		{in: "2025-07-01T10:00", want: time.Date(2025, 7, 1, 10, 0, 0, 0, IST)},
		{in: "2025-07-01", want: time.Date(2025, 7, 1, 0, 0, 0, 0, IST)},
		{in: "07/01/2025", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseISO(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseISO(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

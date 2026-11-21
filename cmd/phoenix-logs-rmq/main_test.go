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

package main

import "testing"

func TestAMQPURL(t *testing.T) {
	tests := []struct {
		name  string
		user  string
		pass  string
		vhost string
		want  string
	}{
		{"default vhost", "guest", "guest", "/", "amqp://guest:guest@localhost:5672/%2F"},
		{"named vhost", "u", "p", "prod", "amqp://u:p@localhost:5672/prod"},
		{"special chars in password", "u", "p@ss:w/rd", "/", "amqp://u:p%40ss%3Aw%2Frd@localhost:5672/%2F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := amqpURL(tt.user, tt.pass, "localhost", 5672, tt.vhost); got != tt.want {
				t.Errorf("amqpURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

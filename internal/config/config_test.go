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

package config

import (
	"slices"
	"testing"
	"time"
)

func TestFromEnvDefaults(t *testing.T) {
	// Clear anything inherited from the developer's shell.
	for _, k := range []string{"BATCH_SIZE", "RB_WORKER_COUNT", "RB_PREFETCH", "RB_PORT", "RB_VHOST", "BATCH_FLUSH_INTERVAL", "CLICK_HOUSE_HTTP"} {
		t.Setenv(k, "")
	}
	c := FromEnv()
	if c.BatchSize != 500 || c.WorkerCount != 1000 || c.RabbitPrefetch != 1000 {
		t.Errorf("batch defaults: size=%d workers=%d prefetch=%d", c.BatchSize, c.WorkerCount, c.RabbitPrefetch)
	}
	if c.RabbitPort != 5672 || c.RabbitVHost != "/" {
		t.Errorf("rabbit defaults: port=%d vhost=%q", c.RabbitPort, c.RabbitVHost)
	}
	if c.FlushInterval != 2*time.Second || !c.ClickHouseHTTP {
		t.Errorf("flush=%v http=%v", c.FlushInterval, c.ClickHouseHTTP)
	}
}

func TestFromEnvOverridesAndBadValues(t *testing.T) {
	t.Setenv("BATCH_SIZE", "10")
	t.Setenv("RB_WORKER_COUNT", "not-a-number")
	t.Setenv("BATCH_FLUSH_INTERVAL", "250ms")
	t.Setenv("CLICK_HOUSE_HTTP", "false")
	c := FromEnv()
	if c.BatchSize != 10 {
		t.Errorf("BatchSize = %d, want 10", c.BatchSize)
	}
	if c.WorkerCount != 20 {
		t.Errorf("invalid RB_WORKER_COUNT should fall back to 2*BATCH_SIZE, got %d", c.WorkerCount)
	}
	if c.FlushInterval != 250*time.Millisecond || c.ClickHouseHTTP {
		t.Errorf("flush=%v http=%v", c.FlushInterval, c.ClickHouseHTTP)
	}
}

func TestMissingRequired(t *testing.T) {
	c := &Config{RabbitHost: "h", RabbitUsername: "u"}
	want := []string{"RB_PASSWD", "CLICK_HOUSE_HOST", "CLICK_HOUSE_PASSWORD"}
	if got := c.MissingRequired(); !slices.Equal(got, want) {
		t.Errorf("MissingRequired() = %v, want %v", got, want)
	}
	full := &Config{RabbitHost: "h", RabbitUsername: "u", RabbitPassword: "p", ClickHouseHost: "c", ClickHousePassword: "p"}
	if got := full.MissingRequired(); len(got) != 0 {
		t.Errorf("MissingRequired() = %v, want none", got)
	}
}

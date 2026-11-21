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

package sink

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// fakeConn records every batch sent through PrepareBatch. Embedding the
// interface satisfies the methods the Batcher never calls.
type fakeConn struct {
	driver.Conn

	mu      sync.Mutex
	queries []string
	batches [][][]any // rows of each successfully sent batch
	sendErr error
}

func (c *fakeConn) PrepareBatch(_ context.Context, query string, _ ...driver.PrepareBatchOption) (driver.Batch, error) {
	c.mu.Lock()
	c.queries = append(c.queries, query)
	c.mu.Unlock()
	return &fakeBatch{conn: c}, nil
}

func (c *fakeConn) sent() [][][]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][][]any(nil), c.batches...)
}

type fakeBatch struct {
	driver.Batch
	conn *fakeConn
	rows [][]any
}

func (b *fakeBatch) Append(v ...any) error { b.rows = append(b.rows, v); return nil }
func (b *fakeBatch) Abort() error          { return nil }
func (b *fakeBatch) Send() error {
	b.conn.mu.Lock()
	defer b.conn.mu.Unlock()
	if b.conn.sendErr != nil {
		return b.conn.sendErr
	}
	b.conn.batches = append(b.conn.batches, b.rows)
	return nil
}

// addAll calls Add for n rows concurrently and returns their results.
func addAll(t *testing.T, b *Batcher, n int) []error {
	t.Helper()
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = b.Add(context.Background(), []any{i})
		}()
	}
	wg.Wait()
	return errs
}

func TestBatcherQuery(t *testing.T) {
	conn := &fakeConn{}
	b := New(conn, "t", []string{"a", "b"}, 1, time.Hour, time.Second)
	defer b.Close()
	if err := b.Add(context.Background(), []any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if want := "INSERT INTO t (a, b)"; conn.queries[0] != want {
		t.Errorf("query = %q, want %q", conn.queries[0], want)
	}
}

func TestBatcherFlushesWhenFull(t *testing.T) {
	conn := &fakeConn{}
	// A long interval means only the size limit can trigger a write.
	b := New(conn, "t", []string{"v"}, 5, time.Hour, time.Second)
	defer b.Close()

	for _, err := range addAll(t, b, 10) {
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	got := conn.sent()
	if len(got) != 2 || len(got[0]) != 5 || len(got[1]) != 5 {
		t.Fatalf("want 2 batches of 5 rows, got %d batches", len(got))
	}
}

func TestBatcherFlushesOnInterval(t *testing.T) {
	conn := &fakeConn{}
	b := New(conn, "t", []string{"v"}, 100, 20*time.Millisecond, time.Second)
	defer b.Close()

	start := time.Now()
	if err := b.Add(context.Background(), []any{1}); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("partial batch was not flushed by the timer")
	}
	if got := conn.sent(); len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("want 1 batch of 1 row, got %v", got)
	}
}

func TestBatcherReportsInsertError(t *testing.T) {
	sendErr := errors.New("boom")
	conn := &fakeConn{sendErr: sendErr}
	b := New(conn, "t", []string{"v"}, 3, time.Hour, time.Second)
	defer b.Close()

	for _, err := range addAll(t, b, 3) {
		if !errors.Is(err, sendErr) {
			t.Errorf("Add error = %v, want %v", err, sendErr)
		}
	}
}

func TestBatcherContextCancel(t *testing.T) {
	conn := &fakeConn{}
	b := New(conn, "t", []string{"v"}, 100, time.Hour, time.Second)
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := b.Add(ctx, []any{1}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Add error = %v, want DeadlineExceeded", err)
	}
}

func TestBatcherCloseFlushesAndRejects(t *testing.T) {
	conn := &fakeConn{}
	b := New(conn, "t", []string{"v"}, 100, time.Hour, time.Second)

	done := make(chan error, 1)
	go func() { done <- b.Add(context.Background(), []any{1}) }()

	// Give Add time to hand the row to the loop. The interval is an hour,
	// so only Close can flush it.
	time.Sleep(50 * time.Millisecond)
	b.Close()

	if err := <-done; err != nil {
		t.Fatalf("pending Add after Close = %v, want nil", err)
	}
	if got := conn.sent(); len(got) != 1 {
		t.Fatalf("Close should flush the pending row, got %d batches", len(got))
	}
	if err := b.Add(context.Background(), []any{2}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Add after Close = %v, want ErrClosed", err)
	}
	b.Close() // second Close must not panic
}

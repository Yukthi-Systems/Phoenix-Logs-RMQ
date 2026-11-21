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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logger"
)

// ErrClosed is returned by Add after Close has been called.
var ErrClosed = errors.New("batcher closed")

// pending is a row waiting to be inserted, together with the channel on
// which its caller receives the insert result.
type pending struct {
	row  []any
	done chan error // buffered(1); receives the insert result
}

// Batcher collects rows for one table and writes them in bulk. All methods
// are safe for concurrent use.
type Batcher struct {
	conn          driver.Conn
	table         string
	query         string
	maxRows       int
	flushInterval time.Duration
	insertTimeout time.Duration

	in       chan pending
	stop     chan struct{}
	wg       sync.WaitGroup
	closeMu  sync.RWMutex
	isClosed bool
}

// New creates a Batcher that inserts into table, with each row's values in
// columns order, and starts its background loop. A batch is written once
// it holds maxRows rows or flushInterval has passed since the last write,
// whichever comes first. Each insert is bounded by insertTimeout.
//
// The caller must call Close to write any remaining rows and stop the loop.
func New(conn driver.Conn, table string, columns []string, maxRows int, flushInterval, insertTimeout time.Duration) *Batcher {
	b := &Batcher{
		conn:          conn,
		table:         table,
		query:         fmt.Sprintf("INSERT INTO %s (%s)", table, strings.Join(columns, ", ")),
		maxRows:       maxRows,
		flushInterval: flushInterval,
		insertTimeout: insertTimeout,
		in:            make(chan pending, maxRows),
		stop:          make(chan struct{}),
	}
	b.wg.Add(1)
	go b.loop()
	return b
}

// Add queues row and blocks until the batch holding it has been inserted. It
// returns nil when the row is stored in ClickHouse. If ctx ends first it
// returns ctx.Err(). The row may still be inserted after that, so the
// message is redelivered and may be stored twice (at-least-once delivery).
func (b *Batcher) Add(ctx context.Context, row []any) error {
	b.closeMu.RLock()
	if b.isClosed {
		b.closeMu.RUnlock()
		return ErrClosed
	}
	p := pending{row: row, done: make(chan error, 1)}
	select {
	case b.in <- p:
		b.closeMu.RUnlock()
	case <-ctx.Done():
		b.closeMu.RUnlock()
		return ctx.Err()
	}

	select {
	case err := <-p.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops taking rows, writes whatever is still queued, and waits for
// that write to finish. Later calls to Add return ErrClosed. Calling Close
// more than once is safe.
func (b *Batcher) Close() {
	b.closeMu.Lock()
	if b.isClosed {
		b.closeMu.Unlock()
		return
	}
	b.isClosed = true
	b.closeMu.Unlock()
	close(b.stop)
	b.wg.Wait()
}

// loop is the Batcher's single writer goroutine. It collects rows from Add
// and flushes them when the batch is full, when the flush timer fires, or
// when Close is called.
func (b *Batcher) loop() {
	defer b.wg.Done()
	buf := make([]pending, 0, b.maxRows)
	timer := time.NewTimer(b.flushInterval)
	defer timer.Stop()

	flush := func() {
		if len(buf) > 0 {
			b.flush(buf)
			buf = buf[:0]
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(b.flushInterval)
	}

	for {
		select {
		case p := <-b.in:
			buf = append(buf, p)
			if len(buf) >= b.maxRows {
				flush()
			}
		case <-timer.C:
			flush()
		case <-b.stop:
			// Close holds closeMu, so no new sends can start. Drain what is
			// already buffered and write it.
			for {
				select {
				case p := <-b.in:
					buf = append(buf, p)
				default:
					if len(buf) > 0 {
						b.flush(buf)
					}
					return
				}
			}
		}
	}
}

// flush inserts items as one batch, logs the outcome, and sends the result
// to every waiting caller.
func (b *Batcher) flush(items []pending) {
	start := time.Now()
	err := b.insert(items)
	if err != nil {
		logger.Error().Err(err).Str("table", b.table).Int("rows", len(items)).
			Msg("clickhouse insert failed")
	} else {
		logger.Info().Str("table", b.table).Int("rows", len(items)).
			Dur("took", time.Since(start)).Msg("successfully inserted logs")
	}
	for _, p := range items {
		p.done <- err
	}
}

// insert writes items to ClickHouse in a single INSERT, bounded by the
// Batcher's insert timeout.
func (b *Batcher) insert(items []pending) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.insertTimeout)
	defer cancel()

	batch, err := b.conn.PrepareBatch(ctx, b.query)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}
	for _, p := range items {
		if err := batch.Append(p.row...); err != nil {
			// A row the driver cannot encode would fail every retry. Drop
			// the whole batch and report the error. The handler requeues
			// the messages.
			_ = batch.Abort()
			return fmt.Errorf("append row: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}
	return nil
}

// Options configures the ClickHouse connection opened by Connect.
type Options struct {
	// Host is the ClickHouse hostname or IP.
	Host string
	// Port is the ClickHouse port: usually 8123 for HTTP, 9000 for native.
	Port int
	// Username is the ClickHouse user.
	Username string
	// Password is the ClickHouse password.
	Password string
	// Database is the default database for queries.
	Database string
	// HTTP selects the HTTP interface instead of the native protocol.
	HTTP bool
}

// Connect opens a ClickHouse connection pool with LZ4 compression and
// pings it, so a bad address or credentials fail at startup. The caller
// must Close the returned connection.
func Connect(ctx context.Context, o Options) (driver.Conn, error) {
	proto := clickhouse.Native
	if o.HTTP {
		proto = clickhouse.HTTP
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr:     []string{fmt.Sprintf("%s:%d", o.Host, o.Port)},
		Protocol: proto,
		Auth: clickhouse.Auth{
			Database: o.Database,
			Username: o.Username,
			Password: o.Password,
		},
		Compression: &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return conn, nil
}

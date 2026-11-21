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

// Package sink batches rows into bulk ClickHouse inserts.
//
// Handler goroutines call Batcher.Add, which blocks until the batch holding
// the row has been written and returns the insert result. That lets the
// caller acknowledge a RabbitMQ message only once its row is stored.
//
// A batch is written when it reaches its maximum size or when the flush
// interval passes, whichever comes first, so a partial batch never waits
// indefinitely. Batcher.Close writes any remaining rows before returning.
package sink

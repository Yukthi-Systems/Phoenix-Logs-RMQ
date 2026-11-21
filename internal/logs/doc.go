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

// Package logs validates incoming log payloads and turns them into
// ClickHouse rows.
//
// Each Parse function decodes a JSON body, checks that every required field
// is present, not null, and of the right type, and returns a row whose
// values are in the column order defined in package model.
//
// Errors that wrap ErrInvalid mean the payload can never be valid. Callers
// should drop such messages rather than retry them. A timestamp that cannot
// be parsed is not an error: it is replaced with the current time in IST
// and the row is kept.
package logs

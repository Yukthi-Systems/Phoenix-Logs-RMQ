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

package logger

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options configures the global logger created by Init.
type Options struct {
	// File is the path of the log file to write to and rotate.
	File string
	// MaxSizeMB is the size in megabytes at which the log file is rotated.
	MaxSizeMB int
	// MaxBackups is the number of rotated log files to keep.
	MaxBackups int
	// MaxAgeDays is the number of days to keep rotated log files.
	MaxAgeDays int
	// Compress gzip-compresses rotated log files.
	Compress bool
	// Level is the minimum level to log, such as "debug", "info" or "warn".
	// An invalid or empty value falls back to info.
	Level string
	// Console also writes log output to stderr in addition to File.
	Console bool
}

var (
	// log is the active logger. It is nil until Init or the first call
	// to get.
	log *zerolog.Logger
	// mu guards log.
	mu sync.RWMutex
)

// Init creates the global logger from opts, replacing any previous one.
// Records are written as JSON to a rotating file and, when opts.Console is
// true, also to stderr. Init is safe to call concurrently with the level
// functions.
func Init(opts Options) {
	lj := &lumberjack.Logger{
		Filename:   opts.File,
		MaxSize:    opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAgeDays,
		Compress:   opts.Compress,
	}
	var out zerolog.LevelWriter
	if opts.Console {
		out = zerolog.MultiLevelWriter(os.Stderr, lj)
	} else {
		out = zerolog.MultiLevelWriter(lj)
	}

	level := zerolog.InfoLevel
	if l, err := zerolog.ParseLevel(strings.ToLower(opts.Level)); err == nil && opts.Level != "" {
		level = l
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "time"
	zerolog.TimestampFunc = func() time.Time { return time.Now().Local() }

	l := zerolog.New(out).Level(level).With().Timestamp().Logger()

	mu.Lock()
	log = &l
	mu.Unlock()

	l.Info().Str("min_level", level.String()).Msg("logger initialized")
}

// get returns the global logger. If Init has not been called yet it lazily
// installs a default JSON logger that writes to stderr, so early log calls
// are never lost.
func get() *zerolog.Logger {
	mu.RLock()
	l := log
	mu.RUnlock()
	if l != nil {
		return l
	}

	mu.Lock()
	defer mu.Unlock()
	if log == nil {
		tmp := zerolog.New(os.Stderr).With().Timestamp().Logger()
		log = &tmp
	}
	return log
}

// Trace starts a new event at trace level on the global logger.
func Trace() *zerolog.Event { return get().Trace() }

// Debug starts a new event at debug level on the global logger.
func Debug() *zerolog.Event { return get().Debug() }

// Info starts a new event at info level on the global logger.
func Info() *zerolog.Event { return get().Info() }

// Warn starts a new event at warn level on the global logger.
func Warn() *zerolog.Event { return get().Warn() }

// Error starts a new event at error level on the global logger.
func Error() *zerolog.Event { return get().Error() }

// Fatal starts a new event at fatal level on the global logger. When the
// event is sent with Msg, Msgf or Send, the process exits with status 1.
func Fatal() *zerolog.Event { return get().Fatal() }

// Panic starts a new event at panic level on the global logger. When the
// event is sent with Msg, Msgf or Send, it panics.
func Panic() *zerolog.Event { return get().Panic() }

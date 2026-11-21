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

package rmq

import (
	"fmt"

	"github.com/Yukthi-Systems/Phoenix-Logs-RMQ/internal/logger"
)

// rabbitLogger adapts our process-wide logger (internal/logger) to the
// github.com/wagslane/go-rabbitmq Logger interface, so the library's own
// connection/consumer lifecycle messages land in our rotating log file
// instead of going straight to stdout via the standard log package.
type rabbitLogger struct{}

// Fatalf logs a formatted fatal-level message and exits the process with
// status 1.
func (rabbitLogger) Fatalf(format string, v ...any) {
	logger.Fatal().Msg(fmt.Sprintf(format, v...))
}

// Errorf logs a formatted error-level message.
func (rabbitLogger) Errorf(format string, v ...any) {
	logger.Error().Msg(fmt.Sprintf(format, v...))
}

// Warnf logs a formatted warn-level message.
func (rabbitLogger) Warnf(format string, v ...any) {
	logger.Warn().Msg(fmt.Sprintf(format, v...))
}

// Infof logs a formatted info-level message.
func (rabbitLogger) Infof(format string, v ...any) {
	logger.Info().Msg(fmt.Sprintf(format, v...))
}

// Debugf logs a formatted debug-level message.
func (rabbitLogger) Debugf(format string, v ...any) {
	logger.Debug().Msg(fmt.Sprintf(format, v...))
}

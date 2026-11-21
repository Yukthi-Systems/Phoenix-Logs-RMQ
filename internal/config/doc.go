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

// Package config loads the worker's configuration from environment
// variables and an optional .env file into a single, process-wide Config.
//
// Call Load once during startup, before any other package reads Cfg.
// Unset, empty, or unparsable values fall back to the defaults documented
// on each Config field. Use Config.MissingRequired to find required values
// that were not provided.
package config

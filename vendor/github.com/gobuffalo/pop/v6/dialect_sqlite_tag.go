//go:build sqlite
// +build sqlite

package pop

import (
	_ "github.com/mattn/go-sqlite3" // Load SQLite3 CGo driver
)

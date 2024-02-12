package pop

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/fatih/color"
	"github.com/gobuffalo/pop/v6/logging"
)

// Debug mode, to toggle verbose log traces
var Debug = false

// Color mode, to toggle colored logs
var Color = true

// SetLogger overrides the default logger.
func SetLogger(logger func(level logging.Level, s string, args ...interface{})) {
	log = logger
}

// SetLogger overrides the default logger.
func SetTxLogger(logger func(level logging.Level, anon interface{}, s string, args ...interface{})) {
	txlog = logger
}

var defaultStdLogger = stdlog.New(os.Stderr, "[POP] ", stdlog.LstdFlags)

var log = func(lvl logging.Level, s string, args ...interface{}) {
	txlog(lvl, nil, s, args...)
}

var txlog = func(lvl logging.Level, anon interface{}, s string, args ...interface{}) {
	if !Debug && lvl <= logging.Debug {
		return
	}
	if lvl == logging.SQL {
		if len(args) > 0 {
			xargs := make([]string, len(args))
			for i, a := range args {
				switch a.(type) {
				case string:
					xargs[i] = fmt.Sprintf("%q", a)
				default:
					xargs[i] = fmt.Sprintf("%v", a)
				}
			}
			s = fmt.Sprintf("%s - %s | %s", lvl, s, xargs)
		} else {
			s = fmt.Sprintf("%s - %s", lvl, s)
		}

		connID := ""
		txID := 0
		extra := ""
		switch typed := anon.(type) {
		case *Connection:
			connID = typed.ID
			if typed.TX != nil {
				txID = typed.TX.ID
			}

			extra = printStats(&typed.Store)
		case *Tx:
			txID = typed.ID
		case store:
			if t, ok := typed.(*Tx); ok {
				txID = t.ID
			}

			extra = printStats(&typed)
		}

		s = fmt.Sprintf("%s (conn=%v, tx=%v%v)", s, connID, txID, extra)
	} else {
		s = fmt.Sprintf(s, args...)
		s = fmt.Sprintf("%s - %s", lvl, s)
	}

	if Color {
		s = color.YellowString(s)
	}

	defaultStdLogger.Println(s)
}

// printStats returns a string represent connection pool information from
// the given store.
func printStats(s *store) string {
	if db, ok := (*s).(*dB); ok {
		s := db.Stats()
		return fmt.Sprintf(", maxconn: %d, openconn: %d, in-use: %d, idle: %d", s.MaxOpenConnections, s.OpenConnections, s.InUse, s.Idle)
	}

	return ""
}

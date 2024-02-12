package pop

import (
	"fmt"

	"github.com/gobuffalo/pop/v6/logging"
)

// CreateDB creates a database, given a connection definition
func CreateDB(c *Connection) error {
	deets := c.Dialect.Details()
	if deets.Database == "" {
		return nil
	}

	log(logging.Info, fmt.Sprintf("create %s (%s)", deets.Database, c.URL()))

	if err := c.Dialect.CreateDB(); err != nil {
		return fmt.Errorf("couldn't create database %s: %w", deets.Database, err)
	}
	return nil
}

// DropDB drops an existing database, given a connection definition
func DropDB(c *Connection) error {
	deets := c.Dialect.Details()
	if deets.Database == "" {
		return nil
	}

	log(logging.Info, fmt.Sprintf("drop %s (%s)", deets.Database, c.URL()))

	if err := c.Dialect.DropDB(); err != nil {
		return fmt.Errorf("couldn't drop database %s: %w", deets.Database, err)
	}
	return nil
}

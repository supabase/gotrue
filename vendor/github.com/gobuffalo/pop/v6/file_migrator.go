package pop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/pop/v6/logging"
)

// FileMigrator is a migrator for SQL and Fizz
// files on disk at a specified path.
type FileMigrator struct {
	Migrator
	Path string
}

// NewFileMigrator for a path and a Connection
func NewFileMigrator(path string, c *Connection) (FileMigrator, error) {
	fm := FileMigrator{
		Migrator: NewMigrator(c),
		Path:     path,
	}
	fm.SchemaPath = path

	runner := func(mf Migration, tx *Connection) error {
		f, err := os.Open(mf.Path)
		if err != nil {
			return err
		}
		defer f.Close()
		content, err := MigrationContent(mf, tx, f, true)
		if err != nil {
			return fmt.Errorf("error processing %s: %w", mf.Path, err)
		}
		if content == "" {
			return nil
		}
		err = tx.RawQuery(content).Exec()
		if err != nil {
			return fmt.Errorf("error executing %s, sql: %s: %w", mf.Path, content, err)
		}
		return nil
	}

	err := fm.findMigrations(runner)
	if err != nil {
		return fm, err
	}

	return fm, nil
}

func (fm *FileMigrator) findMigrations(runner func(mf Migration, tx *Connection) error) error {
	dir := fm.Path
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		// directory doesn't exist
		return nil
	}
	return filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			match, err := ParseMigrationFilename(info.Name())
			if err != nil {
				if strings.HasPrefix(err.Error(), "unsupported dialect") {
					log(logging.Warn, "ignoring migration file with %s", err.Error())
					return nil
				}
				return err
			}
			if match == nil {
				log(logging.Warn, "ignoring file %s because it does not match the migration file pattern", info.Name())
				return nil
			}
			mf := Migration{
				Path:      p,
				Version:   match.Version,
				Name:      match.Name,
				DBType:    match.DBType,
				Direction: match.Direction,
				Type:      match.Type,
				Runner:    runner,
			}
			switch mf.Direction {
			case "up":
				fm.UpMigrations.Migrations = append(fm.UpMigrations.Migrations, mf)
			case "down":
				fm.DownMigrations.Migrations = append(fm.DownMigrations.Migrations, mf)
			default:
				// the regex only matches `(up|down)` for direction, so a panic here is appropriate
				panic("got unknown migration direction " + mf.Direction)
			}
		}
		return nil
	})
}

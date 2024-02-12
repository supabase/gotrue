package pop

import (
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/gobuffalo/pop/v6/logging"
)

// MigrationBox is a wrapper around fs.FS and Migrator.
// This will allow you to run migrations from a fs.FS
// inside of a compiled binary.
type MigrationBox struct {
	Migrator
	FS fs.FS
}

// NewMigrationBox from a fs.FS and a Connection.
func NewMigrationBox(fsys fs.FS, c *Connection) (MigrationBox, error) {
	fm := MigrationBox{
		Migrator: NewMigrator(c),
		FS:       fsys,
	}

	runner := func(r io.Reader) func(mf Migration, tx *Connection) error {
		return func(mf Migration, tx *Connection) error {
			content, err := MigrationContent(mf, tx, r, true)
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
	}

	err := fm.findMigrations(runner)
	if err != nil {
		return fm, err
	}

	return fm, nil
}

func (fm *MigrationBox) findMigrations(runner func(r io.Reader) func(mf Migration, tx *Connection) error) error {
	return fs.WalkDir(fm.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

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

		f, err := fm.FS.Open(path)
		if err != nil {
			return err
		}

		mf := Migration{
			Path:      path,
			Version:   match.Version,
			Name:      match.Name,
			DBType:    match.DBType,
			Direction: match.Direction,
			Type:      match.Type,
			Runner:    runner(f),
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
		return nil
	})
}

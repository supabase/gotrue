package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/pop/v6/logging"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/supabase/gotrue/internal/conf"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "Migrate database strucutures. This will create new tables and add missing columns and indexes. Exits with 123 when migrations could not be applied at once.",
	Run:  migrate,
}

func migrate(cmd *cobra.Command, args []string) {
	globalConfig := loadGlobalConfig(cmd.Context())

	if globalConfig.DB.Driver == "" && globalConfig.DB.URL != "" {
		u, err := url.Parse(globalConfig.DB.URL)
		if err != nil {
			logrus.Fatalf("%+v", errors.Wrap(err, "parsing db connection url"))
		}
		globalConfig.DB.Driver = u.Scheme
	}

	log := logrus.StandardLogger()

	pop.Debug = false
	if globalConfig.Logging.Level != "" {
		level, err := logrus.ParseLevel(globalConfig.Logging.Level)
		if err != nil {
			log.Fatalf("Failed to parse log level: %+v", err)
		}
		log.SetLevel(level)
		if level == logrus.DebugLevel {
			// Set to true to display query info
			pop.Debug = true
		}
		if level != logrus.DebugLevel {
			var noopLogger = func(lvl logging.Level, s string, args ...interface{}) {
			}
			// Hide pop migration logging
			pop.SetLogger(noopLogger)
		}
	}

	u, _ := url.Parse(globalConfig.DB.URL)
	processedUrl := globalConfig.DB.URL
	if len(u.Query()) != 0 {
		processedUrl = fmt.Sprintf("%s&application_name=gotrue_migrations", processedUrl)
	} else {
		processedUrl = fmt.Sprintf("%s?application_name=gotrue_migrations", processedUrl)
	}
	deets := &pop.ConnectionDetails{
		Dialect: globalConfig.DB.Driver,
		URL:     processedUrl,
	}
	deets.Options = map[string]string{
		"migration_table_name": "schema_migrations",
		"Namespace":            globalConfig.DB.Namespace,
	}

	db, err := pop.NewConnection(deets)
	if err != nil {
		log.WithError(err).Fatal("opening db connection failed")
	}
	defer db.Close()

	if err := db.Open(); err != nil {
		log.WithError(err).Fatal("checking database connection failed")
	}

	performMigration(db, globalConfig)
}

func performMigration(db *pop.Connection, globalConfig *conf.GlobalConfiguration) {
	log := logrus.StandardLogger()

	log.Debugf("Reading migrations from %s", globalConfig.DB.MigrationsPath)

	var migrator pop.FileMigrator

	// PostgreSQL DDL is mostly transactional. We do not wish to apply
	// migrations partially, even though the migrations are packaged in
	// single atomic steps. GoTrue releases can be applied in smaller or
	// larger steps, and we wish for the migrations between two releases to
	// either apply fully or none at all. If the migrations can't be fully
	// applied, then there's an issue with the jump from release A to A'.
	err := db.Transaction(func(tx *pop.Connection) error {
		mig, err := pop.NewFileMigrator(globalConfig.DB.MigrationsPath, tx)
		if err != nil {
			log.WithError(err).Fatalf("failed to create migrator")
		}

		migrator = mig

		log.Debug("before status")
		if log.Level == logrus.DebugLevel {
			err = migrator.Status(os.Stdout)
			if err != nil {
				log.WithError(err).Error("migration status issue")
				return err
			}
		}

		// turn off schema dump
		migrator.SchemaPath = ""

		err = migrator.Up()
		if err != nil {
			log.WithError(err).Error("running db migrations in a transaction failed")
			return err
		}

		log.Infof("GoTrue migrations ready for commit")

		return nil
	})

	if err != nil {
		log.WithError(err).Error("failed to commit migrations in a transaction, exiting with 123")

		os.Exit(123) // signal to caller that migrations were unsuccessful
	} else {
		log.Infof("GoTrue migrations applied successfully")

		if log.Level == logrus.DebugLevel {
			err = migrator.Status(os.Stdout)
			if err != nil {
				log.WithError(err).Error("migration status failed")
			}
		}

		log.Debug("after status")
	}
}

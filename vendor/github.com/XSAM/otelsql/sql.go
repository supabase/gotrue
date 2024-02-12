// Copyright Sam Xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/metric/instrument"
)

var registerLock sync.Mutex

var maxDriverSlot = 1000

// Register initializes and registers OTel wrapped database driver
// identified by its driverName, using provided Option.
// It is possible to register multiple wrappers for the same database driver if
// needing different Option for different connections.
func Register(driverName string, options ...Option) (string, error) {
	// Retrieve the driver implementation we need to wrap with instrumentation
	db, err := sql.Open(driverName, "")
	if err != nil {
		return "", err
	}
	dri := db.Driver()
	if err = db.Close(); err != nil {
		return "", err
	}

	registerLock.Lock()
	defer registerLock.Unlock()

	// Since we might want to register multiple OTel drivers to have different
	// configurations, but potentially the same underlying database driver, we
	// cycle through to find available driver names.
	driverName = driverName + "-otelsql-"
	for i := 0; i < maxDriverSlot; i++ {
		var (
			found   = false
			regName = driverName + strconv.FormatInt(int64(i), 10)
		)
		for _, name := range sql.Drivers() {
			if name == regName {
				found = true
			}
		}
		if !found {
			sql.Register(regName, newDriver(dri, newConfig(options...)))
			return regName, nil
		}
	}
	return "", errors.New("unable to register driver, all slots have been taken")
}

// WrapDriver takes a SQL driver and wraps it with OTel instrumentation.
func WrapDriver(dri driver.Driver, options ...Option) driver.Driver {
	return newDriver(dri, newConfig(options...))
}

// Open is a wrapper over sql.Open with OTel instrumentation.
func Open(driverName, dataSourceName string, options ...Option) (*sql.DB, error) {
	// Retrieve the driver implementation we need to wrap with instrumentation
	db, err := sql.Open(driverName, "")
	if err != nil {
		return nil, err
	}
	d := db.Driver()
	if err = db.Close(); err != nil {
		return nil, err
	}

	otDriver := newOtDriver(d, newConfig(options...))

	if _, ok := d.(driver.DriverContext); ok {
		connector, err := otDriver.OpenConnector(dataSourceName)
		if err != nil {
			return nil, err
		}
		return sql.OpenDB(connector), nil
	}

	return sql.OpenDB(dsnConnector{dsn: dataSourceName, driver: otDriver}), nil
}

// OpenDB is a wrapper over sql.OpenDB with OTel instrumentation.
func OpenDB(c driver.Connector, options ...Option) *sql.DB {
	d := newOtDriver(c.Driver(), newConfig(options...))
	connector := newConnector(c, d)

	return sql.OpenDB(connector)
}

// RegisterDBStatsMetrics register sql.DBStats metrics with OTel instrumentation.
func RegisterDBStatsMetrics(db *sql.DB, opts ...Option) error {
	cfg := newConfig(opts...)
	meter := cfg.Meter

	instruments, err := newDBStatsInstruments(meter)
	if err != nil {
		return err
	}

	err = meter.RegisterCallback([]instrument.Asynchronous{
		instruments.connectionMaxOpen,
		instruments.connectionOpen,
		instruments.connectionWaitTotal,
		instruments.connectionWaitDurationTotal,
		instruments.connectionClosedMaxIdleTotal,
		instruments.connectionClosedMaxIdleTimeTotal,
		instruments.connectionClosedMaxLifetimeTotal,
	}, func(ctx context.Context) {
		dbStats := db.Stats()

		recordDBStatsMetrics(ctx, dbStats, instruments, cfg)
	})
	if err != nil {
		return err
	}
	return nil
}

func recordDBStatsMetrics(ctx context.Context, dbStats sql.DBStats, instruments *dbStatsInstruments, cfg config) {
	instruments.connectionMaxOpen.Observe(
		ctx,
		int64(dbStats.MaxOpenConnections),
		cfg.Attributes...,
	)

	instruments.connectionOpen.Observe(
		ctx,
		int64(dbStats.InUse),
		append(cfg.Attributes, connectionStatusKey.String("inuse"))...,
	)
	instruments.connectionOpen.Observe(
		ctx,
		int64(dbStats.Idle),
		append(cfg.Attributes, connectionStatusKey.String("idle"))...,
	)

	instruments.connectionWaitTotal.Observe(
		ctx,
		dbStats.WaitCount,
		cfg.Attributes...,
	)
	instruments.connectionWaitDurationTotal.Observe(
		ctx,
		float64(dbStats.WaitDuration.Nanoseconds())/1e6,
		cfg.Attributes...,
	)
	instruments.connectionClosedMaxIdleTotal.Observe(
		ctx,
		dbStats.MaxIdleClosed,
		cfg.Attributes...,
	)
	instruments.connectionClosedMaxIdleTimeTotal.Observe(
		ctx,
		dbStats.MaxIdleTimeClosed,
		cfg.Attributes...,
	)
	instruments.connectionClosedMaxLifetimeTotal.Observe(
		ctx,
		dbStats.MaxLifetimeClosed,
		cfg.Attributes...,
	)
}

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
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/unit"
)

const (
	namespace = "db.sql"
)

type dbStatsInstruments struct {
	connectionMaxOpen                asyncint64.Gauge
	connectionOpen                   asyncint64.Gauge
	connectionWaitTotal              asyncint64.Counter
	connectionWaitDurationTotal      asyncfloat64.Counter
	connectionClosedMaxIdleTotal     asyncint64.Counter
	connectionClosedMaxIdleTimeTotal asyncint64.Counter
	connectionClosedMaxLifetimeTotal asyncint64.Counter
}

type instruments struct {
	// The latency of calls in milliseconds
	latency syncfloat64.Histogram
}

func newInstruments(meter metric.Meter) (*instruments, error) {
	var instruments instruments
	var err error

	if instruments.latency, err = meter.SyncFloat64().Histogram(
		strings.Join([]string{namespace, "latency"}, "."),
		instrument.WithDescription("The latency of calls in milliseconds"),
		instrument.WithUnit(unit.Milliseconds),
	); err != nil {
		return nil, fmt.Errorf("failed to create latency instrument, %v", err)
	}
	return &instruments, nil
}

func newDBStatsInstruments(meter metric.Meter) (*dbStatsInstruments, error) {
	var instruments dbStatsInstruments
	var err error
	subsystem := "connection"

	if instruments.connectionMaxOpen, err = meter.AsyncInt64().Gauge(
		strings.Join([]string{namespace, subsystem, "max_open"}, "."),
		instrument.WithDescription("Maximum number of open connections to the database"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionMaxOpen instrument, %v", err)
	}

	if instruments.connectionOpen, err = meter.AsyncInt64().Gauge(
		strings.Join([]string{namespace, subsystem, "open"}, "."),
		instrument.WithDescription("The number of established connections both in use and idle"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionOpen instrument, %v", err)
	}

	if instruments.connectionWaitTotal, err = meter.AsyncInt64().Counter(
		strings.Join([]string{namespace, subsystem, "wait_total"}, "."),
		instrument.WithDescription("The total number of connections waited for"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionWaitTotal instrument, %v", err)
	}

	if instruments.connectionWaitDurationTotal, err = meter.AsyncFloat64().Counter(
		strings.Join([]string{namespace, subsystem, "wait_duration_total"}, "."),
		instrument.WithDescription("The total time blocked waiting for a new connection"),
		instrument.WithUnit(unit.Milliseconds),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionWaitDurationTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxIdleTotal, err = meter.AsyncInt64().Counter(
		strings.Join([]string{namespace, subsystem, "closed_max_idle_total"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetMaxIdleConns"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxIdleTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxIdleTimeTotal, err = meter.AsyncInt64().Counter(
		strings.Join([]string{namespace, subsystem, "closed_max_idle_time_total"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxIdleTimeTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxLifetimeTotal, err = meter.AsyncInt64().Counter(
		strings.Join([]string{namespace, subsystem, "closed_max_lifetime_total"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetConnMaxLifetime"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxLifetimeTotal instrument, %v", err)
	}

	return &instruments, nil
}

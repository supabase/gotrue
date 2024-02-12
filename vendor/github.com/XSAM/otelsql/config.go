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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricglobal "go.opentelemetry.io/otel/metric/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/XSAM/otelsql"
)

var (
	connectionStatusKey = attribute.Key("status")
	queryStatusKey      = attribute.Key("status")
	queryMethodKey      = attribute.Key("method")
)

// SpanNameFormatter is an interface that used to format span names.
type SpanNameFormatter interface {
	Format(ctx context.Context, method Method, query string) string
}

type config struct {
	TracerProvider trace.TracerProvider
	Tracer         trace.Tracer

	MeterProvider metric.MeterProvider
	Meter         metric.Meter

	Instruments *instruments

	SpanOptions SpanOptions

	// Attributes will be set to each span.
	Attributes []attribute.KeyValue

	// SpanNameFormatter will be called to produce span's name.
	// Default use method as span name
	SpanNameFormatter SpanNameFormatter

	// SQLCommenterEnabled enables context propagation for database
	// by injecting a comment into SQL statements.
	//
	// Experimental
	//
	// Notice: This config is EXPERIMENTAL and may be changed or removed in a
	// later release.
	SQLCommenterEnabled bool
	SQLCommenter        *commenter
}

// SpanOptions holds configuration of tracing span to decide
// whether to enable some features.
// By default all options are set to false intentionally when creating a wrapped
// driver and provide the most sensible default with both performance and
// security in mind.
type SpanOptions struct {
	// Ping, if set to true, will enable the creation of spans on Ping requests.
	Ping bool

	// RowsNext, if set to true, will enable the creation of events in spans on RowsNext
	// calls. This can result in many events.
	RowsNext bool

	// DisableErrSkip, if set to true, will suppress driver.ErrSkip errors in spans.
	DisableErrSkip bool

	// DisableQuery if set to true, will suppress db.statement in spans.
	DisableQuery bool

	// RecordError, if set, will be invoked with the current error, and if the func returns true
	// the record will be recorded on the current span.
	//
	// If this is not set it will default to record all errors (possible not ErrSkip, see option
	// DisableErrSkip).
	RecordError func(err error) bool

	// OmitConnResetSession if set to true will suppress sql.conn.reset_session spans
	OmitConnResetSession bool

	// OmitConnPrepare if set to true will suppress sql.conn.prepare spans
	OmitConnPrepare bool

	// OmitConnQuery if set to true will suppress sql.conn.query spans
	OmitConnQuery bool

	// OmitRows if set to true will suppress sql.rows spans
	OmitRows bool

	// OmitConnectorConnect if set to true will suppress sql.connector.connect spans
	OmitConnectorConnect bool
}

type defaultSpanNameFormatter struct{}

func (f *defaultSpanNameFormatter) Format(ctx context.Context, method Method, query string) string {
	return string(method)
}

// newConfig returns a config with all Options set.
func newConfig(options ...Option) config {
	cfg := config{
		TracerProvider:    otel.GetTracerProvider(),
		MeterProvider:     metricglobal.MeterProvider(),
		SpanNameFormatter: &defaultSpanNameFormatter{},
	}
	for _, opt := range options {
		opt.Apply(&cfg)
	}

	cfg.Tracer = cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	)
	cfg.Meter = cfg.MeterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(Version()),
	)

	cfg.SQLCommenter = newCommenter(cfg.SQLCommenterEnabled)

	var err error
	if cfg.Instruments, err = newInstruments(cfg.Meter); err != nil {
		otel.Handle(err)
	}

	return cfg
}

func withDBStatement(cfg config, query string) []attribute.KeyValue {
	if cfg.SpanOptions.DisableQuery {
		return cfg.Attributes
	}
	return append(cfg.Attributes, semconv.DBStatementKey.String(query))
}

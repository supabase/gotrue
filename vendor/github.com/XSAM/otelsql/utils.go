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
	"database/sql/driver"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func recordSpanError(span trace.Span, opts SpanOptions, err error) {
	if span == nil {
		return
	}
	if opts.RecordError != nil && !opts.RecordError(err) {
		return
	}

	switch err {
	case nil:
		return
	case driver.ErrSkip:
		if !opts.DisableErrSkip {
			span.RecordError(err)
			span.SetStatus(codes.Error, "")
		}
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")
	}
}

func recordMetric(ctx context.Context, instruments *instruments, defaultAttributes []attribute.KeyValue, method Method) func(error) {
	startTime := time.Now()

	return func(err error) {
		duration := float64(time.Since(startTime).Nanoseconds()) / 1e6

		attributes := defaultAttributes
		if err != nil {
			attributes = append(attributes, queryStatusKey.String("error"))
		} else {
			attributes = append(attributes, queryStatusKey.String("ok"))
		}

		attributes = append(attributes, queryMethodKey.String(string(method)))

		instruments.latency.Record(
			ctx,
			duration,
			attributes...,
		)
	}
}

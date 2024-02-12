// Copyright The OpenTelemetry Authors
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

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"context"
	goruntime "runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
)

// Runtime reports the work-in-progress conventional runtime metrics specified by OpenTelemetry.
type runtime struct {
	config config
	meter  metric.Meter
}

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MinimumReadMemStatsInterval sets the mininum interval
	// between calls to runtime.ReadMemStats().  Negative values
	// are ignored.
	MinimumReadMemStatsInterval time.Duration

	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// Option supports configuring optional settings for runtime metrics.
type Option interface {
	apply(*config)
}

// DefaultMinimumReadMemStatsInterval is the default minimum interval
// between calls to runtime.ReadMemStats().  Use the
// WithMinimumReadMemStatsInterval() option to modify this setting in
// Start().
const DefaultMinimumReadMemStatsInterval time.Duration = 15 * time.Second

// WithMinimumReadMemStatsInterval sets a minimum interval between calls to
// runtime.ReadMemStats(), which is a relatively expensive call to make
// frequently.  This setting is ignored when `d` is negative.
func WithMinimumReadMemStatsInterval(d time.Duration) Option {
	return minimumReadMemStatsIntervalOption(d)
}

type minimumReadMemStatsIntervalOption time.Duration

func (o minimumReadMemStatsIntervalOption) apply(c *config) {
	if o >= 0 {
		c.MinimumReadMemStatsInterval = time.Duration(o)
	}
}

// WithMeterProvider sets the Metric implementation to use for
// reporting.  If this option is not used, the global metric.MeterProvider
// will be used.  `provider` must be non-nil.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return metricProviderOption{provider}
}

type metricProviderOption struct{ metric.MeterProvider }

func (o metricProviderOption) apply(c *config) {
	if o.MeterProvider != nil {
		c.MeterProvider = o.MeterProvider
	}
}

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider:               global.MeterProvider(),
		MinimumReadMemStatsInterval: DefaultMinimumReadMemStatsInterval,
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Start initializes reporting of runtime metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MinimumReadMemStatsInterval < 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	if c.MeterProvider == nil {
		c.MeterProvider = global.MeterProvider()
	}
	r := &runtime{
		meter: c.MeterProvider.Meter(
			"go.opentelemetry.io/contrib/instrumentation/runtime",
			metric.WithInstrumentationVersion(SemVersion()),
		),
		config: c,
	}
	return r.register()
}

func (r *runtime) register() error {
	startTime := time.Now()
	uptime, err := r.meter.AsyncInt64().UpDownCounter(
		"runtime.uptime",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("Milliseconds since application was initialized"),
	)
	if err != nil {
		return err
	}

	goroutines, err := r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.goroutines",
		instrument.WithDescription("Number of goroutines that currently exist"),
	)
	if err != nil {
		return err
	}

	cgoCalls, err := r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.cgo.calls",
		instrument.WithDescription("Number of cgo calls made by the current process"),
	)
	if err != nil {
		return err
	}

	err = r.meter.RegisterCallback(
		[]instrument.Asynchronous{
			uptime,
			goroutines,
			cgoCalls,
		},
		func(ctx context.Context) {
			uptime.Observe(ctx, time.Since(startTime).Milliseconds())
			goroutines.Observe(ctx, int64(goruntime.NumGoroutine()))
			cgoCalls.Observe(ctx, goruntime.NumCgoCall())
		},
	)
	if err != nil {
		return err
	}

	return r.registerMemStats()
}

func (r *runtime) registerMemStats() error {
	var (
		err error

		heapAlloc    asyncint64.UpDownCounter
		heapIdle     asyncint64.UpDownCounter
		heapInuse    asyncint64.UpDownCounter
		heapObjects  asyncint64.UpDownCounter
		heapReleased asyncint64.UpDownCounter
		heapSys      asyncint64.UpDownCounter
		liveObjects  asyncint64.UpDownCounter

		// TODO: is ptrLookups useful? I've not seen a value
		// other than zero.
		ptrLookups asyncint64.Counter

		gcCount      asyncint64.Counter
		pauseTotalNs asyncint64.Counter
		gcPauseNs    syncint64.Histogram

		lastNumGC    uint32
		lastMemStats time.Time
		memStats     goruntime.MemStats

		// lock prevents a race between batch observer and instrument registration.
		lock sync.Mutex
	)

	lock.Lock()
	defer lock.Unlock()

	if heapAlloc, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_alloc",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of allocated heap objects"),
	); err != nil {
		return err
	}

	if heapIdle, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_idle",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes in idle (unused) spans"),
	); err != nil {
		return err
	}

	if heapInuse, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_inuse",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes in in-use spans"),
	); err != nil {
		return err
	}

	if heapObjects, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_objects",
		instrument.WithDescription("Number of allocated heap objects"),
	); err != nil {
		return err
	}

	// FYI see https://github.com/golang/go/issues/32284 to help
	// understand the meaning of this value.
	if heapReleased, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_released",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of idle spans whose physical memory has been returned to the OS"),
	); err != nil {
		return err
	}

	if heapSys, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.heap_sys",
		instrument.WithUnit(unit.Bytes),
		instrument.WithDescription("Bytes of heap memory obtained from the OS"),
	); err != nil {
		return err
	}

	if ptrLookups, err = r.meter.AsyncInt64().Counter(
		"process.runtime.go.mem.lookups",
		instrument.WithDescription("Number of pointer lookups performed by the runtime"),
	); err != nil {
		return err
	}

	if liveObjects, err = r.meter.AsyncInt64().UpDownCounter(
		"process.runtime.go.mem.live_objects",
		instrument.WithDescription("Number of live objects is the number of cumulative Mallocs - Frees"),
	); err != nil {
		return err
	}

	if gcCount, err = r.meter.AsyncInt64().Counter(
		"process.runtime.go.gc.count",
		instrument.WithDescription("Number of completed garbage collection cycles"),
	); err != nil {
		return err
	}

	// Note that the following could be derived as a sum of
	// individual pauses, but we may lose individual pauses if the
	// observation interval is too slow.
	if pauseTotalNs, err = r.meter.AsyncInt64().Counter(
		"process.runtime.go.gc.pause_total_ns",
		// TODO: nanoseconds units
		instrument.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses since the program started"),
	); err != nil {
		return err
	}

	if gcPauseNs, err = r.meter.SyncInt64().Histogram(
		"process.runtime.go.gc.pause_ns",
		// TODO: nanoseconds units
		instrument.WithDescription("Amount of nanoseconds in GC stop-the-world pauses"),
	); err != nil {
		return err
	}

	err = r.meter.RegisterCallback(
		[]instrument.Asynchronous{
			heapAlloc,
			heapIdle,
			heapInuse,
			heapObjects,
			heapReleased,
			heapSys,
			liveObjects,

			ptrLookups,

			gcCount,
			pauseTotalNs,
		}, func(ctx context.Context) {
			lock.Lock()
			defer lock.Unlock()

			now := time.Now()
			if now.Sub(lastMemStats) >= r.config.MinimumReadMemStatsInterval {
				goruntime.ReadMemStats(&memStats)
				lastMemStats = now
			}

			heapAlloc.Observe(ctx, int64(memStats.HeapAlloc))
			heapIdle.Observe(ctx, int64(memStats.HeapIdle))
			heapInuse.Observe(ctx, int64(memStats.HeapInuse))
			heapObjects.Observe(ctx, int64(memStats.HeapObjects))
			heapReleased.Observe(ctx, int64(memStats.HeapReleased))
			heapSys.Observe(ctx, int64(memStats.HeapSys))
			liveObjects.Observe(ctx, int64(memStats.Mallocs-memStats.Frees))
			ptrLookups.Observe(ctx, int64(memStats.Lookups))
			gcCount.Observe(ctx, int64(memStats.NumGC))
			pauseTotalNs.Observe(ctx, int64(memStats.PauseTotalNs))

			computeGCPauses(ctx, gcPauseNs, memStats.PauseNs[:], lastNumGC, memStats.NumGC)

			lastNumGC = memStats.NumGC
		})
	if err != nil {
		return err
	}
	return nil
}

func computeGCPauses(
	ctx context.Context,
	recorder syncint64.Histogram,
	circular []uint64,
	lastNumGC, currentNumGC uint32,
) {
	delta := int(int64(currentNumGC) - int64(lastNumGC))

	if delta == 0 {
		return
	}

	if delta >= len(circular) {
		// There were > 256 collections, some may have been lost.
		recordGCPauses(ctx, recorder, circular)
		return
	}

	length := uint32(len(circular))

	i := lastNumGC % length
	j := currentNumGC % length

	if j < i { // wrap around the circular buffer
		recordGCPauses(ctx, recorder, circular[i:])
		recordGCPauses(ctx, recorder, circular[:j])
		return
	}

	recordGCPauses(ctx, recorder, circular[i:j])
}

func recordGCPauses(
	ctx context.Context,
	recorder syncint64.Histogram,
	pauses []uint64,
) {
	for _, pause := range pauses {
		recorder.Record(ctx, int64(pause))
	}
}

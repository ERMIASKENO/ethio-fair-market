package kit

import (
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metrics is a tiny Prometheus-compatible registry (text exposition format 0.0.4).
// It covers what the services need: labelled counters and histograms plus Go runtime gauges.
type Metrics struct {
	mu         sync.Mutex
	service    string
	start      time.Time
	counters   map[string]*counter
	histograms map[string]*histogram
	help       map[string]string
}

type counter struct {
	values map[string]float64 // label string -> value
}

type histogram struct {
	buckets []float64
	series  map[string]*histSeries
}

type histSeries struct {
	counts []uint64
	sum    float64
	count  uint64
}

// DefaultBuckets are latency buckets in seconds.
var DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

// NewMetrics creates a registry.
func NewMetrics(service string) *Metrics {
	return &Metrics{
		service:    service,
		start:      time.Now(),
		counters:   map[string]*counter{},
		histograms: map[string]*histogram{},
		help:       map[string]string{},
	}
}

// Labels renders label pairs in a stable order: Labels("method","GET","code","200").
func Labels(kv ...string) string {
	if len(kv) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		v := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(kv[i+1])
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, kv[i], v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// Inc adds 1 to a counter.
func (m *Metrics) Inc(name, help, labels string) { m.Add(name, help, labels, 1) }

// Add adds v to a counter.
func (m *Metrics) Add(name, help, labels string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.counters[name]
	if !ok {
		c = &counter{values: map[string]float64{}}
		m.counters[name] = c
		m.help[name] = help
	}
	c.values[labels] += v
}

// Observe records a histogram value.
func (m *Metrics) Observe(name, help, labels string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.histograms[name]
	if !ok {
		h = &histogram{buckets: DefaultBuckets, series: map[string]*histSeries{}}
		m.histograms[name] = h
		m.help[name] = help
	}
	s, ok := h.series[labels]
	if !ok {
		s = &histSeries{counts: make([]uint64, len(h.buckets))}
		h.series[labels] = s
	}
	for i, b := range h.buckets {
		if v <= b {
			s.counts[i]++
		}
	}
	s.sum += v
	s.count++
}

func joinLabels(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "," + b
	}
}

func fmtFloat(f float64) string {
	if math.IsInf(f, 1) {
		return "+Inf"
	}
	return fmt.Sprintf("%g", f)
}

// Handler serves /metrics.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var b strings.Builder

		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		fmt.Fprintf(&b, "# HELP process_uptime_seconds Seconds since the process started.\n# TYPE process_uptime_seconds gauge\nprocess_uptime_seconds %g\n", time.Since(m.start).Seconds())
		fmt.Fprintf(&b, "# HELP go_goroutines Number of goroutines.\n# TYPE go_goroutines gauge\ngo_goroutines %d\n", runtime.NumGoroutine())
		fmt.Fprintf(&b, "# HELP go_memstats_heap_alloc_bytes Heap bytes in use.\n# TYPE go_memstats_heap_alloc_bytes gauge\ngo_memstats_heap_alloc_bytes %d\n", ms.HeapAlloc)
		fmt.Fprintf(&b, "# HELP app_build_info Build information.\n# TYPE app_build_info gauge\napp_build_info{%s} 1\n", Labels("service", m.service, "version", Version))

		m.mu.Lock()
		names := make([]string, 0, len(m.counters))
		for n := range m.counters {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s counter\n", n, m.help[n], n)
			keys := make([]string, 0, len(m.counters[n].values))
			for k := range m.counters[n].values {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "%s{%s} %g\n", n, k, m.counters[n].values[k])
			}
		}
		hnames := make([]string, 0, len(m.histograms))
		for n := range m.histograms {
			hnames = append(hnames, n)
		}
		sort.Strings(hnames)
		for _, n := range hnames {
			h := m.histograms[n]
			fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s histogram\n", n, m.help[n], n)
			keys := make([]string, 0, len(h.series))
			for k := range h.series {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				s := h.series[k]
				for i, bound := range h.buckets {
					fmt.Fprintf(&b, "%s_bucket{%s} %d\n", n, joinLabels(k, Labels("le", fmtFloat(bound))), s.counts[i])
				}
				fmt.Fprintf(&b, "%s_bucket{%s} %d\n", n, joinLabels(k, `le="+Inf"`), s.count)
				fmt.Fprintf(&b, "%s_sum{%s} %g\n%s_count{%s} %d\n", n, k, s.sum, n, k, s.count)
			}
		}
		m.mu.Unlock()
		_, _ = w.Write([]byte(b.String()))
	})
}

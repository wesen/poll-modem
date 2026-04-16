package cmd

import (
	"sync"
	"time"

	"github.com/go-go-golems/poll-modem/internal/modem"
	"github.com/prometheus/client_golang/prometheus"
)

type collectorMetrics struct {
	pollsTotal         *prometheus.CounterVec
	pollDuration       prometheus.Histogram
	lastSuccessUnix    prometheus.Gauge
	lastFailureUnix    prometheus.Gauge
	lastErrorUnix      prometheus.Gauge
	currentStatus      prometheus.Gauge
	downstreamChannels prometheus.Gauge
	upstreamChannels   prometheus.Gauge
	errorChannels      prometheus.Gauge

	registerOnce sync.Once
}

var pollMetrics = newCollectorMetrics()

func newCollectorMetrics() *collectorMetrics {
	m := &collectorMetrics{
		pollsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "poll_modem",
				Name:      "polls_total",
				Help:      "Total modem polls grouped by result.",
			},
			[]string{"result"},
		),
		pollDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "poll_modem",
			Name:      "poll_duration_seconds",
			Help:      "Duration of modem polling and storage cycles.",
			Buckets:   prometheus.DefBuckets,
		}),
		lastSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_success_unixtime",
			Help:      "Unix timestamp of the last successful poll.",
		}),
		lastFailureUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_failure_unixtime",
			Help:      "Unix timestamp of the last failed poll.",
		}),
		lastErrorUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_error_unixtime",
			Help:      "Unix timestamp of the last error.",
		}),
		currentStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "up",
			Help:      "Whether the last poll succeeded (1) or failed (0).",
		}),
		downstreamChannels: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_channels",
			Help:      "Number of downstream channels reported by the modem.",
		}),
		upstreamChannels: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_channels",
			Help:      "Number of upstream channels reported by the modem.",
		}),
		errorChannels: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "error_channels",
			Help:      "Number of error-codeword channels reported by the modem.",
		}),
	}

	m.registerOnce.Do(func() {
		prometheus.MustRegister(prometheus.NewGoCollector())
		prometheus.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		prometheus.MustRegister(m.pollsTotal)
		prometheus.MustRegister(m.pollDuration)
		prometheus.MustRegister(m.lastSuccessUnix)
		prometheus.MustRegister(m.lastFailureUnix)
		prometheus.MustRegister(m.lastErrorUnix)
		prometheus.MustRegister(m.currentStatus)
		prometheus.MustRegister(m.downstreamChannels)
		prometheus.MustRegister(m.upstreamChannels)
		prometheus.MustRegister(m.errorChannels)
	})

	return m
}

func (m *collectorMetrics) observeSuccess(duration time.Duration, info *modem.ModemInfo) {
	if m == nil {
		return
	}
	m.pollsTotal.WithLabelValues("success").Inc()
	m.pollDuration.Observe(duration.Seconds())
	m.lastSuccessUnix.SetToCurrentTime()
	m.currentStatus.Set(1)
	m.lastErrorUnix.Set(0)
	m.downstreamChannels.Set(float64(len(info.Downstream)))
	m.upstreamChannels.Set(float64(len(info.Upstream)))
	m.errorChannels.Set(float64(len(info.ErrorCodewords)))
}

func (m *collectorMetrics) observeFailure(duration time.Duration) {
	if m == nil {
		return
	}
	m.pollsTotal.WithLabelValues("failure").Inc()
	m.pollDuration.Observe(duration.Seconds())
	m.lastFailureUnix.SetToCurrentTime()
	m.lastErrorUnix.SetToCurrentTime()
	m.currentStatus.Set(0)
}

package cmd

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/poll-modem/internal/modem"
	"github.com/prometheus/client_golang/prometheus"
)

var numericPattern = regexp.MustCompile(`[-+]?\d*\.?\d+(?:[eE][-+]?\d+)?`)

type collectorMetrics struct {
	pollsTotal   *prometheus.CounterVec
	pollDuration prometheus.Histogram

	lastSuccessUnix *prometheus.GaugeVec
	lastFailureUnix *prometheus.GaugeVec
	lastErrorUnix   *prometheus.GaugeVec
	currentStatus   *prometheus.GaugeVec

	downstreamChannels *prometheus.GaugeVec
	upstreamChannels   *prometheus.GaugeVec
	errorChannels      *prometheus.GaugeVec

	downstreamSNR               *prometheus.GaugeVec
	downstreamPower             *prometheus.GaugeVec
	downstreamFrequency         *prometheus.GaugeVec
	downstreamLocked            *prometheus.GaugeVec
	upstreamPower               *prometheus.GaugeVec
	upstreamFrequency           *prometheus.GaugeVec
	upstreamSymbolRate          *prometheus.GaugeVec
	upstreamLocked              *prometheus.GaugeVec
	errorUnerroredCodewords     *prometheus.GaugeVec
	errorCorrectableCodewords   *prometheus.GaugeVec
	errorUncorrectableCodewords *prometheus.GaugeVec

	registerOnce sync.Once
}

var metricsRegistry = prometheus.NewRegistry()
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
		lastSuccessUnix: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_success_unixtime",
			Help:      "Unix timestamp of the last successful poll.",
		}, []string{}),
		lastFailureUnix: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_failure_unixtime",
			Help:      "Unix timestamp of the last failed poll.",
		}, []string{}),
		lastErrorUnix: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "last_error_unixtime",
			Help:      "Unix timestamp of the last error.",
		}, []string{}),
		currentStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "up",
			Help:      "Whether the last poll succeeded (1) or failed (0).",
		}, []string{}),
		downstreamChannels: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_channels",
			Help:      "Number of downstream channels reported by the modem.",
		}, []string{}),
		upstreamChannels: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_channels",
			Help:      "Number of upstream channels reported by the modem.",
		}, []string{}),
		errorChannels: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "error_channels",
			Help:      "Number of error-codeword channels reported by the modem.",
		}, []string{}),
		downstreamSNR: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_snr_db",
			Help:      "Downstream SNR in dB per channel.",
		}, []string{"channel_id"}),
		downstreamPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_power_dbmv",
			Help:      "Downstream power level in dBmV per channel.",
		}, []string{"channel_id"}),
		downstreamFrequency: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_frequency_hz",
			Help:      "Downstream frequency in hertz per channel.",
		}, []string{"channel_id"}),
		downstreamLocked: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "downstream_locked",
			Help:      "Whether each downstream channel is locked (1) or not (0).",
		}, []string{"channel_id"}),
		upstreamPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_power_dbmv",
			Help:      "Upstream power level in dBmV per channel.",
		}, []string{"channel_id"}),
		upstreamFrequency: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_frequency_hz",
			Help:      "Upstream frequency in hertz per channel.",
		}, []string{"channel_id"}),
		upstreamSymbolRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_symbol_rate_sps",
			Help:      "Upstream symbol rate in symbols per second per channel.",
		}, []string{"channel_id"}),
		upstreamLocked: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "upstream_locked",
			Help:      "Whether each upstream channel is locked (1) or not (0).",
		}, []string{"channel_id"}),
		errorUnerroredCodewords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "error_unerrored_codewords",
			Help:      "Unerrored codeword count per downstream channel.",
		}, []string{"channel_id"}),
		errorCorrectableCodewords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "error_correctable_codewords",
			Help:      "Correctable codeword count per downstream channel.",
		}, []string{"channel_id"}),
		errorUncorrectableCodewords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "poll_modem",
			Name:      "error_uncorrectable_codewords",
			Help:      "Uncorrectable codeword count per downstream channel.",
		}, []string{"channel_id"}),
	}

	m.registerOnce.Do(func() {
		metricsRegistry.MustRegister(prometheus.NewGoCollector())
		metricsRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		metricsRegistry.MustRegister(m.pollsTotal)
		metricsRegistry.MustRegister(m.pollDuration)
		metricsRegistry.MustRegister(m.lastSuccessUnix)
		metricsRegistry.MustRegister(m.lastFailureUnix)
		metricsRegistry.MustRegister(m.lastErrorUnix)
		metricsRegistry.MustRegister(m.currentStatus)
		metricsRegistry.MustRegister(m.downstreamChannels)
		metricsRegistry.MustRegister(m.upstreamChannels)
		metricsRegistry.MustRegister(m.errorChannels)
		metricsRegistry.MustRegister(m.downstreamSNR)
		metricsRegistry.MustRegister(m.downstreamPower)
		metricsRegistry.MustRegister(m.downstreamFrequency)
		metricsRegistry.MustRegister(m.downstreamLocked)
		metricsRegistry.MustRegister(m.upstreamPower)
		metricsRegistry.MustRegister(m.upstreamFrequency)
		metricsRegistry.MustRegister(m.upstreamSymbolRate)
		metricsRegistry.MustRegister(m.upstreamLocked)
		metricsRegistry.MustRegister(m.errorUnerroredCodewords)
		metricsRegistry.MustRegister(m.errorCorrectableCodewords)
		metricsRegistry.MustRegister(m.errorUncorrectableCodewords)
	})

	return m
}

func (m *collectorMetrics) observeSuccess(duration time.Duration, info *modem.ModemInfo) {
	if m == nil {
		return
	}
	m.pollsTotal.WithLabelValues("success").Inc()
	m.pollDuration.Observe(duration.Seconds())
	m.lastSuccessUnix.WithLabelValues().SetToCurrentTime()
	m.currentStatus.WithLabelValues().Set(1)
	m.lastErrorUnix.WithLabelValues().Set(0)
	m.downstreamChannels.WithLabelValues().Set(float64(len(info.Downstream)))
	m.upstreamChannels.WithLabelValues().Set(float64(len(info.Upstream)))
	m.errorChannels.WithLabelValues().Set(float64(len(info.ErrorCodewords)))
	m.observeChannelMetrics(info)
}

func (m *collectorMetrics) observeFailure(duration time.Duration) {
	if m == nil {
		return
	}
	m.pollsTotal.WithLabelValues("failure").Inc()
	m.pollDuration.Observe(duration.Seconds())
	m.lastFailureUnix.WithLabelValues().SetToCurrentTime()
	m.lastErrorUnix.WithLabelValues().SetToCurrentTime()
	m.currentStatus.WithLabelValues().Set(0)
}

func (m *collectorMetrics) observeChannelMetrics(info *modem.ModemInfo) {
	m.resetChannelMetrics()

	for _, ch := range info.Downstream {
		channelID := channelLabel(ch.ChannelID)
		if value, ok := parseMeasurement(ch.SNR); ok {
			m.downstreamSNR.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.PowerLevel); ok {
			m.downstreamPower.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.Frequency); ok {
			m.downstreamFrequency.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseLockStatus(ch.LockStatus); ok {
			m.downstreamLocked.WithLabelValues(channelID).Set(value)
		}
	}

	for _, ch := range info.Upstream {
		channelID := channelLabel(ch.ChannelID)
		if value, ok := parseMeasurement(ch.PowerLevel); ok {
			m.upstreamPower.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.Frequency); ok {
			m.upstreamFrequency.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.SymbolRate); ok {
			m.upstreamSymbolRate.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseLockStatus(ch.LockStatus); ok {
			m.upstreamLocked.WithLabelValues(channelID).Set(value)
		}
	}

	for _, ch := range info.ErrorCodewords {
		channelID := channelLabel(ch.ChannelID)
		if value, ok := parseMeasurement(ch.UnerroredCodewords); ok {
			m.errorUnerroredCodewords.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.CorrectableCodewords); ok {
			m.errorCorrectableCodewords.WithLabelValues(channelID).Set(value)
		}
		if value, ok := parseMeasurement(ch.UncorrectableCodewords); ok {
			m.errorUncorrectableCodewords.WithLabelValues(channelID).Set(value)
		}
	}
}

func (m *collectorMetrics) resetChannelMetrics() {
	m.downstreamSNR.Reset()
	m.downstreamPower.Reset()
	m.downstreamFrequency.Reset()
	m.downstreamLocked.Reset()
	m.upstreamPower.Reset()
	m.upstreamFrequency.Reset()
	m.upstreamSymbolRate.Reset()
	m.upstreamLocked.Reset()
	m.errorUnerroredCodewords.Reset()
	m.errorCorrectableCodewords.Reset()
	m.errorUncorrectableCodewords.Reset()
}

func channelLabel(value string) string {
	label := strings.TrimSpace(value)
	if label == "" {
		return "unknown"
	}
	return label
}

func parseMeasurement(value string) (float64, bool) {
	s := strings.TrimSpace(value)
	if s == "" {
		return 0, false
	}

	s = strings.ReplaceAll(s, ",", "")
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, false
	}

	numeric := strings.TrimSuffix(fields[0], "%")
	parsed, err := strconv.ParseFloat(numeric, 64)
	if err == nil {
		if len(fields) > 1 {
			return parsed * unitScale(fields[1]), true
		}
		return parsed, true
	}

	match := numericPattern.FindString(s)
	if match == "" {
		return 0, false
	}
	parsed, err = strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	if len(fields) > 1 {
		return parsed * unitScale(fields[1]), true
	}
	return parsed, true
}

func parseLockStatus(value string) (float64, bool) {
	s := strings.ToLower(strings.TrimSpace(value))
	if s == "" {
		return 0, false
	}
	if strings.Contains(s, "not") && strings.Contains(s, "lock") {
		return 0, true
	}
	if strings.Contains(s, "unlock") {
		return 0, true
	}
	if strings.Contains(s, "locked") {
		return 1, true
	}
	if strings.Contains(s, "lock") {
		return 1, true
	}
	if strings.Contains(s, "online") || s == "up" || s == "ok" || s == "true" {
		return 1, true
	}
	return 0, false
}

func unitScale(unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "hz":
		return 1
	case "khz":
		return 1e3
	case "mhz":
		return 1e6
	case "ghz":
		return 1e9
	case "sps", "sym/s":
		return 1
	case "ksps", "ksym/s":
		return 1e3
	case "msps", "msym/s":
		return 1e6
	default:
		return 1
	}
}

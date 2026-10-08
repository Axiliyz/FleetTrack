// Package metrics объявляет метрики Prometheus приложения и функции их записи.
package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "fleettrack",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests processed",
	},
	[]string{"method", "path", "status"},
)

var httpRequestsDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "fleettrack",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "Time of processing of HTTP requests",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)

var telemetryStageDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "fleettrack",
		Subsystem: "telemetry",
		Name:      "stage_duration_seconds",
		Help:      "Duration of telemetry processing stages",
		Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 16),
	},
	[]string{"stage"},
)

// RecordTelemetryStage записывает длительность стадии обработки телеметрии.
func RecordTelemetryStage(stage string, seconds float64) {
	telemetryStageDuration.WithLabelValues(stage).Observe(seconds)
}

// RecordHTTPRequest увеличивает счётчик HTTP-запросов по методу, маршруту и статусу.
func RecordHTTPRequest(method, path string, status int) {
	strStatus := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(method, path, strStatus).Inc()
}

// RecordHTTPDuration записывает время обработки HTTP-запроса в секундах.
func RecordHTTPDuration(method, path string, seconds float64) {
	httpRequestsDuration.WithLabelValues(method, path).Observe(seconds)
}

var dbTxDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "fleettrack",
		Subsystem: "db",
		Name:      "tx_duration_seconds",
		Help:      "Duration of database transaction phases (begin, commit)",
		Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 16),
	},
	[]string{"phase"},
)

// RecordDBTxPhase записывает длительность фазы транзакции БД.
func RecordDBTxPhase(phase string, d time.Duration) {
	dbTxDuration.WithLabelValues(phase).Observe(d.Seconds())
}

var alertQueueDropped = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "fleettrack",
	Subsystem: "alerts",
	Name:      "queue_dropped_total",
	Help:      "Telemetry points not queued for alert evaluation because the request was cancelled or the queue was closed",
})

// RecordAlertQueueDropped увеличивает счётчик точек, не попавших в очередь алертов.
func RecordAlertQueueDropped() {
	alertQueueDropped.Inc()
}

// RegisterAlertQueueLength публикует текущую длину очереди алертов.
func RegisterAlertQueueLength(length func() float64) {
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "fleettrack",
		Subsystem: "alerts",
		Name:      "queue_length",
		Help:      "Number of telemetry points waiting for alert evaluation",
	}, length)
}

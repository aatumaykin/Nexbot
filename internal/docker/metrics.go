package docker

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	registry         prometheus.Registerer
	containersTotal  prometheus.Gauge
	containersActive prometheus.Gauge
	tasksTotal       *prometheus.CounterVec
	taskDuration     *prometheus.HistogramVec
	circuitState     prometheus.Gauge
	pendingRequests  prometheus.Gauge
}

func InitPrometheusMetrics(namespace string, reg prometheus.Registerer) *PrometheusMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	m := &PrometheusMetrics{
		registry: reg,
		containersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "subagent_containers_total",
				Help:      "Total number of containers",
			},
		),
		containersActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "subagent_containers_active",
				Help:      "Number of active (busy) containers",
			},
		),
		tasksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "subagent_tasks_total",
				Help:      "Total number of subagent tasks",
			},
			[]string{"status"},
		),
		taskDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "subagent_task_duration_seconds",
				Help:      "Duration of subagent tasks",
				Buckets:   []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300},
			},
			[]string{"status"},
		),
		circuitState: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "subagent_circuit_breaker_state",
				Help:      "Circuit breaker state: 0=closed, 1=open, 2=half-open",
			},
		),
		pendingRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "subagent_pending_requests",
				Help:      "Number of pending requests",
			},
		),
	}

	reg.MustRegister(
		m.containersTotal,
		m.containersActive,
		m.tasksTotal,
		m.taskDuration,
		m.circuitState,
		m.pendingRequests,
	)

	return m
}

func (m *PrometheusMetrics) RecordTask(status string, duration time.Duration) {
	m.tasksTotal.WithLabelValues(status).Inc()
	m.taskDuration.WithLabelValues(status).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) SetCircuitState(state CircuitState) {
	m.circuitState.Set(float64(state))
}

func (m *PrometheusMetrics) SetPendingCount(count int64) {
	m.pendingRequests.Set(float64(count))
}

func (m *PrometheusMetrics) SetContainerCounts(total, active int64) {
	m.containersTotal.Set(float64(total))
	m.containersActive.Set(float64(active))
}

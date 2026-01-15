// Package metrics provides Prometheus metrics for the Keeper K8s Injector.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "keeper"

// Webhook metrics
var (
	// MutationsTotal counts total pod mutations
	MutationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "injector",
			Name:      "mutations_total",
			Help:      "Total number of pod mutations",
		},
		[]string{"namespace", "result"},
	)

	// MutationDuration tracks mutation latency
	MutationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "injector",
			Name:      "mutation_duration_seconds",
			Help:      "Time spent processing pod mutations",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"namespace"},
	)

	// SecretsInjected tracks number of secrets injected per mutation
	SecretsInjected = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "injector",
			Name:      "secrets_injected",
			Help:      "Number of secrets injected per pod",
			Buckets:   []float64{0, 1, 2, 3, 5, 10, 20},
		},
		[]string{"namespace"},
	)
)

// Sidecar metrics
var (
	// SecretFetchesTotal counts total secret fetches
	SecretFetchesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sidecar",
			Name:      "secret_fetches_total",
			Help:      "Total number of secret fetches",
		},
		[]string{"secret", "result"},
	)

	// SecretFetchDuration tracks fetch latency
	SecretFetchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "sidecar",
			Name:      "secret_fetch_duration_seconds",
			Help:      "Time spent fetching secrets from KSM",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"secret"},
	)

	// SecretsActive tracks number of active secrets
	SecretsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "sidecar",
			Name:      "secrets_active",
			Help:      "Number of secrets currently being managed",
		},
	)

	// LastRefreshTimestamp tracks last successful refresh
	LastRefreshTimestamp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "sidecar",
			Name:      "last_refresh_timestamp",
			Help:      "Unix timestamp of last successful secret refresh",
		},
	)

	// RefreshCyclesTotal counts refresh cycles
	RefreshCyclesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sidecar",
			Name:      "refresh_cycles_total",
			Help:      "Total number of refresh cycles",
		},
		[]string{"result"},
	)
)

// RecordMutation records a mutation event
func RecordMutation(namespace string, success bool, duration float64, secretCount int) {
	result := "success"
	if !success {
		result = "error"
	}
	MutationsTotal.WithLabelValues(namespace, result).Inc()
	MutationDuration.WithLabelValues(namespace).Observe(duration)
	if success {
		SecretsInjected.WithLabelValues(namespace).Observe(float64(secretCount))
	}
}

// RecordSecretFetch records a secret fetch event
func RecordSecretFetch(secretName string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "error"
	}
	SecretFetchesTotal.WithLabelValues(secretName, result).Inc()
	SecretFetchDuration.WithLabelValues(secretName).Observe(duration)
}

// RecordRefreshCycle records a refresh cycle completion
func RecordRefreshCycle(success bool) {
	result := "success"
	if !success {
		result = "error"
	}
	RefreshCyclesTotal.WithLabelValues(result).Inc()
}

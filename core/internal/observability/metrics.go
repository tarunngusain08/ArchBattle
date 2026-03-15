package observability

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	WSConnectionsActive       prometheus.Gauge
	WSMessageLatencyMillis    prometheus.Histogram
	MatchStateTransitionError prometheus.Counter
	MatchmakingWaitSeconds    prometheus.Histogram
	AnswerSubmitLatencyMillis prometheus.Histogram
	DailyChallengeBufferDays  prometheus.Gauge
}

func NewMetrics(namespace string, registry prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		WSConnectionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ws_connections_active",
			Help:      "Current number of open websocket connections.",
		}),
		WSMessageLatencyMillis: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "ws_message_latency_p99_ms",
			Help:      "Observed websocket message processing latency in milliseconds.",
			Buckets:   prometheus.ExponentialBuckets(5, 1.8, 10),
		}),
		MatchStateTransitionError: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "match_state_transition_errors",
			Help:      "Total invalid or failed match state transitions.",
		}),
		MatchmakingWaitSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "matchmaking_wait_p95_s",
			Help:      "Observed matchmaking wait time in seconds.",
			Buckets:   prometheus.LinearBuckets(5, 5, 18),
		}),
		AnswerSubmitLatencyMillis: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "answer_submit_latency_p99_ms",
			Help:      "Observed answer submission latency in milliseconds.",
			Buckets:   prometheus.ExponentialBuckets(2, 2, 10),
		}),
		DailyChallengeBufferDays: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "daily_challenge_buffer_days",
			Help:      "Available number of future daily challenges staged in the database.",
		}),
	}

	registry.MustRegister(
		metrics.WSConnectionsActive,
		metrics.WSMessageLatencyMillis,
		metrics.MatchStateTransitionError,
		metrics.MatchmakingWaitSeconds,
		metrics.AnswerSubmitLatencyMillis,
		metrics.DailyChallengeBufferDays,
	)

	return metrics
}

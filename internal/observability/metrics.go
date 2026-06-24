package observability

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

var (
	QueueDepthGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tasktide_queue_depth",
			Help: "Current number of pending jobs in the queue",
		},
		[]string{"queue_name"},
	)

	JobDurationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tasktide_job_processing_duration_seconds",
			Help:    "Duration of job processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue_name", "status"},
	)

	JobExecutionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasktide_jobs_executed_total",
			Help: "Total number of executed jobs",
		},
		[]string{"queue_name", "status"},
	)

	ActiveWorkersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tasktide_active_workers_count",
			Help: "Current number of active workers",
		},
	)

	RetryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasktide_job_retries_total",
			Help: "Total number of job retries",
		},
		[]string{"queue_name"},
	)
)

// StartMetricsCollector periodically updates the database-derived gauges (like queue depth and active worker count)
func StartMetricsCollector(ctx context.Context, queries *db.Queries) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update active workers
				threshold := time.Now().Add(-15 * time.Second)
				thresholdTimestamp := pgtype.Timestamp{Time: threshold, Valid: true}
				activeWorkers, err := queries.GetActiveWorkers(ctx, thresholdTimestamp)
				if err == nil {
					ActiveWorkersGauge.Set(float64(len(activeWorkers)))
				}

				// Update queue depth for "default"
				depth, err := queries.GetQueueDepth(ctx, "default")
				if err == nil {
					QueueDepthGauge.WithLabelValues("default").Set(float64(depth))
				}
			}
		}
	}()
}

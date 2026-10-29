package observability

import (
	"context"
	"log"
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

	JobSubmissionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasktide_jobs_submitted_total",
			Help: "Total number of submitted jobs",
		},
		[]string{"queue_name"},
	)

	JobExecutionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasktide_jobs_executed_total",
			Help: "Total number of executed jobs by outcome (succeeded, failed_retry, exhausted_dlq)",
		},
		[]string{"queue_name", "outcome"},
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

// StartMetricsCollector periodically updates the database-derived gauges (like queue depth and active worker count).
// It stops when ctx is cancelled.
func StartMetricsCollector(ctx context.Context, queries *db.Queries) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		// Queue names gauged on the previous tick, so depth series for
		// queues that go empty or become paused can be removed.
		lastQueues := map[string]bool{}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update active workers
				threshold := time.Now().Add(-15 * time.Second)
				thresholdTimestamp := pgtype.Timestamp{Time: threshold, Valid: true}
				activeWorkers, err := queries.GetActiveWorkers(ctx, thresholdTimestamp)
				if err != nil {
					log.Printf("MetricsCollector: failed to query active workers: %v\n", err)
				} else {
					ActiveWorkersGauge.Set(float64(len(activeWorkers)))
				}

				// Update queue depth for every ACTIVE queue (queues with
				// pending, non-paused work), not just "default". Enumerated
				// from the existing GetPendingJobs query since no
				// list-all-queues query exists yet.
				pending, err := queries.GetPendingJobs(ctx)
				if err != nil {
					log.Printf("MetricsCollector: failed to query pending jobs for queue depths: %v\n", err)
					continue
				}
				current := map[string]bool{}
				for _, j := range pending {
					if current[j.QueueName] {
						continue
					}
					current[j.QueueName] = true
					depth, err := queries.GetQueueDepth(ctx, j.QueueName)
					if err != nil {
						log.Printf("MetricsCollector: failed to query queue depth for %s: %v\n", j.QueueName, err)
						continue
					}
					QueueDepthGauge.WithLabelValues(j.QueueName).Set(float64(depth))
				}
				// Drop depth series for queues that no longer report pending work.
				for name := range lastQueues {
					if !current[name] {
						QueueDepthGauge.DeleteLabelValues(name)
					}
				}
				lastQueues = current
			}
		}
	}()
}

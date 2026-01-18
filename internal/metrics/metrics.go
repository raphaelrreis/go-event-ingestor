package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	EventsReceived  prometheus.Counter
	EventsPublished prometheus.Counter
	EventsFailed    prometheus.Counter
	IngestQueueSize prometheus.Gauge
	IngestLatency   prometheus.Histogram
	HTTPRequests    *prometheus.CounterVec
}

var (
	once     sync.Once
	instance *Metrics
)

func New() *Metrics {
	once.Do(func() {
		instance = &Metrics{
			EventsReceived: promauto.NewCounter(prometheus.CounterOpts{
				Name: "events_received_total",
				Help: "Total number of events received via HTTP",
			}),
			EventsPublished: promauto.NewCounter(prometheus.CounterOpts{
				Name: "events_published_total",
				Help: "Total number of events successfully published to Kafka",
			}),
			EventsFailed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "events_failed_total",
				Help: "Total number of events failed to process",
			}),
			IngestQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "ingest_queue_size",
				Help: "Current number of events in the internal buffer",
			}),
			IngestLatency: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "ingest_latency_ms",
				Help:    "Latency of event processing from ingestion to publish",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			}),
			HTTPRequests: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests by status code",
			}, []string{"status"}),
		}
	})
	return instance
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts all proxy requests by schema and status
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wireframez_requests_total",
			Help: "Total number of proxy requests",
		},
		[]string{"schema", "status"},
	)

	// TranscodingDuration tracks how long JSON->binary transcoding takes
	TranscodingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wireframez_transcoding_duration_seconds",
			Help:    "Time spent transcoding JSON to binary",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"schema"},
	)

	// PayloadSizeJSON tracks incoming JSON payload sizes
	PayloadSizeJSON = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wireframez_payload_size_json_bytes",
			Help:    "Size of incoming JSON payloads in bytes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10),
		},
		[]string{"schema"},
	)

	// PayloadSizeBinary tracks outgoing binary payload sizes
	PayloadSizeBinary = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wireframez_payload_size_binary_bytes",
			Help:    "Size of outgoing binary payloads in bytes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10),
		},
		[]string{"schema"},
	)

	// CompressionRatio tracks the ratio of binary size to JSON size
	CompressionRatio = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wireframez_compression_ratio",
			Help:    "Ratio of binary payload size to JSON payload size",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10),
		},
		[]string{"schema"},
	)

	// ActiveConnections tracks current in-flight requests
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wireframez_active_connections",
			Help: "Number of currently active proxy connections",
		},
	)

	// SchemaLookupDuration tracks schema registry lookup times
	SchemaLookupDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wireframez_schema_lookup_duration_seconds",
			Help:    "Time spent looking up schemas from the registry",
			Buckets: prometheus.DefBuckets,
		},
	)
)
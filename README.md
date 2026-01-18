# Go Event Ingestor

High-performance, fault-tolerant HTTP-to-Kafka event ingestion service written in Go.

## 🚀 Overview

This service acts as a gateway for ingesting high-volume events via HTTP and reliably publishing them to Apache Kafka. It is designed to handle backpressure, graceful degradation, and production-grade observability.

### Key Features

- **High Throughput**: Uses worker pools and batching for efficient Kafka writes.
- **Fault Tolerance**: Implements a Dead Letter Queue (DLQ) for failed messages.
- **Backpressure**: Rejects requests (HTTP 503) when internal buffers are full to prevent OOM.
- **Rate Limiting**: Token bucket rate limiter to protect downstream systems.
- **Observability**: Prometheus metrics and structured JSON logging (`log/slog`).
- **Graceful Shutdown**: Ensures in-flight requests and buffered events are processed before exiting.

## 🏗 Architecture

1.  **HTTP Layer**: Receives JSON events, validates payload, and enforces rate limits.
2.  **Ingestion Service**: Buffers events in a bounded channel.
3.  **Worker Pool**: Multiple concurrent workers consume the channel and publish to Kafka.
4.  **Kafka Producer**:
    *   **Main Topic**: Primary destination for events.
    *   **DLQ Topic**: Fallback for events that fail to publish after retries.

## 📂 Folder Structure

```
.
├── cmd/
│   └── ingestor/       # Application entrypoint
├── internal/
│   ├── config/         # Configuration loading
│   ├── http/           # HTTP handlers
│   ├── ingest/         # Core business logic (worker pool)
│   ├── kafka/          # Kafka producer wrapper
│   ├── metrics/        # Prometheus metrics definition
│   ├── model/          # Data models
│   └── rate/           # Rate limiter implementation
├── pkg/
│   └── logger/         # Structured logger setup
├── tests/              # Integration and benchmarks
├── Dockerfile          # Multi-stage Docker build
├── Makefile            # Build and maintenance commands
└── go.mod              # Go module definition
```

## 🛠 Getting Started

### Prerequisites

- Go 1.23+
- Docker & Docker Compose (optional, for dependencies)
- A running Kafka instance

### Configuration

The application is configured via environment variables.

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | 8080 | Port to listen on |
| `LOG_LEVEL` | INFO | Log level (DEBUG, INFO, WARN, ERROR) |
| `KAFKA_BROKERS` | localhost:9092 | Comma-separated list of Kafka brokers |
| `KAFKA_TOPIC` | events | Main Kafka topic |
| `KAFKA_DLQ_TOPIC` | events-dlq | Dead Letter Queue topic |
| `WORKER_POOL_SIZE` | 10 | Number of concurrent Kafka publishers |
| `QUEUE_SIZE` | 1000 | Size of internal buffer channel |
| `RATE_LIMIT_RPS` | 1000 | Requests per second limit |
| `RATE_LIMIT_BURST` | 100 | Burst size for rate limiter |

### Running Locally

1.  **Install dependencies**:
    ```bash
    make tidy
    ```

2.  **Build and Run**:
    ```bash
    make run
    ```

3.  **Send a Test Event**:
    ```bash
    curl -i -X POST http://localhost:8080/events \
      -H "Content-Type: application/json" \
      -d '{"type": "click", "payload": {"user_id": 123}}'
    ```

### Running with Docker

```bash
make docker-build
make docker-run
```

## 📊 Observability

### Metrics

The service exposes Prometheus metrics at `/metrics`.

- `events_received_total`: Total events accepted via HTTP.
- `events_published_total`: Events successfully sent to Kafka.
- `events_failed_total`: Events failed (sent to DLQ or dropped).
- `ingest_queue_size`: Current usage of the internal buffer.
- `ingest_latency_ms`: Histogram of end-to-end processing time.
- `http_requests_total`: HTTP request counts by status code.

### Logging

Logs are written to `stdout` in JSON format for easy integration with ELK/Splunk.

## ⚖️ Trade-offs

- **Channel-based Buffer**: We use an in-memory channel. If the pod crashes hard, buffered events are lost. For zero-data-loss requirements, a persistent write-ahead log (WAL) or direct-to-Kafka (synchronous) mode would be needed, at the cost of latency.
- **At-least-once Delivery**: In rare failure scenarios (network partition during ack), duplicates might occur in Kafka. Downstream consumers should be idempotent.
- **No TLS Config**: Currently disabled for simplicity. Production deployment should enable TLS/SASL in `internal/kafka/producer.go`.

## 📝 License

MIT
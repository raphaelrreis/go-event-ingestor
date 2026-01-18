# High-Throughput Event Ingestor

High-performance event ingestion service written in Go. Designed for reliability, backpressure handling, and observability.

## 🏗 Architecture

```
HTTP Request -> [Rate Limiter] -> [Handler] -> [Buffered Channel (Queue)]
                                                     |
                                                     v
                                              [Worker Pool]
                                                     |
                                      +--------------+--------------+
                                      |                             |
                                      v                             v
                                [Kafka Producer]             [DLQ Producer]
                                      |                             |
                                  (Success)                      (Failure)
```

## 🚀 Features

- **Backpressure**: Returns `503 Service Unavailable` when internal queues are full.
- **Rate Limiting**: Token bucket algorithm to protect resources.
- **Reliability**: Automatic retries and Dead Letter Queue (DLQ) for failed messages.
- **Observability**: Prometheus metrics and structured JSON logging.
- **Concurrency**: Worker pool pattern for high-throughput processing.

## 🛠 Configuration

| Env Variable | Default | Description |
|--------------|---------|-------------|
| `HTTP_PORT` | `8080` | Service port |
| `LOG_LEVEL` | `INFO` | DEBUG, INFO, WARN, ERROR |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated brokers |
| `KAFKA_TOPIC` | `events` | Main topic |
| `WORKER_POOL_SIZE` | `10` | Number of concurrent workers |
| `QUEUE_SIZE` | `1000` | Internal buffer size |

## 🏃 Running Local

1. **Start dependencies (Kafka):**
   ```bash
   docker-compose up -d
   ```

2. **Run the service:**
   ```bash
   go run cmd/ingestor/main.go
   ```

3. **Send an event:**
   ```bash
   curl -X POST http://localhost:8080/events \
     -H "Content-Type: application/json" \
     -d '{"type": "click", "payload": {"user_id": 123}}'
   ```

## 📊 Metrics

Metrics are available at `http://localhost:8080/metrics`.

- `events_received_total`: Total events hitting the API.
- `events_published_total`: Successfully sent to Kafka.
- `ingest_queue_size`: Current internal queue depth.
- `ingest_latency_ms`: Histogram of processing time.

## 🧪 Testing

```bash
# Unit & Integration Tests
go test ./...

# Benchmarks
go test ./... -bench=. -benchmem
```

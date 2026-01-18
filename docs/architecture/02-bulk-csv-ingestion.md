# RFC 02: Bulk CSV Ingestion Pipeline

| Status        | Draft |
| :---          | :--- |
| **Author**    | Senior Staff Software Engineer |
| **Feature**   | Bulk CSV Ingestion |
| **Branch**    | feature/02-bulk-csv-ingestion |
| **Date**      | 2026-01-18 |

---

## 1. Motivation

Enterprise environments frequently deal with legacy systems or data dumps where the primary exchange format is CSV (Comma Separated Values). These files can range from megabytes to gigabytes (millions of rows).

The current HTTP-based ingestion is optimized for real-time, low-latency events. It is ill-suited for:
- Historical backfills.
- Large batch transfers.
- Resuming interrupted large transfers.

**Problem Statement**:
We need a robust, memory-efficient mechanism to ingest large CSV files, convert them into events, and publish them to Kafka without overwhelming the system or losing data upon crash.

## 2. Architecture

### 2.1 High-Level Flow

The system follows a classic **Producer-Consumer** pattern with a bounded buffer for backpressure.

```ascii
[CSV Source] 
    | (1) Read Stream (Line-by-Line)
    v
[CSV Reader (Producer)] 
    | (2) Send Job (Bounded Channel)
    v
[Worker Pool (Consumers)] 
    | (3) Transform & Publish
    v
[Kafka Producer] -> (Topic: events)
```

1.  **CSV Source**: Abstracted interface (Local, S3, GCS, Azure Blob).
2.  **CSV Reader**: Reads file line-by-line using `encoding/csv` and `bufio`. Maintains zero-copy semantics where possible.
3.  **Worker Pool**: Configurable number of goroutines. Consumes lines, validates, transforms to `Event`, and publishes.
4.  **Kafka Producer**: Asynchronous writes to Kafka.

### 2.2 Core Components

#### File Abstraction
To ensure cloud agnosticism, we define a `Source` interface. This allows us to swap storage backends without changing the ingestion logic.

#### State Management (Checkpointing)
To enable resumability, we track the **read offset** (byte offset or line number).
- **Strategy**: Checkpoint every N lines or Time T.
- **Storage**: Ideally a small KV store (e.g., Redis, Postgres, or a sidecar file `.meta`). For MVP, we may start with a local state file.

## 3. Concurrency Model

### 3.1 Streaming & Memory Safety
- **Avoid `ioutil.ReadFile`**: Never load the whole file.
- **Use `bufio.Scanner` / `csv.Reader`**: Stream processing row by row.
- **Memory Footprint**: `O(1)` relative to file size. Memory usage depends on `WorkerCount + ChannelSize`.

### 3.2 Worker Pool & Backpressure
- **Uncontrolled Parallelism is bad**: Spawning a goroutine per line is an anti-pattern (thrashing, OOM).
- **Fixed Pool**: We use a Semaphore pattern or a fixed number of worker goroutines (e.g., `runtime.NumCPU() * 2`).
- **Channel Blocking**: If Kafka is slow, the worker pool blocks. If workers block, the channel fills up. If the channel fills, the CSV Reader blocks. This provides natural, propagation-based backpressure.

## 4. Failure Handling

### 4.1 At-Least-Once Delivery
The system guarantees that every row is processed at least once.
- If the process crashes at line 1000, and the last checkpoint was 900, lines 900-1000 will be reprocessed upon restart.
- Downstream consumers **must** be idempotent.

### 4.2 Error Strategy
1.  **Parsing Errors (Row Level)**:
    - Log error to a separate `errors.log`.
    - Increment `csv_processing_errors_total`.
    - Continue processing (do not abort entire file for one bad row).
2.  **System Errors (Network/Kafka)**:
    - Retry locally (exponential backoff).
    - If retry budget exhausted -> **Fatal Error**.
    - Stop workers.
    - Log fatal state.
    - Do NOT mark file as completed.

## 5. File Lifecycle (State Machine)

A file transitions through these states:

1.  **PENDING**: Discovered but not started.
2.  **IN_PROGRESS**: Currently being read. Checkpoints are updating.
3.  **COMPLETED**: EOF reached AND all workers finished successfully.
4.  **FAILED**: Fatal error occurred. Requires manual intervention or retry command.

## 6. Observability

### Metrics (Prometheus)
- `ingest_csv_lines_total`: Counter (status=read|processed|error).
- `ingest_csv_duration_seconds`: Histogram.
- `ingest_csv_current_offset`: Gauge (for progress tracking).

### Logging
- **Info**: "Started processing file X", "Finished file X (Duration: Y, Rows: Z)".
- **Debug**: "Checkpoint saved at offset N".
- **Error**: "Failed to parse row N: invalid format".

## 7. Trade-offs

| Decision | Trade-off |
| :--- | :--- |
| **Go Channels** | + Simplicity, synchronization. <br> - Slight overhead compared to raw ring buffers (negligible here). |
| **Checkpointing** | + Safety, resumability. <br> - I/O overhead if too frequent. |
| **Cloud Agnostic** | + No vendor lock-in. <br> - Cannot use cloud-specific optimized loaders (e.g., S3 Select). |

## 8. Why Go?
- **Streaming I/O**: Go's `io.Reader` interface is standard and powerful for streaming.
- **Concurrency**: Goroutines/Channels are the perfect primitive for Producer-Consumer pipelines.
- **Single Binary**: Easy to deploy as a CLI tool or sidecar in K8s.
- **Performance**: Near C++ performance with GC, vastly superior to Python for this specific CPU/IO-bound task.

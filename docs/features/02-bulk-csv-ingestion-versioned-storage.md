# RFC 02: Bulk CSV Ingestion with Versioned Storage

| Status        | Draft |
| :---          | :--- |
| **Feature**   | Bulk CSV Ingestion (Versioned Storage) |
| **Branch**    | feature/02-bulk-csv-ingestion-versioned-storage |
| **Date**      | 2026-01-18 |

---

## 1. Executive Summary

This RFC defines the architecture for ingesting massive CSV datasets (millions of rows) with a strict requirement for **historical data retention**.

Customers upload data periodically (e.g., monthly). Each upload represents a new "snapshot" of the data for that period. The system must process these files efficiently, publish events to Kafka, and strictly preserve all previous versions for audit, compliance, and reprocessing purposes.

The chosen strategy is **Path-Based Versioning (Option A)**, treating Object Storage (S3/GCS/Azure Blob) as an immutable Data Lake.

---

## 2. Business Context

### 2.1 Use Case
Our system ingests large batch files from enterprise customers. These files typically represent:
- Monthly financial statements
- Inventory snapshots
- Transaction logs
- User activity reports

### 2.2 Requirements
1.  **No Data Loss**: Once a file is uploaded, it must never be overwritten or deleted (until retention policy expires).
2.  **Audit Trail**: We must be able to prove what data was received and when.
3.  **Reprocessing**: We must be able to re-ingest data from any previous month (e.g., to fix a bug in the processing logic).
4.  **Cost Efficiency**: Current/hot data is accessed frequently; old/cold data is rarely accessed but must be kept cheap.
5.  **Simplicity**: The storage layout must be understandable by humans and standard tools without complex metadata databases.

---

## 3. Core Design Principles

### 3.1 Immutability
**Principle**: Uploaded files are effectively WORM (Write Once, Read Many).
**Justification**: Overwriting files leads to race conditions, lost data, and impossible audits. Every upload creates a distinct object.

### 3.2 Explicit Versioning (Path-Based)
**Principle**: The version (timestamp/month) is encoded in the file path.
**Justification**: Relying on S3 "Bucket Versioning" is hidden magic. Explicit paths make versioning visible, portable across clouds, and easy to query via standard tools (e.g., `aws s3 ls`).

### 3.3 Separation of Concerns
**Principle**:
- **Object Storage**: The single source of truth for raw data.
- **Ingestion Service**: Stateless processor.
- **State Store**: Metadata tracker (what has been processed).
**Justification**: Decoupling storage from processing allows independent scaling and simpler disaster recovery.

### 3.4 Cloud Agnosticism
**Principle**: The design must work identically on AWS S3, Google Cloud Storage, and Azure Blob Storage.
**Justification**: We avoid vendor-specific features (like S3 Select or Event Grid triggers) in the core logic to maintain portability.

---

## 4. Storage Strategy (Option A: Path Versioning)

We adopt a directory structure based on Hive-style partitioning.

### 4.1 Canonical Bucket Layout

```text
bucket-name/
├── customer_id=123/
│   ├── dataset=orders/
│   │   ├── year=2024/
│   │   │   ├── month=01/
│   │   │   │   └── orders_2024_01.csv
│   │   │   ├── month=02/
│   │   │   │   └── orders_2024_02.csv
│   │   ├── year=2025/
│   │   │   ├── month=01/
│   │   │   │   └── orders_2025_01.csv
│   │   │
│   │   └── _current.json  <-- Pointer to active dataset
```

### 4.2 Benefits
- **Time-Travel**: "What did the data look like in Jan 2024?" -> Open `year=2024/month=01`.
- **Lifecycle Policies**: It is trivial to configure rules like "Move `year=2023/*` to Glacier/Coldline".
- **Collision Avoidance**: Since paths include the period, collisions only occur if a customer uploads the same month twice (which is handled by timestamping or rejecting).

---

## 5. Current Data Pointer (Manifest)

To avoid scanning thousands of files to find the "latest", we maintain a lightweight manifest file: `_current.json`.

### 5.1 Manifest Schema
```json
{
  "customer_id": "123",
  "dataset": "orders",
  "year": 2025,
  "month": 1,
  "path": "year=2025/month=01/orders_2025_01.csv",
  "uploaded_at": "2025-01-31T22:10:00Z",
  "checksum": "sha256:abc123..."
}
```

### 5.2 Usage
- **Reading**: The application reads `_current.json` to know which file to process for the "current view".
- **Updating**: When a new file is uploaded successfully, `_current.json` is updated atomically.
- **History**: Updating the manifest does NOT delete the old files. They remain in their year/month folders.

---

## 6. Ingestion Process Flow

1.  **Upload**: Client uploads `orders_2025_02.csv` to `.../year=2025/month=02/`.
2.  **Validation**: System checks file integrity (checksum) and basic schema.
3.  **Activation**: System updates `_current.json` to point to the new file.
4.  **Job Trigger**: An event triggers the Go Event Ingestor.
5.  **Processing**:
    - Reader opens the file stream (from Object Storage).
    - Rows are read line-by-line.
    - Workers transform rows to Events.
    - Events are published to Kafka.
6.  **Checkpointing**: Progress is saved periodically.
7.  **Completion**: Job marked `COMPLETED` in the state store.

---

## 7. Ingestion State Management

We do NOT use Object Storage metadata to track processing state (it's slow and eventually consistent). We use a relational DB or KV store.

### 7.1 State Model
| Field | Type | Description |
|---|---|---|
| `job_id` | UUID | Unique run ID |
| `customer_id` | String | Owner of data |
| `dataset` | String | Type of data (orders, transactions) |
| `period` | String | "2025-01" |
| `file_path` | String | Full path in bucket |
| `status` | Enum | PENDING, IN_PROGRESS, COMPLETED, FAILED |
| `checkpoint` | Long | Last successfully processed byte/row offset |
| `updated_at` | Timestamp | Last heartbeat |

### 7.2 Why this matters?
- **Idempotency**: Before starting, check if `file_path` is already `COMPLETED`.
- **Resumability**: If `FAILED` or `IN_PROGRESS` (stale), resume from `checkpoint`.
- **Observability**: Easy to query "Which jobs failed yesterday?".

---

## 8. Failure & Recovery Model

### 8.1 Processing Semantics
**At-Least-Once Delivery**: We prioritize data safety. If a crash occurs, we may reprocess a small batch of rows. Downstream consumers must handle duplicates.

### 8.2 Scenarios
- **Bad Row (Parsing Error)**:
    - Log error.
    - Increment error counter.
    - Continue processing (do not abort mostly-valid files).
- **Transient Failure (Network/Kafka)**:
    - Retry with exponential backoff.
- **Fatal Failure (Pod Crash/OOM)**:
    - Process dies.
    - State remains `IN_PROGRESS` (eventually becomes stale).
    - Watchdog/Supervisor restarts job.
    - New process reads `checkpoint` and resumes.

---

## 9. Terraform Responsibilities

We treat infrastructure as immutable.

### ✅ Terraform Manages:
- **Bucket Creation**: S3/GCS/Azure Storage.
- **Lifecycle Rules**: e.g., "Transition objects older than 365 days to Archive".
- **Encryption**: KMS/CMK configuration.
- **IAM**: Service accounts and role bindings.

### ❌ Terraform Does NOT Manage:
- **Data Uploads**: No CSV files in Terraform state.
- **Manifest Updates**: Application logic, not infra logic.
- **Schema Management**: Handled by the application.

---

## 10. Non-Goals (Out of Scope)

- **Overwriting**: We never overwrite files. We upload new versions.
- **Real-time Streaming**: This is a batch process.
- **In-place Updates**: We do not edit CSV rows inside the bucket.
- **Analytics**: This system moves data; it does not query it (use Snowflake/BigQuery for that).
- **Exactly-Once**: Too expensive. We accept at-least-once.

---

## 11. Architecture Summary

```ascii
[Upload Client] 
      | (1) Upload CSV
      v
[Object Storage (S3/GCS/Azure)] <------ (4) Stream Data
      |                                        ^
      +-> /year=2025/month=01/*.csv            |
      +-> /_current.json                       |
                                               |
[State Store (DB)] <--- (3) Create Job --- [Go Ingestor]
      ^                                        |
      +------- (5) Update Progress ------------+
                                               | (6) Publish
                                               v
                                            [Kafka]
```

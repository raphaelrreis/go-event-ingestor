# RFC 03: Bulk CSV Ingestion with Versioned Storage (Design)

| Status        | Draft |
| :---          | :--- |
| **Feature**   | Bulk CSV Ingestion with Historical Retention |
| **Branch**    | feature/03-bulk-csv-ingestion-versioned-storage |
| **Type**      | Design / Architecture |
| **Date**      | 2026-01-18 |

---

## 1. Executive Summary

This architecture design enables the **Go Event Ingestor** to handle massive CSV datasets (millions of rows) uploaded periodically (e.g., monthly) by enterprise customers.

The solution prioritizes **data integrity**, **auditability**, and **historical retention**. It treats Object Storage (S3/GCS/Azure Blob) as an immutable Data Lake where every upload creates a new versioned path, never overwriting existing data.

---

## 2. Business Context

### 2.1 Use Case
Large enterprise customers periodically upload "snapshots" of their business data to our platform. These are not real-time streams but bulk updates representing a specific period state.

**Typical Datasets:**
- **Financial Records:** Monthly ledger dumps.
- **Inventory:** End-of-month stock counts.
- **Transactions:** Historical sales logs.

### 2.2 The Problem with "Overwriting"
In many simple systems, uploading `orders.csv` overwrites the previous `orders.csv`. This is **unacceptable** for this feature because:
1.  **Lost History:** We lose the state of the data from last month.
2.  **No Audit:** We cannot prove what data was processed 6 months ago.
3.  **Race Conditions:** If an upload happens during processing, the ingestor might read corrupted data.

### 2.3 Explicit Business Requirements
*   **Zero Data Loss:** Every file ever uploaded must be preserved (until lifecycle policy expires).
*   **Full Audit Trail:** "Show me the exact file processed in Jan 2024."
*   **Reprocessing Capability:** We must be able to re-run ingestion for a past period if a bug is found in our logic.
*   **Cost Predictability:** Hot storage for current data, Cold/Archive for history.

---

## 3. Core Design Principles

### 3.1 Immutability
**Rule:** Files are WORM (Write Once, Read Many).
**Justification:** Immutability simplifies concurrency control. We don't need file locking if we never update a file in place.

### 3.2 Explicit Path Versioning
**Rule:** The version is embedded in the object key (path), e.g., `year=2025/month=01`.
**Justification:** S3 "Bucket Versioning" is hidden metadata. Path versioning makes the structure browsable, portable across clouds, and compatible with Big Data tools (Spark, Hive, Presto) that understand partitioning.

### 3.3 Separation of Concerns
**Rule:**
- **Storage Layer**: Dumb, immutable repository of bytes.
- **Processing Layer**: Stateless Go workers.
- **State Layer**: SQL/KV store tracking "what have I processed?".
**Justification:** This allows us to scale ingestion workers independently of storage size.

### 3.4 Cloud Agnosticism
**Rule:** The design uses standard Blob Storage primitives (Put, Get, List).
**Justification:** Avoids vendor lock-in. S3, GCS, and Azure Blob all support this hierarchy perfectly.

---

## 4. Storage Strategy (Option A: Path Versioning)

We adopt a **Hive-style Partitioning** layout.

### 4.1 Canonical Bucket Layout

```text
bucket-name/
├── customer_id=123/
│   ├── dataset=orders/
│   │   ├── year=2024/
│   │   │   ├── month=01/
│   │   │   │   └── orders_2024_01.csv  <-- Immutable Object
│   │   │   ├── month=02/
│   │   │   │   └── orders_2024_02.csv
│   │   ├── year=2025/
│   │   │   ├── month=01/
│   │   │   │   └── orders_2025_01.csv
│   │   │
│   │   └── _current.json               <-- Mutable Pointer (Manifest)
```

### 4.2 Benefits of this Layout
1.  **Time-Travel**: To see the state of the world in Feb 2024, we simply read `year=2024/month=02`.
2.  **Cost Optimization**: Lifecycle rules can easily target `year=2023/*` to move to Glacier/Coldline.
3.  **Human Readable**: An SRE can verify data presence just by listing the path.

---

## 5. Current Data Pointer (Manifest)

Since we have many versions, how does the system know which one is the "Active" dataset?
We introduce a lightweight JSON manifest: `_current.json`.

### 5.1 Manifest Example
**Path:** `bucket-name/customer_id=123/dataset=orders/_current.json`

```json
{
  "customer_id": "123",
  "dataset": "orders",
  "year": 2025,
  "month": 1,
  "path": "year=2025/month=01/orders_2025_01.csv",
  "uploaded_at": "2025-01-31T22:10:00Z",
  "checksum": "sha256:abc123789..."
}
```

### 5.2 Mechanics
- **Writing**: When a customer upload completes and passes validation, the system atomically updates `_current.json`.
- **Reading**: The ingestion service reads `_current.json` to find the target file path.
- **Safety**: Updating the manifest is the "commit" operation. The old files are safely ignored but preserved.

---

## 6. Ingestion Process Flow

1.  **Upload**: Client uploads `orders_2025_03.csv` to `.../year=2025/month=03/`.
2.  **Validation**:
    - Check file size.
    - Validate SHA256 checksum.
    - Check CSV header compatibility.
3.  **Activation**: System updates `_current.json` to point to `month=03`.
4.  **Job Trigger**: An event (SQS/PubSub) notifies the Go Event Ingestor.
5.  **Processing (Go Application)**:
    - Reads manifest.
    - Opens stream to S3/GCS.
    - `csv.Reader` streams rows.
    - Worker pool converts rows to Events.
    - Events published to Kafka.
6.  **Completion**: Job status updated to `COMPLETED`.

---

## 7. Ingestion State Management

We do NOT rely on S3 metadata or file renaming to track progress. We use a dedicated State Store (PostgreSQL or Redis).

### 7.1 Data Model

| Field | Description |
|---|---|
| `job_id` | Unique UUID for this run. |
| `customer_id` | Partition key. |
| `dataset` | "orders", "inventory". |
| `period` | "2025-01". |
| `file_path` | Full object key in bucket. |
| `status` | `PENDING`, `IN_PROGRESS`, `COMPLETED`, `FAILED`. |
| `last_offset` | Byte/Row offset for resume. |
| `started_at` | Timestamp. |

### 7.2 Why external state?
- **Resumability**: If the pod crashes at row 5,000,000, the new pod reads `last_offset` from DB and seeks to that position in the CSV stream.
- **Deduplication**: Before starting, the system checks: *Has this file path already been successfully ingested?*

---

## 8. Failure & Recovery Model

### 8.1 Processing Semantics
**At-Least-Once Delivery**: We guarantee all rows reach Kafka. In a crash/resume scenario, a small window of rows might be re-published. Downstream consumers must handle idempotency.

### 8.2 Failure Scenarios
1.  **Bad CSV Row**:
    - Log error details.
    - Increment `csv_parsing_errors` metric.
    - **Continue**: Do not abort a 10GB file for 1 bad character.
2.  **Kafka Unavailable**:
    - Retry locally (exponential backoff).
    - If exhausted, mark Job as `FAILED`.
    - **Resume**: Operator triggers retry; system resumes from last checkpoint.
3.  **Fatal Crash (OOM)**:
    - Kubernetes restarts pod.
    - App sees `IN_PROGRESS` job with stale heartbeat.
    - App resumes processing from `last_offset`.

---

## 9. Terraform Responsibilities

Terraform manages the **Container** (Buckets), not the **Content** (Files).

### ✅ Terraform Scope
- **Bucket Creation**: `aws_s3_bucket`, `google_storage_bucket`.
- **Lifecycle Rules**:
    - `Transition to IA after 30 days`.
    - `Transition to Glacier after 365 days`.
    - `Expire/Delete after 7 years`.
- **IAM Policies**: Granting the Go App `GetObject` permission.
- **Server-Side Encryption**: Enabling KMS.

### ❌ Application Scope
- Uploading the actual CSV files.
- Generating the `year=...` directory structure.
- Writing `_current.json`.

---

## 10. Non-Goals

- **Real-time Streaming**: This is a batch system. Latency is minutes/hours, not milliseconds.
- **Data Editing**: We do not support "updating row 50" inside a CSV. Upload a new file instead.
- **Analytics Querying**: This system ingests data into Kafka. It is NOT a replacement for Snowflake, BigQuery, or Athena.

---

## 11. Architecture Summary

```ascii
[Client] 
   | (Upload)
   v
[Object Storage (Immutable Data Lake)]
   |
   +-- /customer/dataset/year=2024/month=01/data.csv (Archived)
   +-- /customer/dataset/year=2025/month=01/data.csv (Active)
   +-- /customer/dataset/_current.json (Pointer)
   
[Ingestion Service (Go)]
   | (1) Read Manifest
   | (2) Stream Active CSV
   | (3) Checkpoint to DB
   v
[Kafka] -> [Downstream Consumers]
```

# Go Event Ingestor (Multi-Cloud)

A production-grade, high-performance event ingestion service written in Go, designed to demonstrate **cloud-agnostic architecture**, **infrastructure as code (IaC)**, and **professional CI/CD workflows**.

This project serves as a reference implementation for backend engineers looking to master:
- **Go** (Concurrency, Channels, Interfaces, HTTP, Kafka)
- **Kubernetes** (Stateless deployment, Probes, ConfigMaps)
- **Terraform** (Modules, Multi-cloud providers)
- **Observability** (Prometheus Metrics, Structured Logging)

---

## 🚀 Key Features

### Application Layer
- **High Throughput**: Optimized worker pool pattern for concurrent event processing.
- **Fault Tolerance**: Automatic retries and Dead Letter Queue (DLQ) for failed events.
- **Backpressure**: Intelligent queue management to prevent Out-Of-Memory (OOM) crashes.
- **Rate Limiting**: Token bucket algorithm to protect downstream systems.
- **Graceful Shutdown**: Ensures zero data loss during rolling updates.

### Infrastructure Layer
- **Multi-Cloud Support**: Deployable on AWS (EKS), GCP (GKE), and Azure (AKS).
- **IaC**: Fully managed via Terraform with reusable modules.
- **Containerized**: Optimized multi-stage Docker builds (distroless/alpine).

---

## 🏗 Architecture

The system follows a clean **Hexagonal Architecture** (Ports & Adapters) to decouple business logic from external dependencies.

```ascii
[Client] -> (HTTP/JSON) -> [API Handler]
                                |
                          [Rate Limiter]
                                |
                         [Ingest Service] -> (Channel Buffer)
                                |
                          [Worker Pool]
                           /    |    \
                  (Kafka) (Kafka) (Kafka)
                     |       |       |
                [Main Topic] |  [DLQ Topic]
```

### Core Components
1.  **HTTP API**: Validates incoming payloads and enforces rate limits.
2.  **Ingest Service**: Acts as a buffer/queue to absorb traffic spikes.
3.  **Workers**: Async consumers that push data to Kafka, handling retries and errors.
4.  **Observability**: Exposes `/metrics` for Prometheus and logs to `stdout` (JSON).

---

## 🛠 Developer Workflow

We enforce a strict **"Local == CI"** policy. What runs on your machine is exactly what runs in the pipeline.

### The Golden Rule
**NEVER push code without running:**
```bash
make ci
```

### Commands
| Command | Description |
|---|---|
| `make run` | Run the application locally (connects to local Kafka) |
| `make ci` | Run full validation (Format, Lint, Test, Build) |
| `make test` | Run unit tests |
| `make test-int` | Run integration tests (uses Docker/Testcontainers) |
| `make bench` | Run performance benchmarks |
| `docker-compose up` | Start local infrastructure (Kafka, Prometheus) |

---

## ☁️ Infrastructure & Deployment

The infrastructure is decoupled from the application code.
Please refer to the **[Infrastructure Documentation](infra/README.md)** for detailed guides on deploying to AWS, GCP, and Azure.

---

## 🧠 Design Decisions & Trade-offs

1.  **Channel-based Buffer**:
    *   *Decision*: Use an in-memory Go channel for buffering.
    *   *Trade-off*: Extremely fast, but risk of data loss if the pod crashes. For zero-data-loss, a Write-Ahead Log (WAL) or direct synchronous write would be required (at the cost of latency).

2.  **External Kafka**:
    *   *Decision*: Terraform does not provision Kafka.
    *   *Reason*: In production, you should use managed services (MSK, Confluent Cloud, Event Hubs). Provisioning stateful clusters via Terraform is complex and rarely done in app repositories.

3.  **Shared Terraform Modules**:
    *   *Decision*: Abstract Kubernetes manifests into a shared module.
    *   *Reason*: A Deployment definition is 99% identical across clouds. DRY (Don't Repeat Yourself) reduces maintenance.

## 📝 License
MIT
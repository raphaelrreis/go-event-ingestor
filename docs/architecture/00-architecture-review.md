# Architecture & Documentation Strategy Review

| Status | Approved |
| :--- | :--- |
| **Date** | 2026-01-18 |
| **Type** | Architecture & Infrastructure Review |
| **Scope** | Documentation, Multi-Cloud Parity, Terraform Strategy |

---

## 1. High-Level Summary

This document formalizes the architectural standards for the **Go Event Ingestor** project. It addresses the current mix of feature RFCs and infrastructure documentation by proposing a strict separation of concerns.

The review confirms the "Hub and Spoke" Terraform pattern is correct but identifies gaps in "Production Readiness" definition, particularly regarding state management and secret handling which are currently at "Educational" maturity levels.

---

## 2. Documentation Architecture

We enforce a strict separation between **What we build (Architecture)** and **How we deploy (Infrastructure)**.

### Proposed Structure

```
docs/
├── architecture/           # High-Level System Design & RFCs
│   ├── 00-architecture-review.md  <-- This file
│   ├── 01-hybrid-cloud-architecture.md
│   ├── 02-bulk-csv-ingestion.md
│   └── 03-bulk-csv-ingestion-versioned-storage.md
│
├── infrastructure/         # Concrete Cloud Implementation Details
│   ├── aws-eks-guide.md    # Specifics of EKS + VPC
│   ├── gcp-gke-guide.md    # Specifics of GKE + VPC
│   ├── azure-aks-guide.md  # Specifics of AKS + VNet
│   └── terraform-modules.md # Documentation of shared modules
```

### Rationale
1.  **Lifecycle Decoupling**: Application architecture (e.g., CSV processing logic) changes independently of Cloud Provider implementation details (e.g., how to provision GKE).
2.  **Target Audience**: Application developers read `architecture/`; SRE/Platform engineers read `infrastructure/`.
3.  **Avoid Duplication**: RFCs should reference infrastructure docs for deployment details, rather than repeating Terraform commands.

---

## 3. Cloud Capability Matrix

This matrix defines the **parity status** across the three supported clouds.

| Capability | AWS | GCP | Azure | Status | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Kubernetes** | EKS | GKE (Standard) | AKS | ✅ Implemented | Versions managed via Terraform variables. |
| **Registry** | ECR | Artifact Registry | ACR | ✅ Implemented | Image pull secrets handled via native integrations/IAM. |
| **Network** | VPC (Official Module) | VPC (Custom Resource) | Resource Group (Custom) | ⚠️ Partial | AWS uses verified modules; GCP/Azure use simpler resources. Acceptable for this scope. |
| **Identity** | IRSA (OIDC) | Workload Identity | Managed Identity | ⏳ Planned | Currently relies on node permissions/keys in some places. Need unification. |
| **State Backend** | S3 + DynamoDB | GCS | Blob Storage | ❌ Local Only | Currently using local state. Must move to remote for Prod. |
| **Ingress** | ALB Controller | GKE Ingress | App Gateway | ⏳ Planned | Currently using Service Type: LoadBalancer (L4). |
| **Observability**| CloudWatch | Cloud Operations | Azure Monitor | ⚠️ Partial | Basic integration enabled; Deep Prometheus integration varies. |

**Legend:**
- ✅ **Implemented**: Fully working in Terraform code.
- ⚠️ **Partial**: Working but simplified (e.g., simplified networking).
- ⏳ **Planned**: Design exists but code not yet fully mature.
- ❌ **Local Only**: Currently configured for local development/learning.

---

## 4. RFC Review & Gap Analysis

### RFC 01: Hybrid Cloud Architecture
- **Correct**: Hub & Spoke pattern for K8s apps.
- **Gap**: Does not specify **Cross-Cloud Connectivity**. How do clouds talk to each other? (VPN, Public Internet + mTLS?).
- **Action**: Explicitly declare connectivity as "Public Internet + TLS" for this phase.

### RFC 03: Bulk CSV Ingestion
- **Correct**: Path-based versioning and Manifest pattern.
- **Gap**: **Lifecycle Management** of the *State Store*. We mention a DB/KV for state, but don't provision it in Terraform.
- **Action**: Clarify that the State Store (Postgres/Redis) is currently "External Dependency" (like Kafka).

---

## 5. Infrastructure Best Practices

### Hub & Spoke Validation
- **Status**: **Approved**.
- **Reasoning**: Abstracting Kubernetes Deployments (`infra/modules/k8s-app`) is safe because K8s APIs are standardized. Abstracting Networking (VPC vs VNet) is **dangerous** and leads to leaky abstractions. Keeping network specific per cloud is the correct choice.

### Terraform State Strategy
- **Current**: Local state (`terraform.tfstate`).
- **Verdict**: Acceptable for **Level 1 (Educational)**. Unacceptable for **Level 2**.
- **Requirement**: Move to remote state buckets for any "Real World" usage to support team locking.

---

## 6. Minimal Production-Ready Baseline

To label this system "Production Ready", the following **MUST** be added:

1.  **Remote State**: No local tfstate.
2.  **Secret Management**: Replace Env Vars in Terraform with External Secret Store (Vault/AWS Secrets Manager) OR Sealed Secrets.
3.  **Workload Identity**: Zero static keys. Pods must authenticate to Cloud APIs (S3/GCS) via OIDC Federation.
4.  **Immutability**: Docker tags must be SHA digests or SemVer, never `latest`.

---

## 7. Roadmap & Maturity Levels

### Level 1: Functional / Educational (Current State)
- Local Terraform state.
- `latest` image tags.
- Public LoadBalancers.
- Basic Liveness/Readiness probes.
- **Goal**: Works on my machine & easy to demo.

### Level 2: Production-Ready (Next Milestone)
- Remote State with Locking.
- Semantic Versioning for Images.
- Structured Logging aggregation.
- HPA (Horizontal Pod Autoscaler) enabled.
- **Goal**: Stable for a single team.

### Level 3: Enterprise (Future)
- Multi-Region / Active-Active.
- Service Mesh (mTLS).
- GitOps (ArgoCD) instead of `terraform apply`.
- Policy-as-Code (OPA/Sentinel).
- **Goal**: Scale, Compliance, Zero-Trust.

---

## 8. Action Plan

1.  **Refactor Docs**: Move RFCs to `docs/architecture/` and create specific `docs/infrastructure/` guides.
2.  **Fix Versioning**: Remove `latest` tag usage in Terraform default variables.
3.  **Identity Spike**: Prototype Workload Identity for one cloud (AWS IRSA) to prove the pattern.

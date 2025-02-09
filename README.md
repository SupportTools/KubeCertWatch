<p align="center">
  <img src="https://cdn.support.tools/KubeCertWatch/logo-nobg.png">
</p>

# KubeCertWatch

`KubeCertWatch` is a Kubernetes-based tool for monitoring TLS secrets, certificates, and ingress objects to identify expiring or expired certificates. The tool runs as a pod in a Kubernetes cluster, providing visibility into the status of TLS secrets, and outputs metrics and a user-friendly status page for easy management.

[![Go Report Card](https://goreportcard.com/badge/github.com/supporttools/KubeCertWatch)](https://goreportcard.com/report/github.com/supporttools/KubeCertWatch)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

---

## Features

- **Comprehensive Certificate Monitoring**:
  - Monitors TLS secrets (`kubernetes.io/tls`) across all namespaces
  - Integrates with cert-manager to monitor Certificate resources
  - Tracks certificate expiration with detailed status reporting
  - Parallel processing for efficient cluster-wide scanning

- **Advanced Metrics & Monitoring**:
  - Rich Prometheus metrics for certificate health and status
  - Detailed expiration tracking with days-until-expiry metrics
  - Error tracking and operational metrics
  - Health check endpoint with service status

- **Robust Architecture**:
  - Automatic retries with exponential backoff
  - Concurrent certificate checking
  - Context-aware operations with timeouts
  - Thread-safe operations

- **User Interface & API**:
  - Clean status pages for TLS secrets and cert-manager certificates
  - RESTful API for programmatic access
  - Configurable refresh intervals
  - Search and filtering capabilities

- **Security & Best Practices**:
  - Proper RBAC permissions model
  - Secure certificate handling
  - Resource efficient operations
  - Kubernetes native design

---

## Getting Started

### Prerequisites

- A Kubernetes cluster
- Go (for building the binary locally)
- `kubectl` for deploying the application

---

### Installation

#### Using Helm (Recommended)

1. Add the Helm repository:
   ```bash
   helm repo add supporttools https://charts.support.tools
   helm repo update
   ```

2. Install KubeCertWatch:
   ```bash
   # Basic installation
   helm install kubecertwatch supporttools/kubecertwatch \
     --set settings.clusterName=my-cluster

   # With cert-manager integration
   helm install kubecertwatch supporttools/kubecertwatch \
     --set settings.clusterName=my-cluster \
     --set cert-manager.enabled=true
   ```

#### Manual Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/supporttools/KubeCertWatch.git
   cd KubeCertWatch
   ```

2. Build the Docker image:
   ```bash
   docker build -t supporttools/kubecertwatch:latest .
   ```

3. Apply RBAC and deploy:
   ```bash
   kubectl apply -f charts/KubeCertWatch/templates/rbac.yaml
   kubectl apply -f charts/KubeCertWatch/templates/deployment.yaml
   ```

---

### Configuration

#### Helm Chart Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `settings.debug` | Enable debug logging | `false` |
| `settings.metrics.enabled` | Enable Prometheus metrics | `true` |
| `settings.metrics.port` | Metrics server port | `9990` |
| `settings.cronSchedule` | Certificate check schedule | `0 */12 * * *` |
| `settings.clusterName` | Cluster name for metrics | Required |
| `cert-manager.enabled` | Enable cert-manager integration | `false` |

#### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEBUG` | Enable debug logging | `false` |
| `METRICS_PORT` | Port for metrics server | `9990` |
| `CRON_SCHEDULE` | Check schedule (cron format) | `0 */12 * * *` |
| `CLUSTER_NAME` | Cluster name for metrics | Required |

---

### Usage

1. Access the application via the HTTP API:

   - **Metrics**: [http://localhost:8080/metrics](http://localhost:8080/metrics)
   - **Health Check**: [http://localhost:8080/healthz](http://localhost:8080/healthz)
   - **Version Info**: [http://localhost:8080/version](http://localhost:8080/version)
   - **Status Page**: [http://localhost:8080/secrets/status](http://localhost:8080/secrets/status)

2. Trigger a manual TLS secret check:

   ```bash
   curl http://localhost:8080/check/secrets
   ```

---

### Status Page

The `/secrets/status` endpoint provides a summary table:

| Namespace   | Secret Name  | Expiration Date | Days Until | Status        |
|-------------|--------------|-----------------|------------|---------------|
| default     | my-secret    | 2025-01-15      | 13         | valid         |
| kube-system | kube-secret  | 2025-01-08      | 6          | expiring soon |
| test-ns     | test-secret  | 2023-12-25      | -7         | expired       |

Features:
- **Search**: Filter secrets by name or namespace.
- **Sorting**: Sort by any column, including `Days Until` or `Status`.

---

### Prometheus Metrics

The following metrics are exposed:

- **Timing Metrics**:
  - `last_check_time{check_name="tls-secret-check"}`: Timestamp of last TLS secret check
  - `last_check_time{check_name="cert-manager-check"}`: Timestamp of last cert-manager check

- **Error Metrics**:
  - `certificate_check_errors_total{check_type="tls-secrets",error_type="check_error"}`: TLS secret check errors
  - `certificate_check_errors_total{check_type="cert-manager",error_type="check_error"}`: Cert-manager check errors

- **Certificate Status**:
  - `certificate_expiry_days{namespace="",secret_name=""}`: Days until certificate expiration

---

### Development

1. Install dependencies:

   ```bash
   go mod tidy
   ```

2. Run the application locally:

   ```bash
   go run main.go
   ```

3. Run tests:

   ```bash
   go test ./...
   ```

---

### Health Monitoring

The application exposes several health-related endpoints:

- `/health`: Overall service health status
- `/metrics`: Prometheus metrics endpoint
- `/readyz`: Readiness probe endpoint
- `/status/secrets`: TLS secrets status page
- `/status/certificates`: Cert-manager certificates status page

### Roadmap

- Add Kubernetes Events integration for certificate status changes
- Implement alert manager integration for expiration notifications
- Add support for custom certificate authorities
- Enhance metrics with certificate chain validation

---

### License

This project is licensed under the [Apache 2.0 License](LICENSE).

---

### Contributions

Contributions are welcome! Please submit a pull request or file an issue for any bugs or feature requests.

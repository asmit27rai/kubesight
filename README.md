# KubeSight - Approximate Query Engine

[![Go Version](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://docker.com)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Compatible-green.svg)](https://kubernetes.io)

**KubeSight** is a high-performance approximate query engine designed specifically for Kubernetes observability data. It delivers **10-100x faster** query performance with **90-95% accuracy** using advanced probabilistic data structures and intelligent sampling techniques.

## Overview
Traditional observability queries on Kubernetes metrics can take hours to complete on large datasets. KubeSight solves this by providing approximate answers in milliseconds while maintaining business-grade accuracy for trend analysis and decision making.

## Features
- **High Performance**: 10-100x faster than exact queries with sub-second response times
- **Probabilistic Algorithms**: HyperLogLog, Count-Min Sketch, and Bloom Filters for efficient approximation
- **Real-time Processing**: Stream processing for live Kubernetes metrics via Kafka
- **SQL-like Interface**: Familiar query syntax with automatic error bounds
- **Kubernetes Native**: Designed specifically for container and pod metrics
- **Production Ready**: Complete monitoring stack with Prometheus and Grafana integration

## Quick Start

### Prerequisites

- **Go 1.22+**
- **Docker & Docker Compose**
- **kubectl** (for K8s deployment)
- **Make** (recommended)

### 1. Clone and Setup

```bash
git clone https://github.com/asmit27rai/kubesight.git
cd kubesight
```

### 2. Build The Server

```bash
make build
```

### 3. Run The Server

```bash
make run
```

### 4. Start All Services

```bash
make docker-up
```

This starts:
- **KubeSight Server**: http://localhost:8080
- **Kafka UI**: http://localhost:8081
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### 5. Start Grafana

```bash
make setup-grafana
```

### 6. Generate Some Random Queries To Check The Working

```bash
make generate-query
```

### 7. Stop All Services

```bash
make docker-down
```

### 8. Clean Your Environment

```bash
make clean
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Kubernetes    │───▶│  Stream Processor │───▶│  Probabilistic DS   │
│   Metrics/Logs  │    │  (Kafka + Flink)  │    │  (HLL, CMS, Bloom)  │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                                           │
┌─────────────────┐    ┌──────────────────┐              │
│   Dashboard     │◀───│   Query Engine   │◀─────────────┘
│   (Web UI)      │    │   (Go Service)   │
└─────────────────┘    └──────────────────┘
```

## Query Types

### Count Distinct Pods
```sql
COUNT_DISTINCT(pod_name) WHERE cluster_id='production'
```

### Percentile Queries
```sql
PERCENTILE(95, cpu_usage) WHERE namespace='web-services'
```

### Top-K Analysis
```sql
TOP_K(10, memory_usage) WHERE cluster_id='production'
```

### Sum And Average
```sql
SUM(network_bytes) WHERE timestamp > '1h ago'
AVG(response_time) WHERE service='api'
```

### Membership Testing
```sql
CONTAINS('pod-xyz-123') FROM pod_restarts
```

## API Reference

### Query Execution
```bash
POST /api/v1/query
Content-Type: application/json

{
  "query": "COUNT_DISTINCT(pod_name)",
  "query_type": "count_distinct",
  "filters": {
    "cluster_id": "production",
    "namespace": "default"
  }
}
```

### System Statistics
```bash
GET /api/v1/stats
```

### Health Check
```bash
GET /api/v1/health
```

### Generate Test Data
```bash
POST /api/v1/demo/generate
Content-Type: application/json

{
  "count": 10000,
  "cluster_id": "test-cluster"
}
```

## Config.yaml
```bash
server:
  host: "0.0.0.0"
  port: 8080

kafka:
  brokers: ["localhost:9092"]
  topics:
    metrics: "k8s-metrics"
    logs: "k8s-logs"
    events: "k8s-events"

sampling:
  default_rate: 0.05      # 5% base sampling
  incident_rate: 0.5      # 50% during anomalies
  reservoir_size: 10000
  window_size_min: 60
  adaptive_enabled: true

storage:
  hll_precision: 14       # ±1.6% error  
  cms_width: 2048
  cms_depth: 5
  bloom_size: 1000000
  bloom_hashes: 5
```

## Monitoring

### Prometheus Metrics
- `kubesight_queries_total` - Total queries processed
- `kubesight_query_duration_milliseconds` - Query latency distribution
- `kubesight_samples_total` - Total samples processed
- `kubesight_error_rate` - Approximation error rate
- `kubesight_memory_usage_bytes ` - Memory consumption

### Grafana Setup
```bash
make setup-grafana
```

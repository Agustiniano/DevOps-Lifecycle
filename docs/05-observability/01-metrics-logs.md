# Observability Stack - Logs, Metrics, Traces

## 🎯 Three Pillars of Observability

```
┌─────────────────────────────────────────────────────────┐
│                    OBSERVABILITY                        │
├──────────────┬──────────────────┬──────────────────────┤
│    LOGS      │     METRICS      │       TRACES         │
│              │                  │                      │
│ • Events     │ • Gauges         │ • Spans              │
│ • Errors     │ • Counters       │ • Dependencies       │
│ • Debug info │ • Histograms     │ • Latency breakdown  │
│              │                  │                      │
│   Loki       │   Prometheus     │      Tempo           │
└──────────────┴──────────────────┴──────────────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │     Grafana      │
              │  (Visualization) │
              └──────────────────┘
```

## 📊 Stack Tecnológico

| Componente | Herramienta | Propósito |
|------------|-------------|-----------|
| **Metrics** | Prometheus | Time-series database |
| **Logs** | Loki + Fluentd | Log aggregation |
| **Traces** | Tempo + OpenTelemetry | Distributed tracing |
| **Visualization** | Grafana | Dashboards & alerting |
| **Instrumentation** | OpenTelemetry | Unified observability |
| **Service Mesh** | Istio | Automatic instrumentation |

## 🔧 Prometheus Setup

### 1. Helm Installation

```bash
# Add Prometheus helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install kube-prometheus-stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  -f observability/prometheus/values.yaml
```

### 2. Values Configuration

```yaml
# observability/prometheus/values.yaml
prometheus:
  prometheusSpec:
    retention: 30d
    retentionSize: "50GB"
    
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 100Gi
          storageClassName: gp3
    
    resources:
      requests:
        memory: 4Gi
        cpu: 2000m
      limits:
        memory: 8Gi
        cpu: 4000m
    
    # Remote write para long-term storage
    remoteWrite:
      - url: https://prometheus-remote-write.example.com/api/v1/write
        basicAuth:
          username:
            name: prometheus-remote-write
            key: username
          password:
            name: prometheus-remote-write
            key: password
    
    # ServiceMonitor selectors
    serviceMonitorSelector:
      matchLabels:
        prometheus: monitoring
    
    # PodMonitor selectors
    podMonitorSelector:
      matchLabels:
        prometheus: monitoring
    
    # Additional scrape configs
    additionalScrapeConfigs:
      - job_name: 'kubernetes-pods'
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
            action: replace
            regex: ([^:]+)(?::\d+)?;(\d+)
            replacement: $1:$2
            target_label: __address__

grafana:
  adminPassword: changeme
  
  ingress:
    enabled: true
    hosts:
      - grafana.acme.com
    tls:
      - secretName: grafana-tls
        hosts:
          - grafana.acme.com
  
  datasources:
    datasources.yaml:
      apiVersion: 1
      datasources:
        - name: Prometheus
          type: prometheus
          url: http://prometheus-operated:9090
          isDefault: true
          
        - name: Loki
          type: loki
          url: http://loki:3100
          
        - name: Tempo
          type: tempo
          url: http://tempo:3100
          
  dashboardProviders:
    dashboardproviders.yaml:
      apiVersion: 1
      providers:
        - name: 'default'
          orgId: 1
          folder: ''
          type: file
          disableDeletion: false
          editable: true
          options:
            path: /var/lib/grafana/dashboards/default

  dashboards:
    default:
      kubernetes-cluster:
        gnetId: 7249
        revision: 1
        datasource: Prometheus
      
      istio-mesh:
        gnetId: 7639
        revision: 1
        datasource: Prometheus
```

### 3. Application Instrumentation (Go)

```go
// shared/libraries/go-commons/metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

var (
    // HTTP metrics
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "path"},
    )

    // Business metrics
    ordersCreated = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "orders_created_total",
            Help: "Total number of orders created",
        },
    )

    ordersValue = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "orders_value_dollars",
            Help:    "Distribution of order values",
            Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000},
        },
    )

    // Database metrics
    dbQueriesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "db_queries_total",
            Help: "Total number of database queries",
        },
        []string{"query_type", "table", "status"},
    )

    dbQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"query_type", "table"},
    )

    // Cache metrics
    cacheHitsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Total number of cache hits",
        },
        []string{"cache_name"},
    )

    cacheMissesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_misses_total",
            Help: "Total number of cache misses",
        },
        []string{"cache_name"},
    )
)

// HTTPMiddleware instrumenta requests HTTP
func HTTPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, r.URL.Path))
        defer timer.ObserveDuration()

        // Wrap ResponseWriter para capturar status code
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(rw, r)

        httpRequestsTotal.WithLabelValues(
            r.Method,
            r.URL.Path,
            http.StatusText(rw.statusCode),
        ).Inc()
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// MetricsHandler expone endpoint /metrics
func MetricsHandler() http.Handler {
    return promhttp.Handler()
}

// RecordOrder registra una orden creada
func RecordOrder(value float64) {
    ordersCreated.Inc()
    ordersValue.Observe(value)
}

// RecordDBQuery registra una query de base de datos
func RecordDBQuery(queryType, table string, duration float64, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }
    
    dbQueriesTotal.WithLabelValues(queryType, table, status).Inc()
    dbQueryDuration.WithLabelValues(queryType, table).Observe(duration)
}

// RecordCacheOperation registra operaciones de cache
func RecordCacheHit(cacheName string) {
    cacheHitsTotal.WithLabelValues(cacheName).Inc()
}

func RecordCacheMiss(cacheName string) {
    cacheMissesTotal.WithLabelValues(cacheName).Inc()
}
```

### 4. ServiceMonitor para Aplicación

```yaml
# gitops/environments/base/user-service/servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: user-service
  labels:
    app: user-service
    prometheus: monitoring
spec:
  selector:
    matchLabels:
      app: user-service
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
      scheme: http
```

### 5. PrometheusRules (Alertas)

```yaml
# observability/prometheus/alerts/application-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: application-alerts
  namespace: monitoring
spec:
  groups:
    - name: application
      interval: 30s
      rules:
        # High error rate
        - alert: HighErrorRate
          expr: |
            (
              sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)
              /
              sum(rate(http_requests_total[5m])) by (service)
            ) > 0.05
          for: 5m
          labels:
            severity: critical
            component: application
          annotations:
            summary: "High error rate on {{ $labels.service }}"
            description: "{{ $labels.service }} has error rate of {{ $value | humanizePercentage }}"

        # High latency
        - alert: HighLatency
          expr: |
            histogram_quantile(0.99,
              sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service)
            ) > 1
          for: 10m
          labels:
            severity: warning
            component: application
          annotations:
            summary: "High latency on {{ $labels.service }}"
            description: "{{ $labels.service }} P99 latency is {{ $value }}s"

        # Low availability
        - alert: ServiceDown
          expr: up{job="user-service"} == 0
          for: 2m
          labels:
            severity: critical
            component: application
          annotations:
            summary: "Service {{ $labels.job }} is down"
            description: "{{ $labels.job }} has been down for more than 2 minutes"

        # Database issues
        - alert: HighDatabaseQueryLatency
          expr: |
            histogram_quantile(0.99,
              sum(rate(db_query_duration_seconds_bucket[5m])) by (le, service)
            ) > 0.5
          for: 10m
          labels:
            severity: warning
            component: database
          annotations:
            summary: "High database query latency"
            description: "{{ $labels.service }} database P99 latency is {{ $value }}s"

        - alert: HighDatabaseErrorRate
          expr: |
            (
              sum(rate(db_queries_total{status="error"}[5m])) by (service)
              /
              sum(rate(db_queries_total[5m])) by (service)
            ) > 0.01
          for: 5m
          labels:
            severity: critical
            component: database
          annotations:
            summary: "High database error rate"
            description: "{{ $labels.service }} has {{ $value | humanizePercentage }} database errors"

        # Cache issues
        - alert: LowCacheHitRate
          expr: |
            (
              sum(rate(cache_hits_total[10m])) by (service, cache_name)
              /
              (
                sum(rate(cache_hits_total[10m])) by (service, cache_name)
                +
                sum(rate(cache_misses_total[10m])) by (service, cache_name)
              )
            ) < 0.8
          for: 15m
          labels:
            severity: warning
            component: cache
          annotations:
            summary: "Low cache hit rate"
            description: "{{ $labels.service }}/{{ $labels.cache_name }} has hit rate of {{ $value | humanizePercentage }}"
```

## 📝 Loki for Logs

### 1. Installation

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack \
  --namespace monitoring \
  -f observability/loki/values.yaml
```

### 2. Configuration

```yaml
# observability/loki/values.yaml
loki:
  config:
    auth_enabled: false
    
    server:
      http_listen_port: 3100
    
    ingester:
      lifecycler:
        ring:
          kvstore:
            store: inmemory
          replication_factor: 1
      chunk_idle_period: 15m
      chunk_retain_period: 30s
    
    schema_config:
      configs:
        - from: 2023-01-01
          store: boltdb-shipper
          object_store: s3
          schema: v11
          index:
            prefix: loki_index_
            period: 24h
    
    storage_config:
      boltdb_shipper:
        active_index_directory: /loki/boltdb-shipper-active
        cache_location: /loki/boltdb-shipper-cache
        shared_store: s3
      
      aws:
        s3: s3://us-east-1/loki-logs-bucket
        s3forcepathstyle: true
    
    compactor:
      working_directory: /loki/compactor
      shared_store: s3
      compaction_interval: 10m
    
    limits_config:
      enforce_metric_name: false
      reject_old_samples: true
      reject_old_samples_max_age: 168h  # 7 days
      retention_period: 30d

promtail:
  enabled: true
  config:
    clients:
      - url: http://loki:3100/loki/api/v1/push
    
    scrapeConfigs:
      # Pod logs
      - job_name: kubernetes-pods
        kubernetes_sd_configs:
          - role: pod
        pipeline_stages:
          - docker: {}
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_node_name]
            target_label: node_name
          - source_labels: [__meta_kubernetes_namespace]
            target_label: namespace
          - source_labels: [__meta_kubernetes_pod_name]
            target_label: pod
          - source_labels: [__meta_kubernetes_pod_container_name]
            target_label: container
```

### 3. Structured Logging (Go)

```go
// shared/libraries/go-commons/logger/logger.go
package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init(service, version, environment string) error {
    config := zap.NewProductionConfig()
    
    // JSON encoding para Loki
    config.Encoding = "json"
    
    // Time format ISO8601
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    // Level
    config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    
    var err error
    log, err = config.Build()
    if err != nil {
        return err
    }
    
    // Agregar campos globales
    log = log.With(
        zap.String("service", service),
        zap.String("version", version),
        zap.String("environment", environment),
    )
    
    return nil
}

func Info(msg string, fields ...zap.Field) {
    log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    log.Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
    log.Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
    log.Debug(msg, fields...)
}

// HTTPMiddleware para logging de requests
func HTTPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(rw, r)
        
        duration := time.Since(start)
        
        log.Info("HTTP request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
            zap.Int("status", rw.statusCode),
            zap.Duration("duration", duration),
            zap.String("remote_addr", r.RemoteAddr),
            zap.String("user_agent", r.UserAgent()),
            zap.String("request_id", r.Header.Get("X-Request-ID")),
        )
    })
}
```

### 4. LogQL Queries

```logql
# Todos los logs de un servicio
{app="user-service"}

# Logs con error
{app="user-service"} |= "error"

# Logs JSON parseados
{app="user-service"} | json | level="error"

# Rate de errores
rate({app="user-service"} |= "error" [5m])

# Top 10 endpoints más lentos
topk(10,
  sum by (path) (
    rate({app="user-service"} | json | __error__="" [5m])
  )
)

# Latency P99
quantile_over_time(0.99,
  {app="user-service"} 
  | json 
  | unwrap duration [5m]
) by (path)
```

Continúo con Tempo para distributed tracing...

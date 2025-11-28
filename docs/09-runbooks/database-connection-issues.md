# Runbook: Database Connection Issues

## 📋 Metadata

- **Severity**: P1 (Critical) - If production database
- **Alert**: DatabaseConnectionFailure / HighDatabaseLatency
- **Trigger**: Connection errors > 1% or query latency > 2s
- **On-Call**: Database Team (primary), Platform Team (secondary)

## 🎯 Symptoms

- Connection timeout errors
- "Too many connections" errors
- High query latency
- Application errors related to database
- Connection pool exhaustion

## 🔍 Quick Diagnosis

```bash
#!/bin/bash
# Quick diagnostic script

NAMESPACE="production"
DB_HOST="postgres-service.production.svc.cluster.local"
DB_PORT="5432"

echo "=== Database Connectivity Test ==="
kubectl run -n $NAMESPACE db-test --rm -i --tty --image=postgres:15 -- \
  psql -h $DB_HOST -U postgres -c "SELECT 1;" 2>&1 | head -5

echo -e "\n=== Active Connections ==="
kubectl run -n $NAMESPACE db-test --rm -i --tty --image=postgres:15 -- \
  psql -h $DB_HOST -U postgres -c "
    SELECT count(*) as total_connections,
           count(*) FILTER (WHERE state = 'active') as active,
           count(*) FILTER (WHERE state = 'idle') as idle,
           count(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction
    FROM pg_stat_activity;
  "

echo -e "\n=== Long Running Queries ==="
kubectl run -n $NAMESPACE db-test --rm -i --tty --image=postgres:15 -- \
  psql -h $DB_HOST -U postgres -c "
    SELECT pid, usename, state, query_start, now() - query_start as duration, query
    FROM pg_stat_activity
    WHERE state != 'idle' AND query_start < now() - interval '30 seconds'
    ORDER BY query_start
    LIMIT 10;
  "

echo -e "\n=== Database Locks ==="
kubectl run -n $NAMESPACE db-test --rm -i --tty --image=postgres:15 -- \
  psql -h $DB_HOST -U postgres -c "
    SELECT locktype, relation::regclass, mode, granted, pid
    FROM pg_locks
    WHERE NOT granted
    LIMIT 10;
  "
```

## 🔍 Investigation Steps

### 1. Verify Database Availability

```bash
# Check database pod/service
kubectl get pods -n production -l app=postgres
kubectl get svc -n production postgres-service

# Test connection from application pod
APP_POD=$(kubectl get pods -n production -l app=user-service -o name | head -1)
kubectl exec -n production $APP_POD -- nc -zv postgres-service 5432

# Check database logs
DB_POD=$(kubectl get pods -n production -l app=postgres -o name | head -1)
kubectl logs -n production $DB_POD --tail=100
```

### 2. Check Connection Count

```sql
-- Total connections by state
SELECT state, count(*) 
FROM pg_stat_activity 
GROUP BY state;

-- Connections by application
SELECT application_name, count(*) 
FROM pg_stat_activity 
GROUP BY application_name;

-- Connections by user
SELECT usename, count(*) 
FROM pg_stat_activity 
GROUP BY usename;

-- Check max connections
SHOW max_connections;

-- Current connection usage percentage
SELECT count(*)*100/(SELECT setting::int FROM pg_settings WHERE name='max_connections') as percentage
FROM pg_stat_activity;
```

### 3. Identify Connection Leaks

```sql
-- Idle in transaction (potential leak)
SELECT pid, usename, application_name, state, query_start, now() - query_start as duration
FROM pg_stat_activity
WHERE state = 'idle in transaction'
  AND query_start < now() - interval '5 minutes'
ORDER BY query_start;

-- Long-running connections
SELECT pid, usename, application_name, backend_start, now() - backend_start as connection_age
FROM pg_stat_activity
WHERE backend_start < now() - interval '1 hour'
ORDER BY backend_start;
```

### 4. Check Query Performance

```sql
-- Slow queries from pg_stat_statements
SELECT calls, 
       round(total_exec_time::numeric, 2) as total_time,
       round(mean_exec_time::numeric, 2) as mean_time,
       round((100 * total_exec_time / sum(total_exec_time) OVER ())::numeric, 2) as percentage,
       query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;

-- Currently executing queries
SELECT pid, usename, state, now() - query_start as duration, query
FROM pg_stat_activity
WHERE state = 'active'
  AND query NOT LIKE '%pg_stat_activity%'
ORDER BY query_start;

-- Blocked queries
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query AS blocked_statement,
       blocking_activity.query AS blocking_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
  AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
  AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
  AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
  AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
  AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
  AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
  AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
  AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
  AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
  AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
```

### 5. Check Resource Usage

```bash
# Database resource usage
kubectl top pod -n production $DB_POD

# Check database size
kubectl exec -n production $DB_POD -- psql -U postgres -c "
  SELECT pg_database.datname,
         pg_size_pretty(pg_database_size(pg_database.datname)) AS size
  FROM pg_database
  ORDER BY pg_database_size(pg_database.datname) DESC;
"

# Check table sizes
kubectl exec -n production $DB_POD -- psql -U postgres -d app -c "
  SELECT schemaname,
         tablename,
         pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
  LIMIT 10;
"
```

## 🚨 Immediate Actions

### 1. Connection Pool Exhaustion

```bash
# Emergency: Increase max connections temporarily
kubectl exec -n production $DB_POD -- psql -U postgres -c "
  ALTER SYSTEM SET max_connections = 200;
  SELECT pg_reload_conf();
"

# Or restart database (if ALTER SYSTEM doesn't work)
kubectl rollout restart statefulset postgres -n production

# Scale application down to reduce connection pressure
kubectl scale deployment user-service -n production --replicas=2
```

### 2. Kill Problematic Connections

```sql
-- Kill idle in transaction connections older than 5 minutes
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle in transaction'
  AND query_start < now() - interval '5 minutes';

-- Kill specific long-running query
SELECT pg_cancel_backend(12345);  -- Use actual PID

-- Kill all connections from specific app (emergency)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE application_name = 'user-service'
  AND pid != pg_backend_pid();
```

### 3. Connection Pooler (PgBouncer)

```yaml
# Deploy PgBouncer as connection pooler
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
  namespace: production
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pgbouncer
  template:
    metadata:
      labels:
        app: pgbouncer
    spec:
      containers:
      - name: pgbouncer
        image: pgbouncer/pgbouncer:1.21
        ports:
        - containerPort: 5432
        env:
        - name: DATABASES_HOST
          value: postgres-service
        - name: DATABASES_PORT
          value: "5432"
        - name: DATABASES_USER
          value: postgres
        - name: DATABASES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secrets
              key: password
        - name: POOL_MODE
          value: transaction
        - name: MAX_CLIENT_CONN
          value: "1000"
        - name: DEFAULT_POOL_SIZE
          value: "25"
        - name: RESERVE_POOL_SIZE
          value: "5"
        - name: RESERVE_POOL_TIMEOUT
          value: "3"
---
apiVersion: v1
kind: Service
metadata:
  name: pgbouncer
  namespace: production
spec:
  selector:
    app: pgbouncer
  ports:
  - port: 5432
    targetPort: 5432
```

```bash
kubectl apply -f pgbouncer.yaml

# Update application to use pgbouncer
kubectl set env deployment user-service -n production \
  DATABASE_URL="postgresql://user:pass@pgbouncer:5432/app"
```

## 🔧 Application-Level Fixes

### Go Application Connection Pool Tuning

```go
// applications/microservices/user-service/internal/database/postgres.go
package database

import (
    "database/sql"
    "time"
    _ "github.com/lib/pq"
)

func NewPostgresDB(connectionString string) (*sql.DB, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }

    // Connection pool settings
    db.SetMaxOpenConns(25)                  // Maximum open connections
    db.SetMaxIdleConns(10)                  // Maximum idle connections
    db.SetConnMaxLifetime(5 * time.Minute)  // Maximum connection lifetime
    db.SetConnMaxIdleTime(10 * time.Minute) // Maximum idle time

    // Verify connection
    if err := db.Ping(); err != nil {
        return nil, err
    }

    return db, nil
}
```

### Connection Context with Timeout

```go
// Always use context with timeout for database operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := db.QueryContext(ctx, "SELECT * FROM users WHERE id = $1", userID)
if err != nil {
    return fmt.Errorf("query failed: %w", err)
}
defer rows.Close()
```

### Proper Connection Cleanup

```go
// BAD - Connection leak
rows, _ := db.Query("SELECT * FROM users")
// Processing rows without calling rows.Close()

// GOOD - Proper cleanup
rows, err := db.Query("SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close()  // Always close rows

for rows.Next() {
    // Process row
}
return rows.Err()  // Check for errors during iteration
```

## 📊 Monitoring & Alerting

### Prometheus Metrics

```yaml
# ServiceMonitor for postgres-exporter
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: postgres-metrics
  namespace: production
spec:
  selector:
    matchLabels:
      app: postgres-exporter
  endpoints:
  - port: metrics
    interval: 30s
```

### Key Metrics to Monitor

```promql
# Connection usage percentage
(pg_stat_database_numbackends{datname="app"} / pg_settings_max_connections) * 100

# Connection pool wait time
pg_stat_activity_max_tx_duration{state="active"}

# Idle in transaction
pg_stat_activity_count{state="idle in transaction"}

# Slow queries
rate(pg_stat_statements_total_time[5m]) / rate(pg_stat_statements_calls[5m])

# Deadlocks
rate(pg_stat_database_deadlocks[5m])
```

### Alerts

```yaml
groups:
- name: database
  rules:
  - alert: HighDatabaseConnections
    expr: |
      (pg_stat_database_numbackends / pg_settings_max_connections) * 100 > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Database connection usage high"
      description: "{{ $value }}% of max connections in use"

  - alert: DatabaseConnectionExhaustion
    expr: |
      (pg_stat_database_numbackends / pg_settings_max_connections) * 100 > 95
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Database connection pool nearly exhausted"
      description: "{{ $value }}% of max connections in use"

  - alert: IdleInTransactionConnections
    expr: |
      pg_stat_activity_count{state="idle in transaction"} > 10
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Many idle in transaction connections"
      description: "{{ $value }} connections idle in transaction"

  - alert: SlowQueries
    expr: |
      rate(pg_stat_statements_total_time[5m]) / rate(pg_stat_statements_calls[5m]) > 2000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Slow database queries detected"
      description: "Average query time: {{ $value }}ms"
```

## 🛠️ Preventive Measures

### 1. Application Connection Management

```yaml
# ConfigMap for application database settings
apiVersion: v1
kind: ConfigMap
metadata:
  name: db-connection-config
  namespace: production
data:
  DB_MAX_OPEN_CONNS: "25"
  DB_MAX_IDLE_CONNS: "10"
  DB_CONN_MAX_LIFETIME: "5m"
  DB_CONN_MAX_IDLE_TIME: "10m"
  DB_TIMEOUT: "5s"
```

### 2. Database Configuration

```sql
-- Set statement timeout to prevent runaway queries
ALTER DATABASE app SET statement_timeout = '30s';

-- Set idle in transaction timeout
ALTER DATABASE app SET idle_in_transaction_session_timeout = '10min';

-- Enable auto_explain for slow queries
ALTER DATABASE app SET auto_explain.log_min_duration = '1000';  -- 1 second
ALTER DATABASE app SET auto_explain.log_analyze = 'true';
```

### 3. Regular Maintenance

```bash
#!/bin/bash
# Database maintenance script

kubectl exec -n production $DB_POD -- psql -U postgres -d app <<EOF
-- Vacuum analyze to maintain statistics
VACUUM ANALYZE;

-- Reindex to maintain index health
REINDEX DATABASE app;

-- Check for bloat
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
EOF
```

## 📝 Post-Incident Actions

1. **Analyze connection patterns**: Review application logs and metrics
2. **Tune connection pools**: Adjust based on actual usage
3. **Implement connection pooler**: Deploy PgBouncer if not already in place
4. **Add monitoring**: Ensure all key metrics are tracked
5. **Update runbooks**: Document any new findings
6. **Code review**: Check for connection leaks in application code

## 🔗 Related Resources

- **PostgreSQL Docs**: https://www.postgresql.org/docs/current/runtime-config-connection.html
- **PgBouncer Docs**: https://www.pgbouncer.org/
- **Grafana Dashboard**: https://grafana.acme.com/d/postgres-overview
- **Related Runbooks**:
  - High Error Rate Incident
  - High Latency
  - Pod Crash Loop

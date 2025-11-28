# Runbook: Pod Crash Loop

## 📋 Metadata

- **Severity**: P2 (High)
- **Alert**: PodCrashLooping
- **Trigger**: Pod restart count > 5 in 10 minutes
- **On-Call**: Platform Team

## 🎯 Symptoms

- Pods restarting repeatedly
- Service degradation
- Pod status: CrashLoopBackOff or Error
- Application unavailable or intermittent

## 🔍 Quick Diagnosis

```bash
#!/bin/bash
# Quick diagnostic script

NAMESPACE="production"
SERVICE="user-service"

echo "=== Pod Status ==="
kubectl get pods -n $NAMESPACE -l app=$SERVICE

echo -e "\n=== Recent Events ==="
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | grep $SERVICE | tail -20

echo -e "\n=== Pod Restarts ==="
kubectl get pods -n $NAMESPACE -l app=$SERVICE -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[0].restartCount}{"\n"}{end}'

echo -e "\n=== Last Logs ==="
POD=$(kubectl get pods -n $NAMESPACE -l app=$SERVICE -o name | head -1)
kubectl logs -n $NAMESPACE $POD --tail=50

echo -e "\n=== Previous Instance Logs ==="
kubectl logs -n $NAMESPACE $POD --previous --tail=50
```

## 🔍 Investigation Steps

### 1. Check Pod Status

```bash
# Get pod details
kubectl get pods -n production -l app=user-service -o wide

# Describe problematic pod
POD=$(kubectl get pods -n production -l app=user-service -o name | head -1)
kubectl describe pod -n production $POD

# Check exit code
kubectl get pod -n production $POD -o jsonpath='{.status.containerStatuses[0].lastState.terminated.exitCode}'
```

**Common Exit Codes:**
- `0`: Normal exit (should not crash)
- `1`: Application error
- `2`: Misconfiguration
- `137`: SIGKILL (OOMKilled)
- `139`: SIGSEGV (Segmentation fault)
- `143`: SIGTERM (Graceful termination)

### 2. Check Logs

```bash
# Current logs
kubectl logs -n production $POD --tail=100

# Previous container logs (if restarted)
kubectl logs -n production $POD --previous --tail=100

# All containers in pod
kubectl logs -n production $POD --all-containers=true --tail=100

# Follow logs in real-time
kubectl logs -n production $POD -f
```

### 3. Check Resource Usage

```bash
# Current resource usage
kubectl top pod -n production $POD

# Check resource limits
kubectl get pod -n production $POD -o jsonpath='{.spec.containers[0].resources}'

# Memory usage over time (Prometheus query)
container_memory_working_set_bytes{pod=~"user-service.*", namespace="production"}
```

### 4. Check Configuration

```bash
# Check ConfigMap
kubectl get configmap -n production user-service-config -o yaml

# Check Secrets
kubectl get secret -n production user-service-secrets -o jsonpath='{.data}' | jq 'keys'

# Check environment variables
kubectl exec -n production $POD -- env | sort
```

### 5. Check Dependencies

```bash
# Network connectivity
kubectl exec -n production $POD -- nc -zv postgres-service 5432
kubectl exec -n production $POD -- nc -zv redis-service 6379

# DNS resolution
kubectl exec -n production $POD -- nslookup postgres-service
kubectl exec -n production $POD -- nslookup external-api.example.com

# External API health
kubectl exec -n production $POD -- curl -I https://external-api.example.com/health
```

## 🚨 Common Causes & Solutions

### Cause 1: Out of Memory (Exit Code 137)

**Symptoms:**
```bash
kubectl describe pod $POD | grep -i "OOMKilled"
# Output: Reason: OOMKilled
```

**Solution:**
```bash
# Temporarily increase memory
kubectl set resources deployment user-service -n production \
  --limits=memory=2Gi --requests=memory=1Gi

# Permanent fix: Update GitOps repo
cat <<EOF > gitops/environments/overlays/production/user-service/patches/resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    spec:
      containers:
      - name: user-service
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
EOF

git add . && git commit -m "fix: increase user-service memory limits"
git push origin main
```

### Cause 2: Failed Health Checks

**Symptoms:**
```bash
kubectl describe pod $POD | grep -i "liveness\|readiness"
# Output: Liveness probe failed: HTTP probe failed with statuscode: 500
```

**Solution:**
```bash
# Check health endpoint directly
kubectl exec -n production $POD -- curl -v http://localhost:8080/health

# Temporarily disable probes to allow startup
kubectl patch deployment user-service -n production -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "user-service",
          "livenessProbe": null,
          "readinessProbe": null
        }]
      }
    }
  }
}'

# Investigate actual issue, then restore probes
```

### Cause 3: Missing or Invalid Configuration

**Symptoms:**
```bash
kubectl logs -n production $POD
# Output: Error: DATABASE_URL environment variable not set
```

**Solution:**
```bash
# Check if ConfigMap/Secret exists
kubectl get configmap -n production user-service-config
kubectl get secret -n production user-service-secrets

# Verify values
kubectl get secret -n production user-service-secrets -o json | \
  jq -r '.data.DATABASE_URL' | base64 -d

# Update secret if needed
kubectl create secret generic user-service-secrets -n production \
  --from-literal=DATABASE_URL="postgresql://user:pass@postgres:5432/db" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart deployment
kubectl rollout restart deployment user-service -n production
```

### Cause 4: Dependency Not Available

**Symptoms:**
```bash
kubectl logs -n production $POD
# Output: Failed to connect to postgres-service:5432: connection refused
```

**Solution:**
```bash
# Check if dependency is running
kubectl get pods -n production -l app=postgres

# Check service
kubectl get svc -n production postgres-service

# Test connectivity
kubectl run -n production test-connection --rm -i --tty \
  --image=postgres:15 -- \
  psql -h postgres-service -U postgres -d app

# Check network policies
kubectl get networkpolicies -n production
```

### Cause 5: Image Pull Errors

**Symptoms:**
```bash
kubectl describe pod $POD | grep -i "imagepull"
# Output: Failed to pull image "harbor.acme.com/platform/user-service:v1.2.3"
```

**Solution:**
```bash
# Check image exists
curl -u admin:password https://harbor.acme.com/v2/platform/user-service/tags/list

# Check image pull secret
kubectl get secret -n production harbor-registry-secret

# Recreate secret if needed
kubectl create secret docker-registry harbor-registry-secret -n production \
  --docker-server=harbor.acme.com \
  --docker-username=robot-account \
  --docker-password=$ROBOT_PASSWORD \
  --docker-email=platform@acme.com

# Verify pod has correct imagePullSecrets
kubectl get pod $POD -o jsonpath='{.spec.imagePullSecrets}'
```

### Cause 6: Application Bug

**Symptoms:**
```bash
kubectl logs -n production $POD --previous
# Output: panic: runtime error: invalid memory address or nil pointer dereference
```

**Solution:**
```bash
# Collect crash dump
kubectl cp -n production $POD:/tmp/core.dump ./core.dump

# For Go applications, get stack trace
kubectl logs -n production $POD --previous | grep "goroutine"

# Rollback to previous working version
argocd app history user-service-prod
LAST_WORKING=$(argocd app history user-service-prod --output json | \
  jq -r '.[] | select(.deployedAt != null) | .revision' | sed -n '2p')

argocd app rollback user-service-prod $LAST_WORKING

# Create hotfix branch
git checkout -b hotfix/crash-on-nil-pointer
# Fix code, commit, push, create PR
```

## 🔧 Advanced Troubleshooting

### Debug Container (Kubernetes 1.23+)

```bash
# Start ephemeral debug container
kubectl debug -n production $POD -it --image=busybox --target=user-service

# Inside debug container:
ps aux                    # Check processes
netstat -tuln             # Check listening ports
cat /proc/1/environ       # Check environment variables
ls -la /app               # Check application files
```

### Memory Profiling (Go Application)

```bash
# Get heap profile
kubectl exec -n production $POD -- curl http://localhost:8080/debug/pprof/heap > heap.pprof

# Get goroutine profile
kubectl exec -n production $POD -- curl http://localhost:8080/debug/pprof/goroutine > goroutine.pprof

# Analyze locally
go tool pprof -http=:8081 heap.pprof
```

### Strace Running Process

```bash
# Get PID
kubectl exec -n production $POD -- ps aux | grep user-service

# Attach strace
kubectl exec -n production $POD -- strace -p <PID> -f -t -e trace=all
```

### Check Kernel Logs

```bash
# Get node name
NODE=$(kubectl get pod -n production $POD -o jsonpath='{.spec.nodeName}')

# SSH to node (if accessible)
ssh $NODE

# Check kernel logs for OOM killer
sudo dmesg | grep -i "oom"
sudo journalctl -k | grep -i "killed process"
```

## 🔄 Restart Strategies

### 1. Rolling Restart

```bash
# Graceful rolling restart
kubectl rollout restart deployment user-service -n production

# Watch progress
kubectl rollout status deployment user-service -n production
```

### 2. Delete Single Pod

```bash
# Delete one pod at a time
kubectl delete pod $POD -n production

# Wait for replacement
kubectl wait --for=condition=Ready pod -l app=user-service -n production --timeout=300s
```

### 3. Recreate Deployment

```bash
# WARNING: Causes downtime
kubectl patch deployment user-service -n production -p '
{
  "spec": {
    "strategy": {
      "type": "Recreate"
    }
  }
}'

kubectl rollout restart deployment user-service -n production
```

## 📊 Monitoring Queries

### Prometheus Queries

```promql
# Pod restart rate
rate(kube_pod_container_status_restarts_total{namespace="production",pod=~"user-service.*"}[5m])

# OOMKills
sum(increase(kube_pod_container_status_terminated_reason{reason="OOMKilled"}[1h])) by (pod)

# Memory usage
container_memory_working_set_bytes{namespace="production",pod=~"user-service.*"}
  / 
container_spec_memory_limit_bytes{namespace="production",pod=~"user-service.*"}

# CPU throttling
rate(container_cpu_cfs_throttled_seconds_total{namespace="production",pod=~"user-service.*"}[5m])
```

### Loki Queries (LogQL)

```logql
# Errors before crash
{namespace="production", app="user-service"} 
  |= "ERROR" 
  | json 
  | line_format "{{.timestamp}} {{.level}} {{.message}}"

# Panic messages
{namespace="production", app="user-service"} 
  |= "panic" 
  | line_format "{{.message}}"
```

## 📝 Post-Resolution Actions

### 1. Document Root Cause

```bash
# Update incident log
cat <<EOF >> incidents/2024-01-15-pod-crash.md
# Incident: Pod Crash Loop - user-service

**Date**: 2024-01-15
**Duration**: 30 minutes
**Root Cause**: Memory leak in user session handling
**Resolution**: Increased memory limits, deployed fix in v1.2.4

**Prevention**:
- Added memory leak detection to CI
- Implemented session cleanup job
- Increased staging load tests
EOF
```

### 2. Update Alerts

```yaml
# Add alert for high restart rate
- alert: HighPodRestartRate
  expr: |
    rate(kube_pod_container_status_restarts_total{namespace="production"}[15m]) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Pod {{ $labels.pod }} restarting frequently"
    description: "{{ $labels.pod }} has restarted {{ $value }} times/sec"
```

### 3. Improve Health Checks

```yaml
# Add startup probe for slow-starting apps
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    spec:
      containers:
      - name: user-service
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 0
          periodSeconds: 10
          failureThreshold: 30  # 5 minutes to start
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          failureThreshold: 3
```

## 🔗 Related Resources

- **Kubernetes Debugging Guide**: https://kubernetes.io/docs/tasks/debug/
- **Grafana Dashboard**: https://grafana.acme.com/d/pod-health
- **Related Runbooks**:
  - High Error Rate Incident
  - High Latency
  - Database Connection Issues

## 📞 Escalation

**Platform Team**: @platform-team
**Application Team**: @app-team
**Kubernetes Experts**: @k8s-team

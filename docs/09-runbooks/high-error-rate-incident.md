# Runbook: High Error Rate Incident

## 📋 Metadata

- **Severity**: P1 (Critical)
- **Service**: user-service
- **Alert**: HighErrorRateAfterDeploy
- **Trigger**: Error rate > 5% for 5+ minutes
- **On-Call**: Platform Team

## 🎯 Symptoms

- High 5xx error rate
- Increased latency
- Failed health checks
- Customer complaints

## 🔍 Investigation Steps

### 1. Verify the Alert

```bash
# Check current error rate
kubectl top pods -n production -l app=user-service

# Check Grafana dashboard
open https://grafana.acme.com/d/microservices-overview

# Query Prometheus directly
curl -G https://prometheus.acme.com/api/v1/query \
  --data-urlencode 'query=rate(http_requests_total{service="user-service",status=~"5.."}[5m])'
```

### 2. Identify Recent Changes

```bash
# Check recent deployments
argocd app history user-service-prod

# Check recent commits
git log --oneline -10 --all

# Check recent infrastructure changes
cd infrastructure/terraform/environments/production
terraform show | grep user-service
```

### 3. Check Application Logs

```bash
# Recent errors
kubectl logs -n production -l app=user-service --tail=100 | grep ERROR

# Using Loki (LogQL)
# In Grafana → Explore → Loki
{app="user-service", environment="production"} |= "ERROR" | json

# Specific pod logs
POD=$(kubectl get pods -n production -l app=user-service -o name | head -1)
kubectl logs -n production $POD --tail=500
```

### 4. Check Dependencies

```bash
# Check database connectivity
kubectl exec -n production $POD -- nc -zv postgres-service 5432

# Check Redis
kubectl exec -n production $POD -- nc -zv redis-service 6379

# Check external APIs
kubectl exec -n production $POD -- curl -I https://external-api.example.com/health
```

### 5. Check Resource Usage

```bash
# CPU and Memory
kubectl top pods -n production -l app=user-service

# Describe pod for events
kubectl describe pod -n production $POD

# Check HPA status
kubectl get hpa -n production user-service
```

## 🚨 Immediate Actions

### Decision Tree

```
High Error Rate Detected
        │
        ├─→ Recent Deployment? ──YES──→ ROLLBACK IMMEDIATELY
        │                                      │
        │                                      └─→ Monitor for 5 min
        │                                           │
        │                                           ├─→ Fixed? ──YES──→ Incident closed
        │                                           └─→ Not fixed? ──→ Continue investigation
        │
        └─→ NO ──→ Infrastructure Issue?
                   │
                   ├─→ YES ──→ Scale up resources
                   │            │
                   │            └─→ Fixed? ──YES──→ Root cause analysis
                   │                 │
                   │                 └─→ NO ──→ Continue investigation
                   │
                   └─→ NO ──→ External dependency?
                              │
                              ├─→ YES ──→ Enable circuit breaker
                              │            Notify dependency owner
                              │
                              └─→ NO ──→ Deep investigation needed
```

### 1. Rollback Deployment

```bash
#!/bin/bash
# Immediate rollback

SERVICE="user-service"
ENVIRONMENT="production"

# Get previous working version
PREVIOUS_VERSION=$(argocd app history $SERVICE-$ENVIRONMENT --output json | \
  jq -r '.[] | select(.deployedAt != null) | .revision' | \
  sed -n '2p')

echo "Rolling back to $PREVIOUS_VERSION"

# Execute rollback
argocd app rollback $SERVICE-$ENVIRONMENT $PREVIOUS_VERSION

# Wait for rollback
argocd app wait $SERVICE-$ENVIRONMENT --sync --timeout 300

# Verify
kubectl rollout status deployment/$SERVICE -n $ENVIRONMENT
```

### 2. Scale Up Resources (if not deployment issue)

```bash
# Scale up pods immediately
kubectl scale deployment user-service -n production --replicas=10

# Increase resource limits temporarily
kubectl set resources deployment user-service -n production \
  --limits=cpu=2000m,memory=2Gi

# Force HPA to scale faster
kubectl patch hpa user-service -n production -p '
{
  "spec": {
    "maxReplicas": 20
  }
}'
```

### 3. Enable Circuit Breaker (if external dependency)

```yaml
# Apply Istio DestinationRule with circuit breaker
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: external-api-circuit-breaker
spec:
  host: external-api.example.com
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 2
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 60s
      maxEjectionPercent: 50
```

```bash
kubectl apply -f circuit-breaker.yaml
```

## 🔧 Detailed Troubleshooting

### Database Issues

```bash
# Check database connections
kubectl exec -n production $POD -- sh -c '
  echo "SELECT count(*) FROM pg_stat_activity;" | \
  psql $DATABASE_URL
'

# Check slow queries
kubectl exec -n production $POD -- sh -c '
  echo "SELECT query, calls, total_time FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;" | \
  psql $DATABASE_URL
'

# Check database locks
kubectl exec -n production $POD -- sh -c '
  echo "SELECT * FROM pg_locks WHERE NOT granted;" | \
  psql $DATABASE_URL
'
```

### Memory Leak Detection

```bash
# Get heap dump (if Go application)
kubectl exec -n production $POD -- curl http://localhost:8080/debug/pprof/heap > heap.pprof

# Analyze with pprof
go tool pprof heap.pprof
```

### Network Issues

```bash
# Check network policies
kubectl get networkpolicies -n production

# Test connectivity
kubectl run -n production test-pod --rm -i --tty \
  --image=nicolaka/netshoot -- \
  /bin/bash

# Inside test-pod:
curl -v http://user-service/health
traceroute user-service
```

## 📝 Communication

### Incident Notification Template

```markdown
**Subject**: [P1] High Error Rate - user-service Production

**Status**: INVESTIGATING / MITIGATING / RESOLVED

**Impact**:
- Error rate: 8.5% (threshold: 5%)
- Affected users: ~1000 users/min
- Affected endpoints: /api/v1/users/*

**Timeline**:
- 14:32 UTC: Alert triggered
- 14:33 UTC: On-call acknowledged
- 14:35 UTC: Rollback initiated
- 14:37 UTC: Service recovered

**Actions Taken**:
1. Rolled back to v1.2.2
2. Error rate now at 0.2%
3. Monitoring for 15 minutes

**Root Cause** (if known):
New deployment introduced database connection leak

**Next Steps**:
1. Continue monitoring
2. Root cause analysis
3. Fix and redeploy

**Contact**:
- Incident Commander: @john-doe
- Slack channel: #incident-2024-001
```

### Slack Command

```bash
# Post to Slack
curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -H 'Content-Type: application/json' \
  -d '{
    "text": "🚨 P1 Incident - High Error Rate",
    "attachments": [{
      "color": "danger",
      "fields": [
        {"title": "Service", "value": "user-service", "short": true},
        {"title": "Error Rate", "value": "8.5%", "short": true},
        {"title": "Status", "value": "Investigating", "short": true}
      ]
    }]
  }'
```

## 📊 Post-Incident Actions

### 1. Create Post-Mortem

```markdown
# Post-Mortem: 2024-01-15 High Error Rate Incident

## Summary
High error rate (8.5%) detected in user-service production due to database 
connection leak in new deployment.

## Timeline
- **14:30 UTC**: Deployment v1.2.3 completed
- **14:32 UTC**: Alert triggered (error rate > 5%)
- **14:33 UTC**: On-call acknowledged
- **14:35 UTC**: Rollback initiated
- **14:37 UTC**: Service recovered
- **14:45 UTC**: Incident closed

**Total Duration**: 15 minutes
**User Impact**: ~15,000 failed requests

## Root Cause
Database connection pool not properly closed in new code path, leading to 
connection exhaustion and cascading failures.

## What Went Well
- Alert triggered quickly (2 minutes after issue)
- Rollback procedure worked as expected
- Communication was clear and timely

## What Went Wrong
- Connection leak not caught in staging
- No automated rollback on error threshold
- Limited database connection monitoring

## Action Items
1. [ ] Add connection pool monitoring (Owner: @dba, Due: 2024-01-20)
2. [ ] Implement automated rollback on error rate (Owner: @platform, Due: 2024-01-25)
3. [ ] Add database connection tests to CI (Owner: @dev-team, Due: 2024-01-22)
4. [ ] Update staging to match production load (Owner: @platform, Due: 2024-02-01)
5. [ ] Add connection leak detection to code review (Owner: @tech-lead, Due: 2024-01-18)
```

### 2. Update Monitoring

```yaml
# Add new alert for connection pool
- alert: HighDatabaseConnections
  expr: |
    sum(database_connections_active) by (service) > 80
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High database connections"
    description: "{{ $labels.service }} has {{ $value }} active connections"
```

### 3. Update Runbook

```bash
# Add lessons learned to runbook
echo "
## Lessons Learned from 2024-01-15 Incident
- Check database connection pool metrics early
- Consider automated rollback for error rate > 7%
- Always test with production-like load in staging
" >> docs/09-runbooks/high-error-rate.md
```

## 🔗 Related Resources

- **Grafana Dashboard**: https://grafana.acme.com/d/microservices-overview
- **ArgoCD**: https://argocd.acme.com/applications/user-service-prod
- **Slack Channel**: #incidents
- **PagerDuty**: https://acme.pagerduty.com/incidents
- **Related Runbooks**:
  - Database Connection Issues
  - Pod Crashes
  - High Latency

## 📞 Escalation

**Platform Team**: @platform-team
**Database Team**: @dba-team
**Security Team**: @security-team
**VP Engineering**: @vp-eng (P0 incidents only)

# Disaster Recovery Procedures

## 📋 Overview

This runbook covers disaster recovery procedures for various catastrophic scenarios including cluster failure, data center outages, and data corruption.

## 🎯 Recovery Time Objectives (RTO) & Recovery Point Objectives (RPO)

| Service Tier | RTO | RPO | Backup Frequency |
|--------------|-----|-----|------------------|
| Critical (user-service, order-service) | 15 minutes | 5 minutes | Continuous replication + 15-min snapshots |
| High (notification-service) | 1 hour | 15 minutes | Hourly snapshots |
| Medium (analytics) | 4 hours | 1 hour | Daily backups |
| Low (logs, metrics) | 24 hours | 24 hours | Weekly backups |

## 🚨 Disaster Scenarios

### Scenario 1: Complete EKS Cluster Failure

**Symptoms:**
- Entire cluster unreachable
- All nodes down
- AWS console shows cluster in failed state

**Recovery Steps:**

```bash
#!/bin/bash
# Complete cluster recovery

# 1. Create new cluster from Terraform
cd infrastructure/terraform/environments/production

# Update backend to use new state
cat > backend.tf <<EOF
terraform {
  backend "s3" {
    bucket = "acme-terraform-state-dr"
    key    = "production/eks-recovery.tfstate"
    region = "us-east-1"
  }
}
EOF

# Initialize and apply
terraform init
terraform apply -target=module.eks -auto-approve

# 2. Update kubeconfig
aws eks update-kubeconfig --name acme-prod-eks-cluster --region us-east-1

# 3. Install core platform components
cd ../../../../gitops/bootstrap

# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=600s \
  deployment/argocd-server -n argocd

# Install bootstrap applications
kubectl apply -f bootstrap/app-of-apps.yaml

# 4. Restore data from backups (see Database Recovery section)

# 5. Verify all applications are synced
argocd app list
argocd app sync --all
```

**Estimated Recovery Time:** 30-45 minutes

### Scenario 2: Database Disaster Recovery

**Symptoms:**
- Database unavailable
- Data corruption detected
- Entire database deleted

**Recovery Steps:**

#### A. Restore from AWS RDS Automated Backup

```bash
#!/bin/bash
# RDS Point-in-Time Recovery

# 1. Identify latest backup
aws rds describe-db-snapshots \
  --db-instance-identifier acme-prod-postgres \
  --query 'DBSnapshots[0].DBSnapshotIdentifier' \
  --output text

# 2. Restore to new instance
RESTORE_TIME=$(date -u -d '5 minutes ago' +"%Y-%m-%dT%H:%M:%SZ")

aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier acme-prod-postgres \
  --target-db-instance-identifier acme-prod-postgres-recovery \
  --restore-time "$RESTORE_TIME" \
  --vpc-security-group-ids sg-xxxxxxxxx \
  --db-subnet-group-name acme-prod-db-subnet

# 3. Wait for instance to be available
aws rds wait db-instance-available \
  --db-instance-identifier acme-prod-postgres-recovery

# 4. Get new endpoint
NEW_ENDPOINT=$(aws rds describe-db-instances \
  --db-instance-identifier acme-prod-postgres-recovery \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)

# 5. Update Crossplane database claim
kubectl patch database.acme.com user-db -n production -p "
{
  \"spec\": {
    \"parameters\": {
      \"endpoint\": \"$NEW_ENDPOINT\"
    }
  }
}" --type=merge

# 6. Restart applications to pick up new connection
kubectl rollout restart deployment -n production -l db=user-db

# 7. Verify data integrity
kubectl run -n production db-verify --rm -i --tty --image=postgres:15 -- \
  psql -h $NEW_ENDPOINT -U postgres -c "SELECT count(*) FROM users;"

# 8. Once verified, promote recovery instance to primary
aws rds modify-db-instance \
  --db-instance-identifier acme-prod-postgres-recovery \
  --new-db-instance-identifier acme-prod-postgres \
  --apply-immediately
```

#### B. Restore from Velero Backup

```bash
#!/bin/bash
# Velero-based recovery

# 1. List available backups
velero backup get

# 2. Restore specific backup
velero restore create --from-backup production-daily-20240115

# 3. Monitor restore progress
velero restore describe production-daily-20240115-restore
velero restore logs production-daily-20240115-restore

# 4. Verify PVCs are restored
kubectl get pvc -n production
```

**Estimated Recovery Time:** 15-30 minutes for RDS, 10-20 minutes for Velero

### Scenario 3: Complete AWS Region Failure

**Symptoms:**
- All AWS services in region unavailable
- Multi-AZ redundancy not helping
- AWS status page confirms regional outage

**Recovery Steps:**

```bash
#!/bin/bash
# Multi-region failover

# 1. Switch DNS to secondary region
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.acme.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z0987654321XYZ",
          "DNSName": "acme-dr-alb-us-west-2.elb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'

# 2. Verify secondary region cluster
export AWS_REGION=us-west-2
aws eks update-kubeconfig --name acme-dr-eks-cluster --region us-west-2

# 3. Check application status
kubectl get pods -A
argocd app list

# 4. Verify database replication status
kubectl exec -n production postgres-0 -- psql -U postgres -c "
  SELECT * FROM pg_stat_replication;
"

# 5. Promote read replica to primary (if needed)
aws rds promote-read-replica \
  --db-instance-identifier acme-dr-postgres-us-west-2

# 6. Update application database endpoints
kubectl patch database.acme.com user-db -n production -p "
{
  \"spec\": {
    \"parameters\": {
      \"endpoint\": \"acme-dr-postgres.us-west-2.rds.amazonaws.com\"
    }
  }
}" --type=merge

# 7. Restart applications
kubectl rollout restart deployment -n production

# 8. Monitor traffic and errors
watch kubectl top nodes
watch kubectl get pods -n production
```

**Estimated Recovery Time:** 10-15 minutes (DNS propagation: 5-60 minutes)

### Scenario 4: Data Corruption or Ransomware

**Symptoms:**
- Application reports data inconsistencies
- Unauthorized data modifications detected
- Files encrypted by ransomware

**Recovery Steps:**

```bash
#!/bin/bash
# Isolated recovery environment

# 1. IMMEDIATELY isolate affected systems
kubectl patch networkpolicy default-deny -n production -p '
{
  "spec": {
    "policyTypes": ["Ingress", "Egress"],
    "ingress": [],
    "egress": []
  }
}'

# 2. Create forensic snapshots
aws ec2 create-snapshot \
  --volume-id vol-xxxxxxxxx \
  --description "Forensic snapshot - incident 2024-01-15" \
  --tag-specifications 'ResourceType=snapshot,Tags=[{Key=Purpose,Value=Forensics}]'

# 3. Create isolated recovery environment
kubectl create namespace recovery-isolated

kubectl apply -f - <<EOF
apiVersion: v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: recovery-isolated
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
EOF

# 4. Restore from clean backup (before corruption)
velero restore create recovery-isolated \
  --from-backup production-daily-20240114 \
  --namespace-mappings production:recovery-isolated

# 5. Verify restored data integrity
kubectl run -n recovery-isolated data-verify --rm -i --tty \
  --image=postgres:15 -- bash

# Inside container:
psql -h postgres-service -U postgres -d app <<SQL
-- Check record counts
SELECT 'users' as table_name, count(*) FROM users
UNION ALL
SELECT 'orders', count(*) FROM orders;

-- Verify data integrity
SELECT * FROM users WHERE email IS NULL OR email = '';
SELECT * FROM orders WHERE total < 0;
SQL

# 6. Once verified clean, restore to production
kubectl delete namespace production-infected
kubectl create namespace production
velero restore create production-recovery \
  --from-backup production-daily-20240114 \
  --namespace-mappings recovery-isolated:production

# 7. Rotate all secrets and credentials
kubectl delete secret -n production --all
./scripts/regenerate-secrets.sh

# 8. Full security audit
trivy k8s cluster --namespace production
kubectl run -n production security-audit --rm -i --tty \
  --image=aquasec/trivy -- sh
```

**Estimated Recovery Time:** 2-4 hours (includes verification and security audit)

### Scenario 5: GitOps Repository Corruption

**Symptoms:**
- GitOps repo contains malicious code
- ArgoCD deploying incorrect configurations
- Git history shows unauthorized changes

**Recovery Steps:**

```bash
#!/bin/bash
# Git repository recovery

# 1. Disable ArgoCD auto-sync immediately
kubectl patch application -n argocd --all -p '
{
  "spec": {
    "syncPolicy": {
      "automated": null
    }
  }
}' --type=merge

# 2. Clone repository to safe location
cd /tmp/recovery
git clone https://github.com/acme/devops-lifecycle.git devops-backup

# 3. Identify last known good commit
cd devops-backup
git log --oneline --graph --all | head -50

# Review suspicious commits
git show <suspicious-commit-hash>

# 4. Reset to last known good state
LAST_GOOD_COMMIT="abc123def"
git reset --hard $LAST_GOOD_COMMIT

# 5. Create new branch and force push (after approval)
git checkout -b recovery/$(date +%Y%m%d-%H%M%S)
git push origin recovery/$(date +%Y%m%d-%H%M%S) --force

# 6. Update all ArgoCD applications to use recovery branch
kubectl patch application -n argocd user-service-prod -p "
{
  \"spec\": {
    \"source\": {
      \"targetRevision\": \"recovery/20240115-143000\"
    }
  }
}" --type=merge

# 7. Manual sync after verification
argocd app sync user-service-prod --prune

# 8. Once stable, merge recovery branch to main
cd devops-backup
git checkout main
git reset --hard recovery/20240115-143000
git push origin main --force

# 9. Re-enable auto-sync
kubectl patch application -n argocd --all -p '
{
  "spec": {
    "syncPolicy": {
      "automated": {
        "prune": true,
        "selfHeal": true
      }
    }
  }
}' --type=merge

# 10. Rotate GitHub access tokens
# Via GitHub UI: Settings → Developer settings → Personal access tokens
# Update ArgoCD secret
kubectl create secret generic repo-credentials -n argocd \
  --from-literal=url=https://github.com/acme/devops-lifecycle \
  --from-literal=password=$NEW_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Estimated Recovery Time:** 1-2 hours

## 🔄 Backup Verification

### Weekly Backup Testing

```bash
#!/bin/bash
# Automated backup verification (run weekly)

BACKUP_NAME="production-daily-$(date -d yesterday +%Y%m%d)"

# 1. Create test namespace
kubectl create namespace backup-test

# 2. Restore backup to test namespace
velero restore create test-restore-$(date +%Y%m%d) \
  --from-backup $BACKUP_NAME \
  --namespace-mappings production:backup-test

# 3. Wait for restore to complete
velero restore wait test-restore-$(date +%Y%m%d)

# 4. Verify application pods start
kubectl wait --for=condition=Ready pod -l app=user-service -n backup-test --timeout=300s

# 5. Verify database connectivity
kubectl run -n backup-test db-test --rm -i --tty --image=postgres:15 -- \
  psql -h postgres-service -U postgres -c "SELECT count(*) FROM users;"

# 6. Verify data integrity
kubectl exec -n backup-test deployment/user-service -- \
  curl -s http://localhost:8080/api/v1/users | jq '.total'

# 7. Cleanup
kubectl delete namespace backup-test

# 8. Report results
echo "Backup verification completed: $BACKUP_NAME" | \
  mail -s "Backup Test Report" platform-team@acme.com
```

## 📊 Monitoring & Alerting

### Backup Health Alerts

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: backup-alerts
  namespace: monitoring
spec:
  groups:
  - name: backups
    rules:
    - alert: BackupFailed
      expr: |
        velero_backup_failure_total > 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Backup failure detected"
        description: "Velero backup has failed {{ $value }} times"

    - alert: BackupTooOld
      expr: |
        time() - velero_backup_last_successful_timestamp > 86400
      for: 1h
      labels:
        severity: warning
      annotations:
        summary: "Backup is too old"
        description: "Last successful backup was {{ $value }}s ago"

    - alert: DatabaseReplicationLag
      expr: |
        pg_replication_lag_seconds > 300
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Database replication lag high"
        description: "Replication lag is {{ $value }}s"
```

## 📝 Post-Disaster Actions

1. **Incident Report**: Document timeline, root cause, recovery steps
2. **Post-Mortem**: Conduct blameless post-mortem meeting
3. **Update Runbooks**: Add lessons learned
4. **Improve Automation**: Automate any manual recovery steps
5. **Test DR Plan**: Schedule next DR drill
6. **Review RPO/RTO**: Update targets based on business needs
7. **Insurance Claim**: If applicable, file insurance claim for losses

## 🔗 Related Resources

- **Velero Documentation**: https://velero.io/docs/
- **AWS Disaster Recovery**: https://aws.amazon.com/disaster-recovery/
- **Related Runbooks**:
  - High Error Rate Incident
  - Database Connection Issues
  - Pod Crash Loop

## 📞 Emergency Contacts

- **Incident Commander**: @incident-commander
- **Platform Team Lead**: @platform-lead
- **Database Team Lead**: @dba-lead
- **AWS Support**: 1-800-XXX-XXXX (Enterprise Support)
- **Security Team**: @security-team
- **Executive On-Call**: @exec-oncall

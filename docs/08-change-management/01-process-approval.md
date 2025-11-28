# Gestión del Cambio - Change Management

## 🎯 Objetivo

Gestionar y controlar todos los cambios en la infraestructura y aplicaciones, garantizando:
- **Trazabilidad**: Registro completo de todos los cambios
- **Aprobación**: Proceso de revisión antes de producción
- **Auditoría**: Cumplimiento y evidencia
- **Rollback**: Capacidad de revertir cambios

## 📋 Tipos de Cambios

### 1. Standard Change (Automatizado)

**Características:**
- Pre-aprobado
- Bajo riesgo
- Proceso documentado
- Automatizado

**Ejemplos:**
- Deploy de feature a dev/staging
- Scaling horizontal automático
- Parches de seguridad menores
- Actualización de configuración no crítica

**Proceso:**
```
Code Change → CI/CD → Auto-deploy → Monitoring → Done
```

### 2. Normal Change (Requiere Aprobación)

**Características:**
- Revisión técnica requerida
- Riesgo medio
- Ventana de cambio definida
- Aprobación de 2 personas

**Ejemplos:**
- Deploy a producción
- Cambios en base de datos
- Actualización de versión major
- Cambios en networking

**Proceso:**
```
RFC Creation → Technical Review → Approval → 
Scheduling → Implementation → Verification → Close
```

### 3. Emergency Change (Expedito)

**Características:**
- Urgente
- Proceso simplificado
- Post-approval permitido
- Documentación post-facto

**Ejemplos:**
- Hotfix crítico
- Security patch urgente
- Incident resolution
- Rollback de emergencia

**Proceso:**
```
Emergency → Emergency Approval → Implementation → 
Post-Implementation Review → Documentation
```

## 📝 Request for Change (RFC)

### Template RFC

```markdown
# RFC-2024-001: Migración a PostgreSQL 15

## Metadata
- **Fecha**: 2024-01-15
- **Autor**: John Doe
- **Tipo**: Normal Change
- **Severidad**: Media
- **Ambiente**: Production
- **Servicios Afectados**: user-service, order-service

## Descripción
Migración de PostgreSQL 13 a PostgreSQL 15 para aprovechar mejoras de 
performance y nuevas features.

## Justificación
- Performance improvements (30% faster queries)
- Security patches
- End of life de PostgreSQL 13: 2025-11-13

## Riesgo
**Nivel**: MEDIO

**Riesgos Identificados:**
1. Incompatibilidades en queries
2. Downtime durante migración
3. Performance degradation inicial

**Mitigación:**
1. Testing extensivo en staging
2. Migración durante ventana de mantenimiento
3. Plan de rollback documentado

## Impacto
- **Usuarios**: Downtime de 15 minutos (2 AM - 2:15 AM)
- **Servicios**: user-service, order-service temporalmente offline
- **Datos**: Backup completo antes de migración

## Plan de Implementación

### Pre-requisitos
- [ ] Backup completo de base de datos
- [ ] Testing en staging completado
- [ ] Documentación de rollback
- [ ] Notificación a stakeholders

### Steps
1. **2:00 AM**: Activar maintenance mode
2. **2:02 AM**: Crear backup final
3. **2:05 AM**: Detener servicios
4. **2:07 AM**: Dump de datos
5. **2:10 AM**: Restaurar en PostgreSQL 15
6. **2:12 AM**: Verificar integridad
7. **2:13 AM**: Iniciar servicios
8. **2:14 AM**: Smoke tests
9. **2:15 AM**: Desactivar maintenance mode

### Rollback Plan
Si falla en cualquier paso antes de 2:13 AM:
1. Restaurar desde backup
2. Reiniciar servicios en configuración anterior
3. Notificar falla

### Verificación
- [ ] Servicios healthy
- [ ] Latencia de queries < baseline
- [ ] Error rate < 0.1%
- [ ] Smoke tests passed

### Post-Implementation
- [ ] Monitoreo por 24 horas
- [ ] Eliminar PostgreSQL 13 después de 7 días
- [ ] Actualizar documentación

## Aprobaciones Requeridas
- [ ] Tech Lead: @tech-lead
- [ ] DBA: @database-admin
- [ ] Security: @security-team
- [ ] Product Owner: @product-owner

## Timeline
- **Creación**: 2024-01-15
- **Review**: 2024-01-16 - 2024-01-18
- **Aprobación**: 2024-01-19
- **Implementación**: 2024-01-21 2:00 AM
```

## ✅ Proceso de Aprobación

### 1. Creación de RFC

```bash
# Crear RFC desde template
cp docs/templates/rfc-template.md docs/08-change-management/rfcs/RFC-2024-001.md

# Completar información
vim docs/08-change-management/rfcs/RFC-2024-001.md

# Crear PR
git checkout -b rfc/RFC-2024-001
git add docs/08-change-management/rfcs/RFC-2024-001.md
git commit -m "docs(rfc): add RFC-2024-001 for PostgreSQL migration"
git push origin rfc/RFC-2024-001
```

### 2. Review Process

**Reviewers Automáticos (CODEOWNERS):**
```
# .github/CODEOWNERS
/docs/08-change-management/rfcs/  @tech-leads @platform-team
```

**Checklist de Review:**
- [ ] Justificación clara
- [ ] Riesgos identificados
- [ ] Plan de rollback documentado
- [ ] Testing plan adecuado
- [ ] Timeline realista
- [ ] Aprobaciones necesarias

### 3. Aprobación

```yaml
# GitHub Actions workflow
name: RFC Approval

on:
  pull_request:
    paths:
      - 'docs/08-change-management/rfcs/*.md'

jobs:
  approval-check:
    runs-on: ubuntu-latest
    steps:
      - name: Check approvals
        uses: actions/github-script@v6
        with:
          script: |
            const reviews = await github.rest.pulls.listReviews({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.payload.pull_request.number
            });
            
            const approvals = reviews.data.filter(
              review => review.state === 'APPROVED'
            );
            
            if (approvals.length < 2) {
              throw new Error('RFC requires at least 2 approvals');
            }
```

## 📊 Change Records

### Automated Change Log

```yaml
# .github/workflows/change-log.yml
name: Automated Change Log

on:
  push:
    branches:
      - main
    tags:
      - 'v*'

jobs:
  log-change:
    runs-on: ubuntu-latest
    steps:
      - name: Log to change management system
        run: |
          curl -X POST https://change-mgmt.acme.com/api/changes \
            -H "Authorization: Bearer ${{ secrets.CHANGE_MGMT_TOKEN }}" \
            -d '{
              "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
              "type": "deployment",
              "environment": "production",
              "service": "user-service",
              "version": "'${GITHUB_REF#refs/tags/}'",
              "commit": "'$GITHUB_SHA'",
              "author": "'$GITHUB_ACTOR'",
              "pr_number": "'${{ github.event.pull_request.number }}'"
            }'
```

### Change Log Format

```json
{
  "change_id": "CHG-2024-001",
  "rfc_id": "RFC-2024-001",
  "type": "normal",
  "status": "completed",
  "environment": "production",
  "services": ["user-service", "order-service"],
  "timestamp": "2024-01-21T02:00:00Z",
  "duration_minutes": 15,
  "implementer": "john.doe@acme.com",
  "approvers": [
    "tech-lead@acme.com",
    "dba@acme.com"
  ],
  "verification": {
    "smoke_tests": "passed",
    "error_rate": 0.05,
    "latency_p99": 120
  },
  "rollback_executed": false
}
```

## 🔄 Rollback Procedures

### Automatic Rollback

```yaml
# observability/prometheus/alerts/deployment-alerts.yaml
- alert: HighErrorRateAfterDeploy
  expr: |
    (
      sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)
      /
      sum(rate(http_requests_total[5m])) by (service)
    ) > 0.05
    and
    changes(deployment_version[10m]) > 0
  for: 5m
  labels:
    severity: critical
    action: rollback
  annotations:
    summary: "High error rate detected after deployment"
    description: "{{ $labels.service }} has {{ $value }}% error rate"
    runbook: "https://wiki.acme.com/runbooks/deployment-rollback"
```

### Manual Rollback

```bash
#!/bin/bash
# scripts/rollback-deployment.sh

SERVICE=$1
ENVIRONMENT=$2
PREVIOUS_VERSION=$3

echo "🔙 Rolling back $SERVICE in $ENVIRONMENT to $PREVIOUS_VERSION"

# Update image tag in GitOps repo
cd gitops-repo
git checkout main
git pull

# Revert to previous version
kustomize edit set image \
  $SERVICE=$REGISTRY/$SERVICE:$PREVIOUS_VERSION \
  -f environments/overlays/$ENVIRONMENT/$SERVICE/

# Commit and push
git add .
git commit -m "rollback($SERVICE): revert to $PREVIOUS_VERSION in $ENVIRONMENT"
git push origin main

# Wait for ArgoCD sync
echo "⏳ Waiting for ArgoCD sync..."
argocd app wait $SERVICE-$ENVIRONMENT \
  --timeout 300

echo "✅ Rollback completed"
```

## 📈 Metrics & Reporting

### Change Success Rate

```promql
# Successful changes in last 30 days
sum(changes_completed{status="success"}) / sum(changes_completed)
```

### Change Lead Time

```promql
# Average time from RFC to implementation
histogram_quantile(0.5,
  rate(change_lead_time_seconds_bucket[30d])
) / 86400
```

### Dashboard

```json
{
  "dashboard": {
    "title": "Change Management",
    "panels": [
      {
        "title": "Changes by Type",
        "type": "piechart",
        "targets": [{
          "expr": "sum(changes_total) by (type)"
        }]
      },
      {
        "title": "Change Success Rate",
        "type": "gauge",
        "targets": [{
          "expr": "sum(changes_completed{status=\"success\"}) / sum(changes_completed)"
        }]
      },
      {
        "title": "Changes Over Time",
        "type": "graph",
        "targets": [{
          "expr": "sum(rate(changes_total[1h])) by (environment)"
        }]
      }
    ]
  }
}
```

## 🔒 Compliance & Audit

### Audit Trail

Todos los cambios registrados en:
1. **Git History**: Código y configuración
2. **ArgoCD**: Kubernetes deployments
3. **Terraform State**: Infraestructura
4. **Change Management System**: Metadata y aprobaciones
5. **CloudTrail**: AWS API calls

### Compliance Report

```bash
#!/bin/bash
# scripts/generate-compliance-report.sh

MONTH=$1
YEAR=$2

echo "Generating compliance report for $MONTH/$YEAR"

# Get all changes
jq -r '.[] | select(.timestamp | startswith("'$YEAR-$MONTH'"))' \
  change-log.json > monthly-changes.json

# Generate report
cat <<EOF > compliance-report-$YEAR-$MONTH.md
# Change Management Compliance Report
**Period**: $MONTH/$YEAR

## Summary
- Total Changes: $(jq length monthly-changes.json)
- Standard Changes: $(jq '[.[] | select(.type=="standard")] | length' monthly-changes.json)
- Normal Changes: $(jq '[.[] | select(.type=="normal")] | length' monthly-changes.json)
- Emergency Changes: $(jq '[.[] | select(.type=="emergency")] | length' monthly-changes.json)

## Approval Compliance
- Changes with Required Approvals: $(jq '[.[] | select(.approvers | length >= 2)] | length' monthly-changes.json)
- Emergency Changes (Post-Approval): $(jq '[.[] | select(.type=="emergency")] | length' monthly-changes.json)

## Success Rate
- Successful: $(jq '[.[] | select(.status=="completed")] | length' monthly-changes.json)
- Failed: $(jq '[.[] | select(.status=="failed")] | length' monthly-changes.json)
- Rolled Back: $(jq '[.[] | select(.rollback_executed==true)] | length' monthly-changes.json)
EOF
```

## 📚 Documentation Requirements

### Post-Implementation Documentation

Después de cada cambio, actualizar:

1. **Runbooks**: Si el procedimiento cambia
2. **Architecture Docs**: Si hay cambios arquitectónicos
3. **Configuration Docs**: Nuevos parámetros o configs
4. **Lessons Learned**: Problemas encontrados y soluciones

### Example Post-Implementation Report

```markdown
# Post-Implementation Report: RFC-2024-001

## Summary
PostgreSQL migration from v13 to v15 completed successfully.

## Execution
- **Start**: 2024-01-21 02:00 AM
- **End**: 2024-01-21 02:18 AM
- **Duration**: 18 minutes (3 min over estimate)

## Deviations from Plan
- Smoke tests took 3 extra minutes due to connection pool warmup

## Issues Encountered
1. Initial connection failures
   - **Cause**: Connection pool needed warmup
   - **Resolution**: Extended verification time

## Lessons Learned
1. Include connection pool warmup in future database migrations
2. Increase time buffer for verification steps

## Verification Results
- ✅ All services healthy
- ✅ Query latency 25% improved (better than expected)
- ✅ Error rate: 0.02% (within threshold)
- ✅ All smoke tests passed

## Recommendations
1. Document connection pool warmup procedure
2. Create automated verification script
3. Update runbook with lessons learned
```

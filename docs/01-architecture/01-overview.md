# Arquitectura del Ciclo de Vida DevOps

## 1. Visión Estratégica

Este sistema implementa un **ciclo de vida DevOps completo** siguiendo principios de:
- **Continuous Everything**: CI/CD/Security/Delivery/Deployment
- **Infrastructure as Code**: Todo versionado, reproducible, auditable
- **GitOps**: Git como fuente única de verdad
- **Shift-Left Security**: Seguridad desde el inicio
- **Observability First**: Instrumentación antes de deployment

## 2. Arquitectura de Alto Nivel

```
┌─────────────────────────────────────────────────────────────────────┐
│                          DEVELOPER WORKFLOW                          │
├─────────────────────────────────────────────────────────────────────┤
│  Local Dev → Feature Branch → Pull Request → Code Review → Merge   │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          CI PIPELINE                                 │
├─────────────────────────────────────────────────────────────────────┤
│  1. Checkout                                                         │
│  2. Build (Docker multi-stage)                                       │
│  3. Unit Tests + Coverage                                            │
│  4. SAST (SonarQube, Semgrep)                                       │
│  5. SCA (Snyk, Trivy)                                               │
│  6. Container Scan (Trivy, Grype)                                   │
│  7. Push to Registry (Harbor)                                        │
│  8. Sign Image (Cosign)                                             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          CD PIPELINE                                 │
├─────────────────────────────────────────────────────────────────────┤
│  1. Update GitOps Manifests                                          │
│  2. ArgoCD Detects Changes                                           │
│  3. Pre-deployment Checks (OPA)                                      │
│  4. Deploy to Target Environment                                     │
│  5. Integration Tests                                                │
│  6. Smoke Tests                                                      │
│  7. Progressive Delivery (Canary/Blue-Green)                         │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          RUNTIME                                     │
├─────────────────────────────────────────────────────────────────────┤
│  • Kubernetes Cluster (EKS/GKE/AKS)                                 │
│  • Service Mesh (Istio)                                             │
│  • Ingress (NGINX/ALB)                                              │
│  • Secrets (Vault)                                                  │
│  • Runtime Security (Falco)                                         │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          OBSERVABILITY                               │
├─────────────────────────────────────────────────────────────────────┤
│  • Metrics: Prometheus + Grafana                                     │
│  • Logs: Loki + Fluentd                                             │
│  • Traces: Tempo + OpenTelemetry                                    │
│  • Alerts: AlertManager + PagerDuty                                 │
│  • Dashboards: Grafana + Custom                                     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼ (Feedback Loop)
                    ┌────────────────┐
                    │   DEVELOPER    │
                    └────────────────┘
```

## 3. Estrategia de Branching

### GitFlow Modificado

```
main (production)
  ↑
  └─ release/v1.2.0
       ↑
       └─ develop
            ↑
            ├─ feature/user-auth
            ├─ feature/payment-api
            └─ hotfix/critical-bug
```

### Convenciones de Naming

**Ramas:**
```
feature/<ticket-id>-<short-description>
bugfix/<ticket-id>-<short-description>
hotfix/<ticket-id>-<short-description>
release/v<major>.<minor>.<patch>
```

**Commits (Conventional Commits):**
```
feat(auth): add OAuth2 integration
fix(api): resolve rate limiting issue
docs(readme): update installation steps
chore(deps): upgrade terraform to 1.6
```

**Tags:**
```
v1.2.3                    # Production release
v1.2.3-rc.1              # Release candidate
v1.2.3-dev.20231128      # Development build
```

## 4. Entornos y Promoción

### Definición de Entornos

| Ambiente | Propósito | Actualización | Datos | Durabilidad |
|----------|-----------|---------------|-------|-------------|
| **dev** | Desarrollo rápido | Por cada commit a `develop` | Mocks/Sintéticos | Efímero |
| **staging** | Pre-producción | Por cada RC | Copia de producción (anonimizada) | Persistente |
| **production** | Usuarios reales | Por tag de release | Reales | HA + DR |

### Flujo de Promoción

```
┌────────┐   Auto    ┌─────────┐   Manual    ┌────────────┐
│  DEV   │ ────────► │ STAGING │ ──────────► │ PRODUCTION │
└────────┘           └─────────┘             └────────────┘
    ↑                     │                         │
    │                     └──── Integration Tests   │
    │                                               │
    └──────────────── Feedback / Rollback ──────────┘
```

## 5. Monorepo vs Multirepo

### Estrategia Adoptada: **Híbrida**

**Monorepo para:**
- Microservicios relacionados
- Librerías compartidas
- Configuraciones comunes
- GitOps manifests

**Repos Separados para:**
- IaC platform-level (Terraform/Crossplane)
- Documentación técnica
- Herramientas internas

### Estructura Monorepo

```
/
├── applications/          # Aplicaciones
│   ├── microservices/
│   │   ├── user-service/
│   │   ├── order-service/
│   │   └── notification-service/
│   └── frontends/
│       └── web-app/
├── infrastructure/        # IaC
│   ├── terraform/
│   ├── crossplane/
│   └── ansible/
├── gitops/               # GitOps configs
│   ├── argocd/
│   └── environments/
├── shared/               # Código compartido
│   ├── libraries/
│   ├── configs/
│   └── scripts/
├── pipelines/            # CI/CD definitions
├── security/             # Políticas OPA, scans
├── observability/        # Dashboards, alerts
└── docs/                 # Documentación
```

## 6. Versionado Semántico

Seguimos **SemVer 2.0.0**: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes
- **MINOR**: Nuevas features (backward compatible)
- **PATCH**: Bug fixes

**Ejemplos:**
```
v1.0.0 → v1.1.0  (nueva feature)
v1.1.0 → v1.1.1  (bug fix)
v1.1.1 → v2.0.0  (breaking change)
```

## 7. Convenciones de Naming

### Recursos Cloud (AWS)

```
<org>-<env>-<region>-<service>-<resource>-<id>

Ejemplos:
acme-prod-us-east-1-eks-cluster-01
acme-staging-eu-west-1-rds-postgres-users
```

### Kubernetes Resources

```
<app>-<component>-<env>

Ejemplos:
user-service-api-prod
order-service-worker-staging
```

### Docker Images

```
<registry>/<org>/<app>:<tag>

Ejemplos:
harbor.acme.com/platform/user-service:v1.2.3
harbor.acme.com/platform/user-service:sha-abc123f
```

## 8. Decision Records

Documentamos decisiones arquitectónicas importantes en `/docs/adr/`:

- **ADR-001**: Adopción de GitOps con ArgoCD
- **ADR-002**: Multi-cloud strategy (AWS primary, GCP DR)
- **ADR-003**: Istio como service mesh
- **ADR-004**: Vault para secrets management

## 9. Próximos Pasos

1. Leer [IaC Structure](../02-iac/01-terraform-structure.md)
2. Revisar [CI/CD Pipelines](../03-cicd/01-pipeline-design.md)
3. Implementar [Security Scanning](../04-security/01-shift-left.md)

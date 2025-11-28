# Estructura del Repositorio - Detalles

## 📂 Árbol Completo del Repositorio

```
devops-lifecycle/
│
├── .github/
│   ├── workflows/                    # GitHub Actions pipelines
│   │   ├── ci-microservice.yml
│   │   ├── cd-deploy.yml
│   │   ├── security-scan.yml
│   │   ├── iac-validation.yml
│   │   └── release.yml
│   ├── CODEOWNERS                    # Code review assignments
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── copilot-instructions.md
│
├── applications/                     # Aplicaciones del sistema
│   ├── microservices/
│   │   ├── user-service/            # Servicio de usuarios
│   │   │   ├── cmd/
│   │   │   │   └── api/
│   │   │   │       └── main.go
│   │   │   ├── internal/
│   │   │   │   ├── handlers/
│   │   │   │   ├── models/
│   │   │   │   ├── repository/
│   │   │   │   └── services/
│   │   │   ├── pkg/
│   │   │   ├── tests/
│   │   │   ├── Dockerfile
│   │   │   ├── Makefile
│   │   │   ├── go.mod
│   │   │   └── .gitlab-ci.yml
│   │   │
│   │   ├── order-service/           # Servicio de órdenes
│   │   │   └── [similar structure]
│   │   │
│   │   └── notification-service/    # Servicio de notificaciones
│   │       └── [similar structure]
│   │
│   └── frontends/
│       └── web-app/                 # Frontend React
│           ├── src/
│           ├── public/
│           ├── package.json
│           └── Dockerfile
│
├── infrastructure/                   # Infrastructure as Code
│   │
│   ├── terraform/                   # Terraform configurations
│   │   ├── bootstrap/               # Account setup inicial
│   │   │   ├── main.tf
│   │   │   ├── backend.tf
│   │   │   ├── providers.tf
│   │   │   └── variables.tf
│   │   │
│   │   ├── modules/                 # Terraform modules reutilizables
│   │   │   ├── vpc/
│   │   │   │   ├── main.tf
│   │   │   │   ├── variables.tf
│   │   │   │   ├── outputs.tf
│   │   │   │   └── README.md
│   │   │   ├── eks/
│   │   │   ├── rds/
│   │   │   ├── elasticache/
│   │   │   └── s3-bucket/
│   │   │
│   │   ├── environments/            # Configuraciones por ambiente
│   │   │   ├── dev/
│   │   │   │   ├── main.tf
│   │   │   │   ├── terraform.tfvars
│   │   │   │   └── backend.tf
│   │   │   ├── staging/
│   │   │   └── production/
│   │   │
│   │   └── global/                  # Recursos globales
│   │       ├── route53/
│   │       ├── cloudfront/
│   │       └── iam/
│   │
│   ├── crossplane/                  # Crossplane compositions
│   │   ├── compositions/
│   │   │   ├── database.yaml
│   │   │   ├── cache.yaml
│   │   │   └── storage.yaml
│   │   └── claims/
│   │       └── examples/
│   │
│   └── ansible/                     # Configuration Management
│       ├── playbooks/
│       │   ├── setup-monitoring.yml
│       │   ├── configure-nodes.yml
│       │   └── disaster-recovery.yml
│       ├── roles/
│       └── inventory/
│
├── gitops/                          # GitOps configurations
│   ├── argocd/
│   │   ├── bootstrap/               # ArgoCD self-management
│   │   │   ├── install.yaml
│   │   │   └── app-of-apps.yaml
│   │   │
│   │   ├── apps/                    # Application definitions
│   │   │   ├── user-service.yaml
│   │   │   ├── order-service.yaml
│   │   │   └── monitoring.yaml
│   │   │
│   │   └── projects/                # ArgoCD projects
│   │       ├── platform.yaml
│   │       └── applications.yaml
│   │
│   └── environments/                # Kubernetes manifests por ambiente
│       ├── base/                    # Base configurations
│       │   ├── user-service/
│       │   │   ├── deployment.yaml
│       │   │   ├── service.yaml
│       │   │   ├── hpa.yaml
│       │   │   └── kustomization.yaml
│       │   └── order-service/
│       │
│       ├── overlays/                # Environment-specific overlays
│       │   ├── dev/
│       │   │   ├── kustomization.yaml
│       │   │   └── patches/
│       │   ├── staging/
│       │   └── production/
│       │       ├── kustomization.yaml
│       │       ├── patches/
│       │       └── sealed-secrets/
│       │
│       └── helm-charts/             # Custom Helm charts
│           └── microservice-template/
│
├── shared/                          # Código y configs compartidos
│   ├── libraries/
│   │   ├── go-commons/             # Librería Go compartida
│   │   │   ├── logger/
│   │   │   ├── middleware/
│   │   │   ├── tracing/
│   │   │   └── metrics/
│   │   └── proto/                   # Protocol Buffers
│   │       └── api/
│   │
│   ├── configs/                     # Configuraciones compartidas
│   │   ├── eslint/
│   │   ├── prettier/
│   │   └── sonarqube/
│   │
│   └── scripts/                     # Scripts de automatización
│       ├── setup-local-dev.sh
│       ├── generate-secrets.sh
│       └── backup-databases.sh
│
├── pipelines/                       # Definiciones CI/CD
│   ├── gitlab/
│   │   ├── templates/
│   │   │   ├── build.gitlab-ci.yml
│   │   │   ├── test.gitlab-ci.yml
│   │   │   ├── security.gitlab-ci.yml
│   │   │   └── deploy.gitlab-ci.yml
│   │   └── .gitlab-ci.yml
│   │
│   ├── jenkins/
│   │   ├── Jenkinsfile.groovy
│   │   └── shared-library/
│   │
│   └── github-actions/
│       └── [already in .github/workflows/]
│
├── security/                        # Security configurations
│   ├── policies/                    # OPA policies
│   │   ├── kubernetes/
│   │   │   ├── deny-privileged-containers.rego
│   │   │   ├── require-resource-limits.rego
│   │   │   └── enforce-security-context.rego
│   │   ├── terraform/
│   │   │   ├── require-encryption.rego
│   │   │   └── enforce-tagging.rego
│   │   └── docker/
│   │       └── deny-latest-tag.rego
│   │
│   ├── scanning/                    # Security scan configs
│   │   ├── trivy-config.yaml
│   │   ├── snyk-config.json
│   │   └── sonarqube-project.properties
│   │
│   ├── vault/                       # Vault configurations
│   │   ├── policies/
│   │   └── auth-methods/
│   │
│   └── falco/                       # Falco runtime security
│       ├── rules/
│       └── config.yaml
│
├── observability/                   # Monitoring & Observability
│   ├── prometheus/
│   │   ├── alerts/
│   │   │   ├── infrastructure.yaml
│   │   │   ├── applications.yaml
│   │   │   └── slo-alerts.yaml
│   │   ├── rules/
│   │   └── prometheus.yaml
│   │
│   ├── grafana/
│   │   ├── dashboards/
│   │   │   ├── kubernetes-cluster.json
│   │   │   ├── microservices.json
│   │   │   ├── golden-signals.json
│   │   │   └── business-metrics.json
│   │   └── datasources.yaml
│   │
│   ├── loki/
│   │   └── config.yaml
│   │
│   ├── tempo/
│   │   └── config.yaml
│   │
│   └── opentelemetry/
│       ├── collector-config.yaml
│       └── instrumentation-examples/
│
├── docs/                            # Documentación técnica
│   ├── 01-architecture/
│   │   ├── 01-overview.md
│   │   ├── 02-repository-structure.md
│   │   └── diagrams/
│   │
│   ├── 02-iac/
│   │   ├── 01-terraform-structure.md
│   │   ├── 02-module-development.md
│   │   └── 03-crossplane-guide.md
│   │
│   ├── 03-cicd/
│   │   ├── 01-pipeline-design.md
│   │   ├── 02-deployment-strategies.md
│   │   └── 03-rollback-procedures.md
│   │
│   ├── 04-security/
│   │   ├── 01-shift-left.md
│   │   ├── 02-opa-policies.md
│   │   └── 03-secrets-management.md
│   │
│   ├── 05-observability/
│   │   ├── 01-metrics-instrumentation.md
│   │   ├── 02-logging-standards.md
│   │   └── 03-distributed-tracing.md
│   │
│   ├── 06-gitops/
│   │   ├── 01-argocd-setup.md
│   │   └── 02-kustomize-patterns.md
│   │
│   ├── 07-governance/
│   │   ├── 01-standards.md
│   │   └── 02-compliance.md
│   │
│   ├── 08-change-management/
│   │   └── 01-approval-workflows.md
│   │
│   ├── 09-runbooks/
│   │   ├── incident-response.md
│   │   ├── disaster-recovery.md
│   │   └── common-issues.md
│   │
│   └── adr/                         # Architecture Decision Records
│       ├── 001-gitops-adoption.md
│       ├── 002-multi-cloud.md
│       └── 003-service-mesh.md
│
├── tests/                           # Tests de integración y E2E
│   ├── integration/
│   ├── e2e/
│   └── performance/
│
├── .env.example                     # Template de variables de entorno
├── .gitignore
├── .pre-commit-config.yaml         # Pre-commit hooks
├── CONTRIBUTING.md
├── LICENSE
├── Makefile                         # Comandos comunes
└── README.md
```

## 🎯 Convenciones de Organización

### 1. Separación por Concerns

- **applications/**: Código de aplicación
- **infrastructure/**: Definición de infraestructura
- **gitops/**: Configuración de deployment
- **security/**: Políticas y escaneos
- **observability/**: Monitoreo y alertas

### 2. DRY (Don't Repeat Yourself)

- Módulos de Terraform reutilizables en `infrastructure/terraform/modules/`
- Librerías compartidas en `shared/libraries/`
- Templates de CI/CD en `pipelines/*/templates/`

### 3. Environment Parity

Cada ambiente sigue la misma estructura, solo difieren en valores de configuración.

## 📝 Archivos Clave

| Archivo | Propósito |
|---------|-----------|
| `Makefile` | Comandos comunes para developers |
| `.env.example` | Template de configuración local |
| `.pre-commit-config.yaml` | Validaciones pre-commit |
| `CODEOWNERS` | Asignación de code review |

## 🔍 Navegación Rápida

Para encontrar algo específico:

```bash
# Ver estructura de un microservicio
ls applications/microservices/user-service/

# Ver módulos de Terraform disponibles
ls infrastructure/terraform/modules/

# Ver dashboards de Grafana
ls observability/grafana/dashboards/

# Ver políticas de OPA
ls security/policies/
```

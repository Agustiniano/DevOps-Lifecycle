# Copilot Instructions - DevOps Lifecycle Framework

## 🎯 Project Overview

This is a **complete, production-ready DevOps Lifecycle framework** that models real-world engineering practices from code commit to production deployment. It's a reference implementation, not just documentation.

### Architecture

```
Developer → Git → CI Pipeline → Security Scans → Container Registry → 
GitOps (ArgoCD) → Kubernetes → Observability → Feedback Loop
```

### Major Components

1. **IaC Layer**: Terraform modules (VPC, EKS, RDS) + Crossplane compositions (app-specific resources)
2. **CI/CD**: Multi-stage pipelines with security gates, automated testing, and GitOps integration
3. **Security**: Shift-left approach with SAST, SCA, container scanning, OPA policies, runtime protection
4. **GitOps**: ArgoCD with App of Apps pattern, Kustomize overlays, progressive delivery
5. **Observability**: Prometheus (metrics), Loki (logs), Tempo (traces), Grafana (visualization)
6. **Kubernetes**: EKS clusters with Istio service mesh, autoscaling, network policies

### Key Design Decisions

- **Hybrid Repo Strategy**: Monorepo for apps/configs, separate for platform IaC
- **GitFlow Modified**: Feature branches, release branches, protected main/develop
- **Terraform for Base Infra**: Long-lived infrastructure (VPC, clusters)
- **Crossplane for App Resources**: Dynamic, app-lifecycle-bound resources (databases, caches)
- **GitOps for Deployments**: ArgoCD as deployment engine, Git as source of truth
- **Multi-Environment**: dev (auto-deploy), staging (RC testing), production (manual approval)

## 🚀 Developer Workflows

### Local Development Setup

```bash
# 1. Clone repository
git clone <repo-url> && cd "DevOps Lifecycle"

# 2. Install pre-commit hooks
pip install pre-commit
pre-commit install

# 3. Setup local environment
make setup-local
```

### Feature Development Flow

```bash
# 1. Create feature branch
git checkout develop
git checkout -b feature/TICKET-123-description

# 2. Develop with tests
# Run tests locally: make test
# Run security scan: make security-scan

# 3. Commit with conventional commits
git commit -m "feat(user-service): add OAuth2 integration"

# 4. Push and create PR
git push origin feature/TICKET-123-description
# CI pipeline runs automatically on PR

# 5. After approval, merge triggers deploy to dev
```

### Build Commands

```bash
# Microservice (Go)
cd applications/microservices/user-service
make build          # Build binary
make test           # Run tests
make coverage       # Check coverage
make docker-build   # Build container
make docker-push    # Push to Harbor

# Infrastructure
cd infrastructure/terraform/environments/production
terraform init
terraform plan -out=tfplan
terraform apply tfplan

# GitOps sync
kubectl apply -f gitops/argocd/apps/user-service.yaml
argocd app sync user-service-prod
```

### Debugging

- **Logs**: `kubectl logs -f deployment/user-service -n production` or Grafana/Loki
- **Metrics**: Grafana dashboards or `kubectl port-forward svc/prometheus 9090`
- **Traces**: Grafana/Tempo for distributed tracing
- **Live Debug**: `kubectl debug pod/user-service-xxx -it --image=busybox`

### Deployment Process

1. **To Dev**: Auto-deploy on merge to `develop`
2. **To Staging**: Create release branch, auto-deploy for QA
3. **To Production**: Tag release, requires manual approval in GitHub Actions

## 📐 Conventions & Patterns

### Naming Conventions

**Git Branches:**
```
feature/TICKET-ID-description
bugfix/TICKET-ID-description
hotfix/TICKET-ID-description
release/v1.2.3
```

**Commits (Conventional Commits):**
```
feat(scope): description
fix(scope): description
docs(scope): description
chore(deps): description
```

**Cloud Resources (AWS):**
```
<org>-<env>-<region>-<service>-<resource>-<id>
Example: acme-prod-us-east-1-eks-cluster-01
```

**Kubernetes Resources:**
```
<app>-<component>-<env>
Example: user-service-api-prod
```

**Docker Images:**
```
harbor.acme.com/platform/<service>:<tag>
Example: harbor.acme.com/platform/user-service:v1.2.3
```

### Code Structure Patterns

**Go Microservice Layout:**
```
cmd/api/main.go              # Entry point
internal/                    # Private application code
  handlers/                  # HTTP handlers
  models/                    # Data models
  repository/                # Data access
  services/                  # Business logic
pkg/                         # Public libraries
tests/                       # Integration tests
```

**IaC Module Pattern:**
```
modules/<resource>/
  main.tf                    # Resource definitions
  variables.tf               # Input variables
  outputs.tf                 # Output values
  versions.tf                # Provider versions
  README.md                  # Usage documentation
  examples/                  # Example usage
```

**Kustomize Pattern:**
```
base/                        # Common configs
  kustomization.yaml
  deployment.yaml
overlays/
  dev/kustomization.yaml     # Dev-specific patches
  staging/kustomization.yaml
  production/kustomization.yaml
```

### Security Patterns

- **No Secrets in Code**: Use Vault or Sealed Secrets
- **Image Signing**: All images signed with Cosign
- **SBOM Generation**: Every build generates Software Bill of Materials
- **Network Policies**: Default deny, explicit allow
- **RBAC**: Least privilege for all service accounts
- **OPA Policies**: Enforce security standards at admission time

### Testing Patterns

- **Unit Tests**: 80% minimum coverage
- **Integration Tests**: Test with real dependencies (testcontainers)
- **E2E Tests**: Smoke tests post-deployment
- **Security Tests**: SAST, SCA, container scanning in every pipeline
- **Performance Tests**: Load testing before production

## 📁 Key Files & Directories

### Critical Entry Points

- `README.md`: Project overview and quick start
- `.github/workflows/ci-microservice.yml`: Main CI pipeline
- `infrastructure/terraform/environments/production/main.tf`: Production infrastructure
- `gitops/argocd/bootstrap/app-of-apps.yaml`: ArgoCD root application
- `applications/microservices/user-service/cmd/api/main.go`: Microservice entry

### Configuration Files

- `.pre-commit-config.yaml`: Pre-commit hooks (security scans, linting)
- `sonar-project.properties`: SonarQube configuration
- `security/policies/kubernetes/*.rego`: OPA policies for K8s admission control
- `observability/prometheus/alerts/*.yaml`: Prometheus alerting rules
- `gitops/environments/overlays/*/kustomization.yaml`: Environment configs

### Documentation Structure

- `docs/01-architecture/`: System architecture and design decisions
- `docs/02-iac/`: Infrastructure as Code guides (Terraform, Crossplane)
- `docs/03-cicd/`: Pipeline design and deployment strategies
- `docs/04-security/`: DevSecOps practices and tools
- `docs/05-observability/`: Monitoring, logging, tracing setup
- `docs/06-gitops/`: ArgoCD and GitOps workflows
- `docs/07-governance/`: Standards and compliance
- `docs/08-change-management/`: Approval workflows and auditing
- `docs/09-runbooks/`: Operational procedures

## 🔧 Common Tasks

### Add New Microservice

1. Copy template: `cp -r applications/microservices/user-service applications/microservices/new-service`
2. Update configs in new service directory
3. Create Kustomize base: `gitops/environments/base/new-service/`
4. Create ArgoCD app: `gitops/argocd/apps/applications/new-service.yaml`
5. Add CI workflow: `.github/workflows/ci-new-service.yml`

### Add New Terraform Module

1. Create module: `infrastructure/terraform/modules/new-resource/`
2. Add main.tf, variables.tf, outputs.tf, README.md
3. Write tests using Terratest
4. Use in environment: Reference from `environments/*/main.tf`

### Deploy New Version

1. Tag release: `git tag -a v1.2.3 -m "Release v1.2.3"`
2. Push tag: `git push origin v1.2.3`
3. CI builds and pushes image
4. Update GitOps: ArgoCD detects and syncs
5. Monitor deployment in Grafana

### Troubleshoot Deployment

1. Check ArgoCD: `argocd app get user-service-prod`
2. Check pod status: `kubectl get pods -n production`
3. View logs: Grafana → Explore → Loki → `{app="user-service"}`
4. Check metrics: Grafana → Dashboards → "Microservices Overview"
5. Rollback if needed: `argocd app rollback user-service-prod`

## 🎓 Learning Path for New Contributors

1. Read `docs/01-architecture/01-overview.md` for big picture
2. Study example microservice: `applications/microservices/user-service/`
3. Review CI pipeline: `.github/workflows/ci-microservice.yml`
4. Understand GitOps: `docs/06-gitops/01-argocd-complete.md`
5. Practice: Make change to dev environment and watch it deploy

## ⚠️ Important Notes

- **Never kubectl apply directly to production** - use GitOps only
- **All infrastructure changes via Terraform** - no manual AWS console changes
- **Secrets never in Git** - use Vault or Sealed Secrets
- **Every deploy must pass security scans** - no exceptions
- **Monitor after deploy** - watch metrics for 15 minutes post-deployment

# Gobernanza y Estándares

## 🎯 Propósito

Establecer estándares, políticas y procesos de gobernanza para garantizar:
- **Consistencia**: Todas las aplicaciones siguen las mismas prácticas
- **Seguridad**: Cumplimiento de políticas de seguridad
- **Calidad**: Código y recursos cumplen estándares mínimos
- **Auditoría**: Trazabilidad completa de cambios

## 📋 Estándares de Código

### 1. Conventional Commits

Todos los commits DEBEN seguir la especificación:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types permitidos:**
- `feat`: Nueva funcionalidad
- `fix`: Bug fix
- `docs`: Solo documentación
- `style`: Formato, sin cambio de lógica
- `refactor`: Refactoring
- `perf`: Mejora de performance
- `test`: Tests
- `chore`: Mantenimiento

**Ejemplos:**
```
feat(auth): add OAuth2 integration
fix(api): resolve null pointer exception
docs(readme): update installation steps
```

### 2. Code Coverage

- **Mínimo**: 80% de cobertura
- **Target**: 90% de cobertura
- **Critical paths**: 100% de cobertura

### 3. Linting & Formatting

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - golint
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosec
    - ineffassign
    - misspell
```

## 🔒 Estándares de Seguridad

### 1. Secrets Management

**Prohibido:**
- ❌ Hardcoded credentials
- ❌ Secrets en variables de entorno sin encriptar
- ❌ Secrets en ConfigMaps

**Requerido:**
- ✅ Vault para secrets dinámicos
- ✅ Sealed Secrets para K8s
- ✅ AWS Secrets Manager para cloud resources

### 2. Image Security

**Requerimientos:**
- ✅ Images escaneadas con Trivy/Snyk
- ✅ No vulnerabilidades CRITICAL
- ✅ Máximo 5 vulnerabilidades HIGH
- ✅ Imágenes firmadas con Cosign
- ✅ SBOM generado

### 3. Container Security

**Requerimientos en Deployment:**
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

### 4. Network Policies

Todas las aplicaciones DEBEN tener NetworkPolicy:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: user-service
spec:
  podSelector:
    matchLabels:
      app: user-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: production
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
```

## 📦 Estándares de Recursos

### 1. Resource Limits

Todas las aplicaciones DEBEN especificar:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. Naming Conventions

**Recursos K8s:**
```
<app>-<component>-<env>

Ejemplos:
user-service-api-prod
order-service-worker-staging
```

**Recursos AWS:**
```
<org>-<env>-<region>-<service>-<resource>

Ejemplos:
acme-prod-us-east-1-eks-cluster
acme-staging-eu-west-1-rds-postgres
```

### 3. Labels Requeridos

```yaml
metadata:
  labels:
    app: user-service           # REQUIRED
    version: v1.0.0             # REQUIRED
    component: api              # REQUIRED
    part-of: platform           # REQUIRED
    managed-by: argocd          # REQUIRED
    environment: production     # REQUIRED
```

## 🚦 Quality Gates

### 1. Pull Request

**Requerimientos para merge:**
- ✅ CI pipeline passed
- ✅ Code coverage ≥ 80%
- ✅ Security scans passed
- ✅ At least 2 approvals
- ✅ No merge conflicts
- ✅ Branch up to date with target

### 2. SonarQube Quality Gate

```json
{
  "conditions": [
    {
      "metric": "new_coverage",
      "op": "LT",
      "error": "80"
    },
    {
      "metric": "new_duplicated_lines_density",
      "op": "GT",
      "error": "3"
    },
    {
      "metric": "new_security_hotspots_reviewed",
      "op": "LT",
      "error": "100"
    }
  ]
}
```

### 3. Deployment to Production

**Requerimientos:**
- ✅ Tag versionado (semver)
- ✅ Release notes completadas
- ✅ Smoke tests passed en staging
- ✅ Performance tests passed
- ✅ Security scan < 7 días
- ✅ Aprobación manual (2 personas)

## 📝 Templates

### 1. Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests pass locally
- [ ] Security scan passed
```

### 2. Issue Template

```markdown
## Description
Clear description of the issue

## Steps to Reproduce
1. Step 1
2. Step 2
3. Step 3

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- Service: user-service
- Version: v1.2.3
- Environment: production

## Additional Context
Logs, screenshots, etc.
```

## 🔍 Compliance & Auditing

### 1. Change Logging

Todos los cambios registrados en:
- Git commit history
- ArgoCD sync history
- Terraform state
- CloudTrail (AWS)

### 2. Access Control

**RBAC Matrix:**

| Role | Code Repo | K8s Dev | K8s Staging | K8s Prod | Terraform |
|------|-----------|---------|-------------|----------|-----------|
| Developer | Read/Write | Admin | Read | Read | Read |
| Senior Dev | Read/Write | Admin | Admin | Read | Read/Write |
| Platform | Read/Write | Admin | Admin | Admin | Admin |
| SRE | Read | Admin | Admin | Admin | Admin |

### 3. Retention Policies

- **Git history**: Indefinido
- **Container images**: 90 días (excepto tags versionados)
- **Logs**: 30 días (hot), 1 año (cold)
- **Metrics**: 30 días (Prometheus), 1 año (long-term storage)
- **Terraform state**: Indefinido (con versionado)

## 🎓 Training & Onboarding

### 1. Onboarding Checklist

- [ ] Read architecture documentation
- [ ] Setup local development environment
- [ ] Complete security training
- [ ] Review code standards
- [ ] Make first PR (documentation)
- [ ] Deploy to dev environment
- [ ] Shadow on-call rotation

### 2. Required Certifications

**Platform Team:**
- ☑️ CKA (Certified Kubernetes Administrator)
- ☑️ AWS Solutions Architect
- ☑️ Terraform Associate

**Developers:**
- ☑️ CKAD (Certified Kubernetes Application Developer)
- ☑️ Security awareness training (annual)

## 📊 Metrics & KPIs

### DORA Metrics

**Target:**
- **Deployment Frequency**: Daily
- **Lead Time for Changes**: < 4 hours
- **MTTR**: < 30 minutes
- **Change Failure Rate**: < 15%

**Current (ejemplo):**
- Deployment Frequency: 2.5 per day
- Lead Time: 3.2 hours
- MTTR: 25 minutes
- Change Failure Rate: 12%

### Security Metrics

- **Critical vulnerabilities**: 0
- **High vulnerabilities**: < 5 per service
- **SAST issues**: < 10 per service
- **Security scan coverage**: 100%

## 🚨 Violation Handling

### 1. Policy Violations

**Automated:**
- OPA policies block non-compliant resources
- Pre-commit hooks prevent bad commits
- CI pipelines fail on quality gates

**Manual:**
- Security team review for exceptions
- Architecture review for design decisions
- Post-mortem for incidents

### 2. Exception Process

```yaml
# Exception Request
exception:
  policy: require-resource-limits
  service: legacy-app
  reason: "Migration in progress"
  expires: 2024-12-31
  approvers:
    - security-team
    - platform-team
  compensating-controls:
    - Manual monitoring
    - Resource alerts configured
```

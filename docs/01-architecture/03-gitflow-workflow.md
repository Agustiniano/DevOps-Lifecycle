# GitFlow Workflow - Guía Práctica

## 🌿 Estrategia de Branching

### Ramas Principales (Long-lived)

```
main           (producción, protegida)
develop        (integración, protegida)
```

### Ramas de Soporte (Short-lived)

```
feature/*      (nuevas funcionalidades)
bugfix/*       (corrección de bugs en develop)
hotfix/*       (corrección urgente en producción)
release/*      (preparación de release)
```

## 📋 Flujo Completo

### 1. Feature Development

```bash
# Crear feature branch desde develop
git checkout develop
git pull origin develop
git checkout -b feature/AUTH-123-oauth-integration

# Desarrollo...
git add .
git commit -m "feat(auth): implement OAuth2 provider integration"

# Push y crear PR
git push origin feature/AUTH-123-oauth-integration
```

**Naming Convention:**
```
feature/<ticket-id>-<short-description>

Examples:
feature/JIRA-1234-user-authentication
feature/GH-567-payment-gateway
feature/add-logging-middleware  (si no hay ticket)
```

### 2. Code Review & Merge

```yaml
# Pull Request Requirements
required_checks:
  - CI pipeline passed
  - Code coverage > 80%
  - Security scans passed
  - At least 2 approvals
  - No merge conflicts
  - Branch up to date with develop
```

### 3. Release Preparation

```bash
# Crear release branch desde develop
git checkout develop
git pull origin develop
git checkout -b release/v1.5.0

# Bump version
npm version 1.5.0  # o go, python equivalente
git push origin release/v1.5.0

# Fix last-minute bugs en release branch
git commit -m "fix(release): adjust configuration for production"

# Merge a main Y develop
git checkout main
git merge --no-ff release/v1.5.0
git tag -a v1.5.0 -m "Release version 1.5.0"
git push origin main --tags

git checkout develop
git merge --no-ff release/v1.5.0
git push origin develop

# Eliminar release branch
git branch -d release/v1.5.0
git push origin --delete release/v1.5.0
```

### 4. Hotfix en Producción

```bash
# Crear hotfix desde main
git checkout main
git pull origin main
git checkout -b hotfix/CRITICAL-999-memory-leak

# Fix the issue
git commit -m "fix(api): resolve memory leak in connection pool"

# Merge a main Y develop
git checkout main
git merge --no-ff hotfix/CRITICAL-999-memory-leak
git tag -a v1.5.1 -m "Hotfix: memory leak"
git push origin main --tags

git checkout develop
git merge --no-ff hotfix/CRITICAL-999-memory-leak
git push origin develop

# Cleanup
git branch -d hotfix/CRITICAL-999-memory-leak
git push origin --delete hotfix/CRITICAL-999-memory-leak
```

## 🏷️ Conventional Commits

### Formato

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Descripción | Ejemplo |
|------|-------------|---------|
| `feat` | Nueva funcionalidad | `feat(auth): add JWT token validation` |
| `fix` | Bug fix | `fix(api): resolve null pointer exception` |
| `docs` | Solo documentación | `docs(readme): update installation steps` |
| `style` | Formato, sin cambio de lógica | `style(code): apply prettier formatting` |
| `refactor` | Refactoring sin cambio funcional | `refactor(db): optimize query performance` |
| `perf` | Mejora de performance | `perf(cache): implement Redis caching layer` |
| `test` | Agregar/corregir tests | `test(auth): add unit tests for login flow` |
| `chore` | Mantenimiento, deps, build | `chore(deps): upgrade terraform to 1.6.0` |
| `ci` | Cambios en CI/CD | `ci(github): add security scanning job` |
| `build` | Cambios en build system | `build(docker): optimize multi-stage build` |
| `revert` | Revertir commit anterior | `revert: feat(auth): add JWT token validation` |

### Scopes (ejemplos)

```
auth, api, db, cache, ui, config, docs, ci, terraform, k8s
```

### Breaking Changes

```bash
git commit -m "feat(api)!: change authentication endpoint structure

BREAKING CHANGE: The /auth endpoint now requires a client_id parameter.
Clients must be updated to include this in all authentication requests."
```

## 🏷️ Tagging Strategy

### Production Tags

```bash
# Major.Minor.Patch (SemVer)
v1.0.0    # Initial production release
v1.1.0    # New features
v1.1.1    # Bug fix
v2.0.0    # Breaking changes
```

### Pre-release Tags

```bash
v1.2.0-rc.1       # Release candidate 1
v1.2.0-rc.2       # Release candidate 2
v1.2.0-beta.1     # Beta release
v1.2.0-alpha.1    # Alpha release
```

### Development Tags

```bash
v1.2.0-dev.20231128-sha123abc   # Development build
```

### Crear Tag con Mensaje

```bash
git tag -a v1.2.0 -m "Release v1.2.0

Features:
- OAuth2 integration
- New payment gateway
- Performance improvements

Bug Fixes:
- Memory leak in connection pool
- Race condition in cache layer

Breaking Changes:
- API endpoint structure changed
"

git push origin v1.2.0
```

## 🔒 Branch Protection Rules

### Main Branch

```yaml
branch_protection:
  required_status_checks:
    strict: true
    contexts:
      - ci/build
      - ci/test
      - security/scan
      - ci/integration-tests
  
  required_pull_request_reviews:
    required_approving_review_count: 2
    dismiss_stale_reviews: true
    require_code_owner_reviews: true
  
  restrictions:
    users: []
    teams:
      - platform-team
      - senior-engineers
  
  enforce_admins: true
  allow_force_pushes: false
  allow_deletions: false
```

### Develop Branch

```yaml
branch_protection:
  required_status_checks:
    strict: true
    contexts:
      - ci/build
      - ci/test
      - security/scan
  
  required_pull_request_reviews:
    required_approving_review_count: 1
  
  enforce_admins: false
  allow_force_pushes: false
```

## 🔄 Merge Strategies

### Feature → Develop

```bash
# Squash merge (mantiene develop limpio)
git checkout develop
git merge --squash feature/AUTH-123-oauth
git commit -m "feat(auth): implement OAuth2 integration (#123)"
```

### Release/Hotfix → Main

```bash
# Merge commit (preserva historia)
git checkout main
git merge --no-ff release/v1.5.0
```

## 📊 Workflow Diagram

```
┌─────────────┐
│   Feature   │
│ Development │
└──────┬──────┘
       │ PR + Review
       ▼
┌─────────────┐      ┌──────────┐
│   Develop   │ ───► │ Release  │
└─────────────┘      └────┬─────┘
       ▲                  │ PR + Tests
       │                  ▼
       │             ┌─────────┐
       │             │  Main   │
       │             └────┬────┘
       │                  │
       │                  │ Hotfix needed
       │                  ▼
       │             ┌─────────┐
       └─────────────│ Hotfix  │
                     └─────────┘
```

## 🎯 Ejemplos Prácticos

### Escenario 1: Nueva Feature

```bash
# 1. Crear branch
git checkout -b feature/PAY-456-stripe-integration

# 2. Múltiples commits
git commit -m "feat(payment): add Stripe SDK"
git commit -m "feat(payment): implement payment processing"
git commit -m "test(payment): add integration tests"
git commit -m "docs(payment): update API documentation"

# 3. Push y PR
git push origin feature/PAY-456-stripe-integration
# Crear PR en GitHub/GitLab

# 4. Después de aprobación y merge, limpiar
git checkout develop
git pull origin develop
git branch -d feature/PAY-456-stripe-integration
```

### Escenario 2: Release

```bash
# 1. Crear release branch
git checkout -b release/v2.0.0

# 2. Bump versions y changelog
./scripts/bump-version.sh 2.0.0
git commit -m "chore(release): bump version to 2.0.0"

# 3. Deploy a staging para QA
git push origin release/v2.0.0
# CI automáticamente despliega a staging

# 4. Bugs encontrados en QA
git commit -m "fix(api): adjust rate limiting for production scale"

# 5. Merge a main (triggers production deployment)
git checkout main
git merge --no-ff release/v2.0.0
git tag -a v2.0.0 -m "Release 2.0.0"
git push origin main --tags

# 6. Merge back a develop
git checkout develop
git merge --no-ff release/v2.0.0
git push origin develop
```

## 🚨 Emergency Procedures

### Rollback Rápido

```bash
# Opción 1: Revert del último deployment
git checkout main
git revert HEAD
git push origin main

# Opción 2: Deploy del tag anterior
git checkout v1.4.9
git tag -f v1.5.0  # Force update tag
git push -f origin v1.5.0
```

### Cherry-pick de Fix

```bash
# Aplicar un commit específico de develop a release
git checkout release/v1.5.0
git cherry-pick <commit-hash>
git push origin release/v1.5.0
```

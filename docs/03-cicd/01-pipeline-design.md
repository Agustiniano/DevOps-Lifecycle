# CI/CD Pipeline Design - Arquitectura Completa

## 🎯 Filosofía de Pipelines

### Principios

1. **Fail Fast**: Detectar errores lo antes posible
2. **Shift Left**: Seguridad y calidad desde el inicio
3. **Idempotencia**: Mismo input → mismo output
4. **Observabilidad**: Logs, métricas, traces de cada stage
5. **Rollback Capability**: Siempre poder volver atrás

## 🏗️ Arquitectura de Pipelines

```
┌──────────────────────────────────────────────────────────────┐
│                      TRIGGER EVENTS                          │
├──────────────────────────────────────────────────────────────┤
│  • Push to feature/*     → CI Pipeline                       │
│  • PR to develop         → CI + Security                     │
│  • Merge to develop      → CI + Deploy to Dev                │
│  • Tag v*.*.*           → CD to Production                   │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                      CI PIPELINE                             │
├──────────────────────────────────────────────────────────────┤
│  Stage 1: Checkout & Setup                                   │
│    ├─ Clone repository                                       │
│    ├─ Setup build environment                                │
│    └─ Cache dependencies                                     │
│                                                              │
│  Stage 2: Build                                              │
│    ├─ Compile code                                           │
│    ├─ Build Docker image (multi-stage)                       │
│    └─ Tag image with commit SHA                              │
│                                                              │
│  Stage 3: Test                                               │
│    ├─ Unit tests                                             │
│    ├─ Integration tests                                      │
│    ├─ Coverage report (min 80%)                              │
│    └─ Publish test results                                   │
│                                                              │
│  Stage 4: Static Analysis                                    │
│    ├─ SAST (SonarQube, Semgrep)                             │
│    ├─ Linting (golangci-lint, eslint)                       │
│    └─ Code quality gates                                     │
│                                                              │
│  Stage 5: Security Scanning                                  │
│    ├─ SCA - Dependencies (Snyk, Trivy)                      │
│    ├─ Container scan (Trivy, Grype)                         │
│    ├─ Secrets detection (GitLeaks, TruffleHog)              │
│    └─ License compliance                                     │
│                                                              │
│  Stage 6: Publish Artifacts                                  │
│    ├─ Push image to Harbor                                   │
│    ├─ Sign image (Cosign)                                    │
│    ├─ Generate SBOM                                          │
│    └─ Update image tag in GitOps repo                       │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                      CD PIPELINE                             │
├──────────────────────────────────────────────────────────────┤
│  Stage 1: Pre-deployment                                     │
│    ├─ Validate manifests (OPA)                              │
│    ├─ Check cluster health                                   │
│    └─ Backup current state                                   │
│                                                              │
│  Stage 2: Deploy                                             │
│    ├─ Update GitOps repo                                     │
│    ├─ ArgoCD sync                                            │
│    └─ Wait for rollout                                       │
│                                                              │
│  Stage 3: Verification                                       │
│    ├─ Smoke tests                                            │
│    ├─ Health checks                                          │
│    ├─ Performance tests                                      │
│    └─ Validate metrics                                       │
│                                                              │
│  Stage 4: DAST (for production)                              │
│    ├─ OWASP ZAP scan                                         │
│    └─ API security tests                                     │
│                                                              │
│  Stage 5: Post-deployment                                    │
│    ├─ Send notifications (Slack, email)                     │
│    ├─ Update change logs                                     │
│    └─ Trigger monitoring alerts                              │
└──────────────────────────────────────────────────────────────┘
```

## 📋 GitHub Actions: CI Pipeline Completo

### `.github/workflows/ci-microservice.yml`

```yaml
name: CI Pipeline - Microservice

on:
  push:
    branches:
      - develop
      - feature/**
      - bugfix/**
    paths:
      - 'applications/microservices/**'
      - '.github/workflows/ci-microservice.yml'
  pull_request:
    branches:
      - develop
      - main

env:
  REGISTRY: harbor.acme.com
  IMAGE_NAME: ${{ github.repository }}
  GO_VERSION: '1.21'

jobs:
  # ============================================================================
  # Stage 1: Setup & Detect Changes
  # ============================================================================
  setup:
    name: Setup & Detect Changes
    runs-on: ubuntu-latest
    outputs:
      services: ${{ steps.changes.outputs.services }}
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for better analysis

      - name: Detect changed services
        id: changes
        uses: dorny/paths-filter@v2
        with:
          filters: |
            user-service:
              - 'applications/microservices/user-service/**'
            order-service:
              - 'applications/microservices/order-service/**'
            notification-service:
              - 'applications/microservices/notification-service/**'

  # ============================================================================
  # Stage 2: Build & Test
  # ============================================================================
  build-and-test:
    name: Build & Test
    runs-on: ubuntu-latest
    needs: setup
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: applications/microservices/${{ matrix.service }}/go.sum

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles(format('applications/microservices/{0}/go.sum', matrix.service)) }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Download dependencies
        working-directory: applications/microservices/${{ matrix.service }}
        run: go mod download

      - name: Build binary
        working-directory: applications/microservices/${{ matrix.service }}
        run: |
          CGO_ENABLED=0 GOOS=linux go build \
            -ldflags="-w -s -X main.version=${{ github.sha }}" \
            -o bin/api \
            ./cmd/api

      - name: Run unit tests
        working-directory: applications/microservices/${{ matrix.service }}
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Check test coverage
        working-directory: applications/microservices/${{ matrix.service }}
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${coverage}%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "❌ Coverage ${coverage}% is below 80% threshold"
            exit 1
          fi
          echo "✅ Coverage ${coverage}% meets threshold"

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: applications/microservices/${{ matrix.service }}/coverage.out
          flags: ${{ matrix.service }}
          name: ${{ matrix.service }}-coverage

      - name: Run integration tests
        working-directory: applications/microservices/${{ matrix.service }}
        run: |
          go test -v -tags=integration ./tests/integration/...

  # ============================================================================
  # Stage 3: Static Analysis & Linting
  # ============================================================================
  static-analysis:
    name: Static Analysis
    runs-on: ubuntu-latest
    needs: setup
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for SonarQube

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          working-directory: applications/microservices/${{ matrix.service }}
          args: --timeout=5m

      - name: Run gosec (Security Scanner)
        uses: securego/gosec@master
        with:
          args: '-fmt json -out gosec-results.json ./...'
        working-directory: applications/microservices/${{ matrix.service }}

      - name: SonarQube Scan
        uses: sonarsource/sonarqube-scan-action@master
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}
        with:
          projectBaseDir: applications/microservices/${{ matrix.service }}
          args: >
            -Dsonar.projectKey=${{ matrix.service }}
            -Dsonar.sources=.
            -Dsonar.tests=tests
            -Dsonar.go.coverage.reportPaths=coverage.out

      - name: SonarQube Quality Gate
        uses: sonarsource/sonarqube-quality-gate-action@master
        timeout-minutes: 5
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}

  # ============================================================================
  # Stage 4: Security Scanning
  # ============================================================================
  security-scan:
    name: Security Scanning
    runs-on: ubuntu-latest
    needs: setup
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Run Semgrep (SAST)
        uses: returntocorp/semgrep-action@v1
        with:
          config: >-
            p/security-audit
            p/secrets
            p/owasp-top-ten
          working-directory: applications/microservices/${{ matrix.service }}

      - name: Dependency vulnerability scan (Snyk)
        uses: snyk/actions/golang@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          command: test
          args: --severity-threshold=high --file=applications/microservices/${{ matrix.service }}/go.mod

      - name: Trivy FS scan (dependencies)
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: 'applications/microservices/${{ matrix.service }}'
          format: 'sarif'
          output: 'trivy-fs-results.sarif'
          severity: 'CRITICAL,HIGH'

      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-fs-results.sarif'
          category: trivy-fs-${{ matrix.service }}

      - name: Detect secrets (GitLeaks)
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  # ============================================================================
  # Stage 5: Build & Push Docker Image
  # ============================================================================
  build-image:
    name: Build & Push Docker Image
    runs-on: ubuntu-latest
    needs: [build-and-test, static-analysis, security-scan]
    if: github.event_name == 'push' && github.ref == 'refs/heads/develop'
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    permissions:
      contents: read
      packages: write
      id-token: write  # For Cosign
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Harbor
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.HARBOR_USERNAME }}
          password: ${{ secrets.HARBOR_PASSWORD }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/platform/${{ matrix.service }}
          tags: |
            type=ref,event=branch
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push image
        id: build-push
        uses: docker/build-push-action@v5
        with:
          context: applications/microservices/${{ matrix.service }}
          file: applications/microservices/${{ matrix.service }}/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.sha }}
            BUILD_DATE=${{ github.event.head_commit.timestamp }}

      - name: Install Cosign
        uses: sigstore/cosign-installer@v3

      - name: Sign image with Cosign
        run: |
          cosign sign --yes \
            ${{ env.REGISTRY }}/platform/${{ matrix.service }}@${{ steps.build-push.outputs.digest }}

      - name: Generate SBOM
        uses: anchore/sbom-action@v0
        with:
          image: ${{ env.REGISTRY }}/platform/${{ matrix.service }}:${{ github.sha }}
          format: cyclonedx-json
          output-file: sbom-${{ matrix.service }}.json

      - name: Upload SBOM
        uses: actions/upload-artifact@v3
        with:
          name: sbom-${{ matrix.service }}
          path: sbom-${{ matrix.service }}.json

  # ============================================================================
  # Stage 6: Container Security Scan
  # ============================================================================
  scan-image:
    name: Scan Docker Image
    runs-on: ubuntu-latest
    needs: build-image
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    steps:
      - name: Login to Harbor
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.HARBOR_USERNAME }}
          password: ${{ secrets.HARBOR_PASSWORD }}

      - name: Trivy image scan
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ env.REGISTRY }}/platform/${{ matrix.service }}:${{ github.sha }}
          format: 'sarif'
          output: 'trivy-image-results.sarif'
          severity: 'CRITICAL,HIGH'
          exit-code: '1'  # Fail on vulnerabilities

      - name: Upload Trivy results
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: 'trivy-image-results.sarif'
          category: trivy-image-${{ matrix.service }}

      - name: Grype vulnerability scan
        uses: anchore/scan-action@v3
        with:
          image: ${{ env.REGISTRY }}/platform/${{ matrix.service }}:${{ github.sha }}
          fail-build: true
          severity-cutoff: high

  # ============================================================================
  # Stage 7: Update GitOps Repository
  # ============================================================================
  update-gitops:
    name: Update GitOps Manifests
    runs-on: ubuntu-latest
    needs: scan-image
    if: github.ref == 'refs/heads/develop'
    strategy:
      matrix:
        service: [user-service, order-service, notification-service]
    steps:
      - name: Checkout GitOps repo
        uses: actions/checkout@v4
        with:
          repository: acme/gitops-manifests
          token: ${{ secrets.GITOPS_PAT }}
          path: gitops

      - name: Update image tag
        run: |
          cd gitops/environments/dev/${{ matrix.service }}
          
          # Using kustomize to update image
          kustomize edit set image \
            ${{ matrix.service }}=${{ env.REGISTRY }}/platform/${{ matrix.service }}:${{ github.sha }}

      - name: Commit and push changes
        run: |
          cd gitops
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add .
          git commit -m "chore: update ${{ matrix.service }} to ${{ github.sha }}"
          git push

  # ============================================================================
  # Notifications
  # ============================================================================
  notify:
    name: Send Notifications
    runs-on: ubuntu-latest
    needs: [update-gitops]
    if: always()
    steps:
      - name: Slack notification
        uses: 8398a7/action-slack@v3
        with:
          status: ${{ job.status }}
          text: |
            CI Pipeline: ${{ job.status }}
            Repository: ${{ github.repository }}
            Branch: ${{ github.ref }}
            Commit: ${{ github.sha }}
          webhook_url: ${{ secrets.SLACK_WEBHOOK }}
        if: always()
```

Continúo con más ejemplos de pipelines...

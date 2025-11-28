# DevSecOps - Shift-Left Security

## 🎯 Filosofía Shift-Left

**Integrar seguridad desde el inicio del ciclo de desarrollo**, no como una etapa final.

```
Traditional Security:
Code → Build → Test → [Security] → Deploy
                        ↑
                   Bottleneck

Shift-Left Security:
[Sec] Code → [Sec] Build → [Sec] Test → [Sec] Deploy → [Sec] Runtime
   ↓           ↓              ↓             ↓              ↓
 IDE Scan   SAST/SCA     Container     DAST/Pen    Runtime
 Secrets    Dependencies   Scan       Testing      Protection
```

## 🔒 Security Layers

### 1. Pre-Commit (IDE/Local)

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks

  - repo: https://github.com/Yelp/detect-secrets
    rev: v1.4.0
    hooks:
      - id: detect-secrets
        args: ['--baseline', '.secrets.baseline']

  - repo: https://github.com/returntocorp/semgrep
    rev: 'v1.45.0'
    hooks:
      - id: semgrep
        args:
          - --config=p/security-audit
          - --config=p/secrets
          - --error

  - repo: https://github.com/trufflesecurity/trufflehog
    rev: v3.63.0
    hooks:
      - id: trufflehog
        args:
          - --no-verification
          - --exclude-paths=.truffleHogignore
```

```bash
# Instalación
pip install pre-commit
pre-commit install

# Test manual
pre-commit run --all-files
```

### 2. SAST (Static Application Security Testing)

#### SonarQube Configuration

```properties
# sonar-project.properties
sonar.projectKey=user-service
sonar.projectName=User Service
sonar.projectVersion=1.0

# Source
sonar.sources=.
sonar.tests=tests
sonar.exclusions=**/*_test.go,**/vendor/**,**/mocks/**

# Coverage
sonar.go.coverage.reportPaths=coverage.out
sonar.go.tests.reportPaths=test-report.json

# Security
sonar.security.hotspots.inheritanceEnabled=true

# Quality Gates
sonar.qualitygate.wait=true
sonar.qualitygate.timeout=300
```

#### Semgrep Rules

```yaml
# security/scanning/semgrep-rules.yaml
rules:
  - id: go-sql-injection
    pattern: |
      db.Query($QUERY)
    message: Potential SQL injection
    languages: [go]
    severity: ERROR
    
  - id: go-hardcoded-credentials
    patterns:
      - pattern-either:
          - pattern: |
              password = "..."
          - pattern: |
              apiKey = "..."
          - pattern: |
              token = "..."
    message: Hardcoded credentials detected
    languages: [go]
    severity: ERROR

  - id: go-weak-crypto
    pattern-either:
      - pattern: crypto.MD5.New()
      - pattern: crypto.SHA1.New()
      - pattern: des.NewCipher(...)
    message: Weak cryptographic algorithm
    languages: [go]
    severity: WARNING

  - id: go-insecure-tls
    pattern: |
      tls.Config{InsecureSkipVerify: true}
    message: TLS verification disabled
    languages: [go]
    severity: ERROR

  - id: go-command-injection
    pattern: exec.Command($CMD, ...)
    message: Potential command injection
    languages: [go]
    severity: WARNING
```

```bash
# Ejecutar Semgrep
semgrep --config security/scanning/semgrep-rules.yaml \
  --config p/security-audit \
  --config p/owasp-top-ten \
  --json -o semgrep-results.json \
  applications/microservices/user-service/
```

### 3. SCA (Software Composition Analysis)

#### Snyk Integration

```yaml
# .github/workflows/security-scan.yml (fragmento)
- name: Run Snyk to check for vulnerabilities
  uses: snyk/actions/golang@master
  env:
    SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
  with:
    command: test
    args: >
      --severity-threshold=high
      --fail-on=upgradable
      --file=go.mod
      --json-file-output=snyk-results.json

- name: Run Snyk to check Docker image
  uses: snyk/actions/docker@master
  env:
    SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
  with:
    image: harbor.acme.com/platform/user-service:${{ github.sha }}
    args: --severity-threshold=high
```

#### Trivy for Dependencies

```bash
# Scan filesystem (dependencies)
trivy fs \
  --severity CRITICAL,HIGH \
  --format json \
  --output trivy-fs-results.json \
  --exit-code 1 \
  applications/microservices/user-service/

# Scan with SBOM generation
trivy fs \
  --format cyclonedx \
  --output sbom.json \
  applications/microservices/user-service/
```

### 4. Container Security

#### Multi-stage Dockerfile (Secure)

```dockerfile
# applications/microservices/user-service/Dockerfile

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Security: Run as non-root
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -trimpath \
    -o /app/bin/api \
    ./cmd/api

# Stage 2: Runtime
FROM scratch

# Copy SSL certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user information
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy binary
COPY --from=builder /app/bin/api /api

# Use non-root user
USER appuser:appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/api", "healthcheck"]

# Run
ENTRYPOINT ["/api"]
```

#### Trivy Image Scan

```bash
# Comprehensive scan
trivy image \
  --severity CRITICAL,HIGH,MEDIUM \
  --format table \
  --exit-code 1 \
  --ignore-unfixed \
  harbor.acme.com/platform/user-service:latest

# With secret detection
trivy image \
  --scanners vuln,secret,config \
  harbor.acme.com/platform/user-service:latest
```

#### Grype Alternative

```bash
grype harbor.acme.com/platform/user-service:latest \
  --fail-on high \
  --output json \
  --file grype-results.json
```

### 5. IaC Security Scanning

#### Checkov for Terraform

```bash
# Scan all Terraform
checkov -d infrastructure/terraform/ \
  --framework terraform \
  --output json \
  --output-file-path checkov-results.json

# With specific checks
checkov -d infrastructure/terraform/ \
  --check CKV_AWS_20,CKV_AWS_21,CKV_AWS_23 \
  --framework terraform
```

#### TFSec

```bash
tfsec infrastructure/terraform/ \
  --format json \
  --out tfsec-results.json \
  --minimum-severity HIGH

# With custom checks
tfsec infrastructure/terraform/ \
  --config-file .tfsec.yml
```

```yaml
# .tfsec.yml
minimum_severity: HIGH

exclude:
  - AWS001  # S3 bucket encryption (handled by module)

severity_overrides:
  AWS002: ERROR  # S3 bucket logging
```

#### Terrascan

```bash
terrascan scan -t aws \
  -d infrastructure/terraform/environments/production/ \
  -o json \
  > terrascan-results.json
```

### 6. Kubernetes Security

#### OPA Policies (Rego)

```rego
# security/policies/kubernetes/require-resource-limits.rego
package kubernetes.admission

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  not container.resources.limits.memory
  msg := sprintf("Container '%s' must specify memory limits", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  not container.resources.limits.cpu
  msg := sprintf("Container '%s' must specify CPU limits", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  not container.resources.requests.memory
  msg := sprintf("Container '%s' must specify memory requests", [container.name])
}
```

```rego
# security/policies/kubernetes/deny-privileged-containers.rego
package kubernetes.admission

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  container.securityContext.privileged == true
  msg := sprintf("Privileged container '%s' is not allowed", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  container.securityContext.runAsUser == 0
  msg := sprintf("Container '%s' cannot run as root (UID 0)", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  not input.request.object.spec.securityContext.runAsNonRoot
  msg := "Pod must set 'runAsNonRoot: true'"
}
```

```rego
# security/policies/kubernetes/enforce-security-context.rego
package kubernetes.admission

deny[msg] {
  input.request.kind.kind == "Pod"
  not input.request.object.spec.securityContext.runAsNonRoot
  msg := "Pod must set runAsNonRoot to true"
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  not container.securityContext.allowPrivilegeEscalation == false
  msg := sprintf("Container '%s' must set allowPrivilegeEscalation to false", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  not container.securityContext.readOnlyRootFilesystem
  msg := sprintf("Container '%s' should use read-only root filesystem", [container.name])
}

deny[msg] {
  input.request.kind.kind == "Pod"
  container := input.request.object.spec.containers[_]
  container.image
  endswith(container.image, ":latest")
  msg := sprintf("Container '%s' uses ':latest' tag, which is not allowed", [container.name])
}
```

#### Conftest with OPA

```bash
# Test Kubernetes manifests
conftest test gitops/environments/production/user-service/*.yaml \
  --policy security/policies/kubernetes/ \
  --namespace kubernetes.admission

# Test Terraform
conftest test infrastructure/terraform/modules/eks/*.tf \
  --policy security/policies/terraform/
```

#### Kubesec

```bash
# Scan Kubernetes YAML
kubesec scan gitops/environments/production/user-service/deployment.yaml

# With threshold
kubesec scan gitops/environments/production/user-service/deployment.yaml \
  --fail-threshold 5
```

### 7. DAST (Dynamic Application Security Testing)

#### OWASP ZAP

```yaml
# .github/workflows/dast-scan.yml
name: DAST Scan

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  zap-scan:
    runs-on: ubuntu-latest
    steps:
      - name: ZAP Baseline Scan
        uses: zaproxy/action-baseline@v0.7.0
        with:
          target: 'https://api-staging.acme.com'
          rules_file_name: '.zap/rules.tsv'
          cmd_options: '-a'

      - name: ZAP Full Scan
        uses: zaproxy/action-full-scan@v0.7.0
        with:
          target: 'https://api-staging.acme.com'
          rules_file_name: '.zap/rules.tsv'
          cmd_options: '-a -j'

      - name: Upload ZAP results
        uses: actions/upload-artifact@v3
        with:
          name: zap-results
          path: |
            report_html.html
            report_json.json
```

#### Nuclei

```bash
# Install nuclei
go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

# Run scan
nuclei -u https://api-staging.acme.com \
  -t cves/ \
  -t vulnerabilities/ \
  -t exposures/ \
  -o nuclei-results.txt

# With specific templates
nuclei -u https://api-staging.acme.com \
  -t http/cves/ \
  -t http/exposures/ \
  -severity critical,high
```

### 8. Runtime Security

#### Falco Rules

```yaml
# security/falco/rules/custom-rules.yaml
- rule: Unauthorized Process in Container
  desc: Detect processes not in allowed list
  condition: >
    spawned_process and
    container and
    not proc.name in (allowed_processes)
  output: >
    Unauthorized process started in container
    (user=%user.name process=%proc.name
    container=%container.name image=%container.image.repository)
  priority: WARNING
  tags: [process, container]

- macro: allowed_processes
  condition: proc.name in (api, sh, bash, ps, grep)

- rule: Write to Non-Writable Directory
  desc: Detect writes to directories that should be read-only
  condition: >
    open_write and
    container and
    fd.name startswith /etc/
  output: >
    Write to protected directory
    (user=%user.name file=%fd.name
    container=%container.name)
  priority: ERROR
  tags: [filesystem]

- rule: Outbound Connection to Suspicious IP
  desc: Detect connections to known malicious IPs
  condition: >
    outbound and
    fd.sip in (suspicious_ips)
  output: >
    Suspicious outbound connection
    (user=%user.name ip=%fd.sip port=%fd.sport
    container=%container.name)
  priority: CRITICAL
  tags: [network]

- list: suspicious_ips
  items: [192.0.2.0, 198.51.100.0]
```

#### Install Falco

```bash
# Kubernetes deployment
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update

helm install falco falcosecurity/falco \
  --namespace falco \
  --create-namespace \
  --set falco.grpc.enabled=true \
  --set falco.grpcOutput.enabled=true \
  -f security/falco/values.yaml
```

### 9. Secrets Management

#### HashiCorp Vault

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure
vault write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc:443" \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token

# Create policy
vault policy write user-service - <<EOF
path "secret/data/user-service/*" {
  capabilities = ["read"]
}
EOF

# Create role
vault write auth/kubernetes/role/user-service \
  bound_service_account_names=user-service \
  bound_service_account_namespaces=production \
  policies=user-service \
  ttl=24h
```

#### Vault Sidecar Injector

```yaml
# Deployment with Vault annotations
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    metadata:
      annotations:
        vault.hashicorp.com/agent-inject: "true"
        vault.hashicorp.com/role: "user-service"
        vault.hashicorp.com/agent-inject-secret-database: "secret/data/user-service/database"
        vault.hashicorp.com/agent-inject-template-database: |
          {{- with secret "secret/data/user-service/database" -}}
          export DB_HOST="{{ .Data.data.host }}"
          export DB_USER="{{ .Data.data.username }}"
          export DB_PASS="{{ .Data.data.password }}"
          {{- end }}
    spec:
      serviceAccountName: user-service
      containers:
      - name: api
        image: harbor.acme.com/platform/user-service:v1.0.0
```

#### Sealed Secrets

```bash
# Install sealed-secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# Create sealed secret
kubectl create secret generic user-service-db \
  --from-literal=password='SuperSecret123!' \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

# Apply sealed secret
kubectl apply -f sealed-secret.yaml
```

## 📊 Security Dashboard

```yaml
# observability/grafana/dashboards/security-overview.json
{
  "dashboard": {
    "title": "Security Overview",
    "panels": [
      {
        "title": "Vulnerability Trends",
        "targets": [
          {
            "expr": "sum(vulnerability_count) by (severity)"
          }
        ]
      },
      {
        "title": "Failed Security Scans",
        "targets": [
          {
            "expr": "rate(security_scan_failures_total[1h])"
          }
        ]
      },
      {
        "title": "Falco Alerts",
        "targets": [
          {
            "expr": "sum(rate(falco_events_total[5m])) by (priority)"
          }
        ]
      }
    ]
  }
}
```

## 🚨 Security Incident Response

```bash
#!/bin/bash
# scripts/security-incident-response.sh

# 1. Isolate affected pod
kubectl label pod $AFFECTED_POD quarantine=true

# 2. Network policy to block traffic
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: quarantine-$AFFECTED_POD
spec:
  podSelector:
    matchLabels:
      quarantine: "true"
  policyTypes:
  - Ingress
  - Egress
EOF

# 3. Capture forensics
kubectl exec $AFFECTED_POD -- netstat -an > netstat.log
kubectl exec $AFFECTED_POD -- ps aux > processes.log
kubectl logs $AFFECTED_POD > pod.log

# 4. Terminate pod
kubectl delete pod $AFFECTED_POD
```

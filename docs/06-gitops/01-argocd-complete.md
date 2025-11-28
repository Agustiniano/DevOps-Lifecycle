# GitOps with ArgoCD - Complete Guide

## 🎯 GitOps Principles

```
Git Repository (Source of Truth)
        │
        ▼
   [ArgoCD Sync]
        │
        ▼
Kubernetes Cluster (Desired State)
        │
        ▼
   [Drift Detection]
        │
        ▼
  [Auto/Manual Sync]
```

### Core Principles

1. **Declarative**: Everything defined as code
2. **Versioned**: Git as single source of truth
3. **Immutable**: No manual kubectl apply
4. **Automated**: Continuous reconciliation
5. **Auditable**: Complete change history

## 📦 ArgoCD Installation

### 1. Install ArgoCD

```bash
# Create namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for rollout
kubectl rollout status deployment argocd-server -n argocd
```

### 2. Expose ArgoCD UI

```yaml
# gitops/argocd/bootstrap/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server
  namespace: argocd
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
spec:
  ingressClassName: nginx
  rules:
    - host: argocd.acme.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: argocd-server
                port:
                  name: https
  tls:
    - hosts:
        - argocd.acme.com
      secretName: argocd-tls
```

### 3. Get Initial Password

```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d
```

### 4. ArgoCD CLI

```bash
# Install CLI
brew install argocd

# Login
argocd login argocd.acme.com

# Change password
argocd account update-password
```

## 🏗️ Repository Structure

```
gitops/
├── argocd/
│   ├── bootstrap/
│   │   ├── app-of-apps.yaml          # Root application
│   │   └── argocd-projects.yaml      # Projects definition
│   │
│   └── apps/                          # Application definitions
│       ├── platform/
│       │   ├── monitoring.yaml
│       │   ├── ingress-nginx.yaml
│       │   └── cert-manager.yaml
│       │
│       └── applications/
│           ├── user-service.yaml
│           ├── order-service.yaml
│           └── notification-service.yaml
│
└── environments/
    ├── base/                          # Base configurations
    │   ├── user-service/
    │   │   ├── deployment.yaml
    │   │   ├── service.yaml
    │   │   ├── hpa.yaml
    │   │   └── kustomization.yaml
    │   │
    │   └── order-service/
    │       └── ...
    │
    └── overlays/                      # Environment-specific
        ├── dev/
        │   ├── kustomization.yaml
        │   └── patches/
        │
        ├── staging/
        │   ├── kustomization.yaml
        │   └── patches/
        │
        └── production/
            ├── kustomization.yaml
            ├── patches/
            └── sealed-secrets/
```

## 📱 App of Apps Pattern

### Root Application

```yaml
# gitops/argocd/bootstrap/app-of-apps.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: root
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  
  source:
    repoURL: https://github.com/acme/gitops-manifests.git
    targetRevision: main
    path: argocd/apps
  
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
      allowEmpty: false
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

### Application Definition

```yaml
# gitops/argocd/apps/applications/user-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: user-service-prod
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: applications
  
  source:
    repoURL: https://github.com/acme/gitops-manifests.git
    targetRevision: main
    path: environments/overlays/production/user-service
  
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - PruneLast=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
  
  # Health assessment
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas  # Ignore HPA-managed replicas
  
  # Sync waves for ordered deployment
  syncPolicy:
    syncOptions:
      - Prune=true
```

## 🎛️ ArgoCD Projects

```yaml
# gitops/argocd/bootstrap/argocd-projects.yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: platform
  namespace: argocd
spec:
  description: Platform infrastructure components
  
  sourceRepos:
    - https://github.com/acme/gitops-manifests.git
    - https://prometheus-community.github.io/helm-charts
    - https://grafana.github.io/helm-charts
  
  destinations:
    - namespace: 'monitoring'
      server: https://kubernetes.default.svc
    - namespace: 'ingress-nginx'
      server: https://kubernetes.default.svc
    - namespace: 'cert-manager'
      server: https://kubernetes.default.svc
  
  clusterResourceWhitelist:
    - group: '*'
      kind: '*'
  
  namespaceResourceWhitelist:
    - group: '*'
      kind: '*'

---
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: applications
  namespace: argocd
spec:
  description: Application microservices
  
  sourceRepos:
    - https://github.com/acme/gitops-manifests.git
  
  destinations:
    - namespace: 'dev'
      server: https://kubernetes.default.svc
    - namespace: 'staging'
      server: https://kubernetes.default.svc
    - namespace: 'production'
      server: https://kubernetes.default.svc
  
  # Restrict resource types
  clusterResourceWhitelist: []
  
  namespaceResourceWhitelist:
    - group: 'apps'
      kind: 'Deployment'
    - group: 'apps'
      kind: 'StatefulSet'
    - group: ''
      kind: 'Service'
    - group: ''
      kind: 'ConfigMap'
    - group: ''
      kind: 'Secret'
    - group: 'networking.k8s.io'
      kind: 'Ingress'
    - group: 'autoscaling'
      kind: 'HorizontalPodAutoscaler'
  
  # Orphaned resources handling
  orphanedResources:
    warn: true
```

## 🔧 Kustomize Structure

### Base Configuration

```yaml
# gitops/environments/base/user-service/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: production

resources:
  - deployment.yaml
  - service.yaml
  - hpa.yaml
  - servicemonitor.yaml

commonLabels:
  app: user-service
  managed-by: argocd

configMapGenerator:
  - name: user-service-config
    literals:
      - LOG_LEVEL=info
      - PORT=8080

images:
  - name: user-service
    newName: harbor.acme.com/platform/user-service
    newTag: v1.0.0
```

### Production Overlay

```yaml
# gitops/environments/overlays/production/user-service/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: production

bases:
  - ../../../base/user-service

patches:
  - path: patches/deployment-resources.yaml
  - path: patches/replicas.yaml
  - path: patches/environment.yaml

configMapGenerator:
  - name: user-service-config
    behavior: merge
    literals:
      - LOG_LEVEL=info
      - ENVIRONMENT=production
      - MAX_CONNECTIONS=100

secretGenerator:
  - name: user-service-secrets
    files:
      - secrets/database-url
      - secrets/api-key

images:
  - name: user-service
    newName: harbor.acme.com/platform/user-service
    newTag: v1.2.3  # Updated by CI/CD
```

### Patches

```yaml
# gitops/environments/overlays/production/user-service/patches/deployment-resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              memory: "512Mi"
              cpu: "500m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
```

```yaml
# gitops/environments/overlays/production/user-service/patches/replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  replicas: 5
```

## 🔄 Sync Policies

### Automatic Sync

```yaml
syncPolicy:
  automated:
    prune: true      # Delete resources not in Git
    selfHeal: true   # Force sync when drift detected
    allowEmpty: false # Prevent accidental deletion
```

### Manual Sync with Hooks

```yaml
# Pre-sync hook: Database migration
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: migrate/migrate
          args:
            - "-path=/migrations"
            - "-database=$(DB_URL)"
            - "up"
      restartPolicy: Never
```

### Sync Waves

```yaml
# Deploy in order using sync waves
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  annotations:
    argocd.argoproj.io/sync-wave: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  annotations:
    argocd.argoproj.io/sync-wave: "2"
---
apiVersion: v1
kind: Service
metadata:
  name: service
  annotations:
    argocd.argoproj.io/sync-wave: "3"
```

## 🚨 Health Assessment

### Custom Health Check

```lua
# argocd-cm ConfigMap
resource.customizations: |
  argoproj.io/Rollout:
    health.lua: |
      hs = {}
      if obj.status ~= nil then
        if obj.status.phase == "Healthy" then
          hs.status = "Healthy"
          hs.message = obj.status.message
          return hs
        end
        if obj.status.phase == "Progressing" then
          hs.status = "Progressing"
          hs.message = obj.status.message
          return hs
        end
      end
      hs.status = "Degraded"
      hs.message = "Rollout is degraded"
      return hs
```

## 🔐 RBAC Configuration

```yaml
# argocd-rbac-cm ConfigMap
policy.csv: |
  # Developers can view and sync applications
  p, role:developer, applications, get, */*, allow
  p, role:developer, applications, sync, */*, allow
  g, developers, role:developer

  # Platform team has full access
  p, role:platform, *, *, */*, allow
  g, platform-team, role:platform

  # Read-only for everyone else
  p, role:readonly, applications, get, */*, allow
  p, role:readonly, projects, get, *, allow
  g, authenticated, role:readonly

policy.default: role:readonly
```

## 🔔 Notifications

### Slack Notifications

```yaml
# argocd-notifications-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  service.slack: |
    token: $slack-token
  
  template.app-deployed: |
    message: |
      Application {{.app.metadata.name}} is now running new version.
      {{if eq .serviceType "slack"}}:white_check_mark:{{end}}
    slack:
      attachments: |
        [{
          "title": "{{ .app.metadata.name}}",
          "title_link":"{{.context.argocdUrl}}/applications/{{.app.metadata.name}}",
          "color": "#18be52",
          "fields": [
            {
              "title": "Sync Status",
              "value": "{{.app.status.sync.status}}",
              "short": true
            },
            {
              "title": "Repository",
              "value": "{{.app.spec.source.repoURL}}",
              "short": true
            }
          ]
        }]
  
  template.app-health-degraded: |
    message: |
      Application {{.app.metadata.name}} has degraded health.
      {{if eq .serviceType "slack"}}:exclamation:{{end}}
    slack:
      attachments: |
        [{
          "title": "{{ .app.metadata.name}}",
          "title_link": "{{.context.argocdUrl}}/applications/{{.app.metadata.name}}",
          "color": "#f4c030",
          "fields": [
            {
              "title": "Health Status",
              "value": "{{.app.status.health.status}}",
              "short": true
            }
          ]
        }]
  
  trigger.on-deployed: |
    - when: app.status.operationState.phase in ['Succeeded']
      send: [app-deployed]
  
  trigger.on-health-degraded: |
    - when: app.status.health.status == 'Degraded'
      send: [app-health-degraded]
```

### Subscribe Applications

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: user-service
  annotations:
    notifications.argoproj.io/subscribe.on-deployed.slack: deployments-channel
    notifications.argoproj.io/subscribe.on-health-degraded.slack: alerts-channel
```

## 🔄 Progressive Delivery with Argo Rollouts

```yaml
# gitops/environments/base/user-service/rollout.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: user-service
spec:
  replicas: 5
  revisionHistoryLimit: 3
  
  selector:
    matchLabels:
      app: user-service
  
  template:
    metadata:
      labels:
        app: user-service
    spec:
      containers:
        - name: api
          image: harbor.acme.com/platform/user-service:v1.0.0
          ports:
            - containerPort: 8080
  
  strategy:
    canary:
      canaryService: user-service-canary
      stableService: user-service
      
      steps:
        - setWeight: 20
        - pause: {duration: 5m}
        
        - setWeight: 40
        - pause: {duration: 5m}
        
        - setWeight: 60
        - pause: {duration: 5m}
        
        - setWeight: 80
        - pause: {duration: 5m}
      
      trafficRouting:
        istio:
          virtualService:
            name: user-service
            routes:
              - primary
      
      analysis:
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: user-service
```

## 📊 CLI Commands

```bash
# List applications
argocd app list

# Get application details
argocd app get user-service-prod

# Sync application
argocd app sync user-service-prod

# Sync with prune
argocd app sync user-service-prod --prune

# Rollback
argocd app rollback user-service-prod 5

# Set image
argocd app set user-service-prod \
  --kustomize-image harbor.acme.com/platform/user-service:v1.2.4

# Diff
argocd app diff user-service-prod

# Application logs
argocd app logs user-service-prod

# Delete application
argocd app delete user-service-prod --cascade
```

## 🎯 Best Practices

1. **Single Source of Truth**: Git repository es la única fuente
2. **Separate Repos**: App code y GitOps manifests en repos diferentes
3. **Environment Branches**: `main` para prod, `staging` para staging
4. **Automated Sync**: Habilitar auto-sync con self-heal
5. **Sync Waves**: Usar para dependencias entre recursos
6. **Health Checks**: Definir checks custom cuando sea necesario
7. **RBAC**: Implementar least privilege
8. **Notifications**: Configurar para eventos importantes
9. **Backup**: ArgoCD config y application definitions
10. **Disaster Recovery**: Plan para recrear cluster desde Git

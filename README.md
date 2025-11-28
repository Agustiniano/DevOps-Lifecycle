# DevOps Lifecycle - Framework Completo de Implementación

## 🎯 Visión General

Este repositorio implementa un **Ciclo de Vida DevOps completo de punta a punta**, modelando prácticas reales de ingeniería de plataforma en organizaciones enterprise. Incluye arquitectura, IaC, CI/CD, seguridad, observabilidad, GitOps y gobernanza.

## 📋 Tabla de Contenidos

1. [Arquitectura del Ciclo de Vida](#arquitectura)
2. [Estructura del Repositorio](#estructura)
3. [Infraestructura como Código (IaC)](#iac)
4. [Pipelines CI/CD](#pipelines)
5. [DevSecOps](#devsecops)
6. [Observabilidad](#observabilidad)
7. [GitOps](#gitops)
8. [Gobernanza](#gobernanza)
9. [Gestión del Cambio](#cambio)
10. [Quick Start](#quick-start)

## 🏗️ Arquitectura del Ciclo de Vida {#arquitectura}

### Flujo End-to-End

```
Developer → Git Push → CI Pipeline → Security Scans → Build & Test → 
Container Registry → GitOps Sync → K8s Deployment → Observability → 
Feedback Loop → Developer
```

### Componentes Principales

- **Source Control**: Git con estrategia GitFlow modificada
- **CI/CD**: GitHub Actions / GitLab CI / Jenkins
- **IaC**: Terraform (infraestructura) + Crossplane (cloud resources) + Ansible (configuración)
- **Container Registry**: Harbor con escaneo de vulnerabilidades
- **Orchestration**: Kubernetes (EKS, GKE, AKS)
- **GitOps**: ArgoCD con ApplicationSets
- **Observability**: Prometheus + Grafana + Loki + Tempo + OpenTelemetry
- **Security**: Trivy, Snyk, OPA, Falco, Vault
- **Service Mesh**: Istio (opcional pero recomendado)

### Entornos

```
dev → staging → production
  ↓      ↓          ↓
  EKS   EKS        EKS (multi-region)
```

## 📁 Estructura del Repositorio {#estructura}

Ver `docs/01-architecture/repository-structure.md` para detalles completos.

## 🚀 Quick Start {#quick-start}

```bash
# 1. Clonar el repositorio
git clone <repo-url>
cd devops-lifecycle

# 2. Configurar credenciales
cp .env.example .env
# Editar .env con tus credenciales

# 3. Inicializar infraestructura base
cd infrastructure/terraform/bootstrap
terraform init
terraform plan
terraform apply

# 4. Configurar GitOps
cd ../../../gitops/argocd
kubectl apply -f bootstrap/

# 5. Desplegar aplicación de ejemplo
cd ../../applications/microservices/user-service
make deploy-dev
```

## 📚 Documentación Completa

Toda la documentación detallada está en `/docs`:

- **Arquitectura**: `/docs/01-architecture/`
- **IaC**: `/docs/02-iac/`
- **CI/CD**: `/docs/03-cicd/`
- **Seguridad**: `/docs/04-security/`
- **Observabilidad**: `/docs/05-observability/`
- **GitOps**: `/docs/06-gitops/`
- **Gobernanza**: `/docs/07-governance/`
- **Gestión del Cambio**: `/docs/08-change-management/`
- **Runbooks**: `/docs/09-runbooks/`

## 🎓 Escenarios Incluidos

1. **Microservicio básico**: Go API + PostgreSQL
2. **Pipeline multi-stage**: Build → Test → Scan → Deploy
3. **Blue/Green Deployment**: Con verificación automática
4. **Canary Release**: Traffic shifting progresivo con Istio
5. **Rollback automático**: Basado en métricas de observabilidad
6. **Multi-cloud**: AWS (primario) + GCP (DR)
7. **Disaster Recovery**: Backup, restore y failover

## 🔐 Seguridad

- **Secrets Management**: HashiCorp Vault
- **RBAC**: Kubernetes + ArgoCD + Cloud IAM
- **Network Policies**: Calico
- **Mutual TLS**: Istio service mesh
- **Image Signing**: Cosign + Sigstore

## 📊 Métricas y SLOs

- **Deployment Frequency**: < 1 día
- **Lead Time for Changes**: < 4 horas
- **MTTR**: < 30 minutos
- **Change Failure Rate**: < 15%

## 🤝 Contributing

Ver `CONTRIBUTING.md` para guías de contribución.

## 📄 Licencia

MIT License - Ver `LICENSE` para detalles.

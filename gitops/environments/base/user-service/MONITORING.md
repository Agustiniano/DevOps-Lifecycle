# User Service - Monitoring Setup

## ServiceMonitor (Prometheus Operator)

El archivo `servicemonitor.yaml` está **deshabilitado por defecto** porque requiere que Prometheus Operator esté instalado en el cluster.

### Error sin Prometheus Operator

Si ves este error en ArgoCD:

```
The Kubernetes API could not find monitoring.coreos.com/ServiceMonitor for requested resource production/user-service. 
Make sure the "ServiceMonitor" CRD is installed on the destination cluster.
```

**Causa:** El CRD `ServiceMonitor` no existe en el cluster porque Prometheus Operator no está instalado.

### Solución 1: Instalar Prometheus Operator

```bash
# Usando Helm
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Verificar que los CRDs estén instalados
kubectl get crd | grep monitoring.coreos.com
```

Luego habilita el ServiceMonitor:

```yaml
# En gitops/environments/base/user-service/kustomization.yaml
resources:
  - deployment.yaml
  - servicemonitor.yaml  # ← Descomentar esta línea
```

### Solución 2: Usar anotaciones de Prometheus (sin Operator)

Si usas Prometheus sin el Operator, las **anotaciones ya están configuradas** en el Deployment:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

Prometheus nativo usará estas anotaciones para descubrir y scrapear métricas automáticamente.

### Endpoints de Monitoring

- **Métricas**: `http://user-service:8080/metrics`
- **Health**: `http://user-service:8080/health`
- **Readiness**: `http://user-service:8080/ready`

### Verificar métricas manualmente

```bash
# Port-forward al servicio
kubectl port-forward svc/user-service -n production 8080:80

# Obtener métricas
curl http://localhost:8080/metrics
```

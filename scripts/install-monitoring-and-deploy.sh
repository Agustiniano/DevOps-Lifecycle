#!/usr/bin/env bash
set -euo pipefail

# Install monitoring stack (Prometheus Operator, Grafana, Loki, Tempo) using Helm
# Then wait for CRDs and deploy the user-service overlay with kubectl (kustomize)

HELM=${HELM:-helm}
KUBECTL=${KUBECTL:-kubectl}

NAMESPACE_MON=monitoring
NAMESPACE_LOG=logging
NAMESPACE_TRACING=tracing

echo "Ensure Helm and kubectl are available"
command -v "$HELM" >/dev/null 2>&1 || { echo "helm not found"; exit 1; }
command -v "$KUBECTL" >/dev/null 2>&1 || { echo "kubectl not found"; exit 1; }

echo "Add Helm repos"
$HELM repo add prometheus-community https://prometheus-community.github.io/helm-charts
$HELM repo add grafana https://grafana.github.io/helm-charts
$HELM repo update

echo "Install kube-prometheus-stack (Prometheus Operator)"
$HELM upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace "$NAMESPACE_MON" --create-namespace \
  --version 45.6.0 \
  --set grafana.enabled=false

echo "Install Grafana (standalone)"
$HELM upgrade --install grafana grafana/grafana \
  --namespace "$NAMESPACE_MON" --create-namespace \
  --version 9.14.0 \
  --set persistence.enabled=false \
  --set adminPassword="changeme"

echo "Install Loki"
$HELM upgrade --install loki grafana/loki \
  --namespace "$NAMESPACE_LOG" --create-namespace \
  --version 6.8.0 \
  --set persistence.enabled=false

echo "Install Tempo"
$HELM upgrade --install tempo grafana/tempo \
  --namespace "$NAMESPACE_TRACING" --create-namespace \
  --version 6.6.0 \
  --set persistence.enabled=false

echo "Waiting for Prometheus CRDs (ServiceMonitor) to appear..."
until $KUBECTL get crd servicemonitors.monitoring.coreos.com >/dev/null 2>&1; do
  echo -n '.'; sleep 3
done
echo "\nPrometheus CRDs present."

echo "Waiting for Prometheus pods to be ready..."
$KUBECTL wait --for=condition=available deployment/prometheus-kube-prometheus-prometheus -n "$NAMESPACE_MON" --timeout=300s || true

echo "Applying user-service kustomize overlay (production)"
$KUBECTL apply -k gitops/environments/overlays/production/user-service

echo "Deployment applied. Check ArgoCD or kubectl get pods -n production to monitor status."

echo "Done."

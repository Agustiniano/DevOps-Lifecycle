# Ingress Configuration Guide

## Overview
This directory contains Ingress configurations for exposing services externally.

## Prerequisites

### 1. Install Nginx Ingress Controller

```bash
# Using Helm
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer

# Wait for external IP
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

Or using kubectl:
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml
```

### 2. (Optional) Install Cert-Manager for SSL

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer for Let's Encrypt
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com  # CHANGE THIS
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## Quick Setup

### For Local Development (no DNS)

1. **Edit `/etc/hosts`** (or `C:\Windows\System32\drivers\etc\hosts` on Windows):
```
127.0.0.1  argocd.local
127.0.0.1  user-service.local
```

Or get your LoadBalancer IP:
```bash
INGRESS_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "$INGRESS_IP  argocd.local user-service.local" | sudo tee -a /etc/hosts
```

2. **Apply ArgoCD Ingress**:
```bash
kubectl apply -f argocd-ingress.yaml
```

3. **Access services**:
- ArgoCD: http://argocd.local (or https if SSL configured)
- User Service: http://user-service.local

### For Production (with real domain)

1. **Update domain names** in ingress files:
   - `argocd-ingress.yaml`: Change `argocd.local` to `argocd.yourdomain.com`
   - Production overlay: Change `user-service.yourdomain.com` to your domain

2. **Configure DNS**: Point your domains to the LoadBalancer IP:
```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

3. **Uncomment TLS sections** in ingress files after cert-manager is installed

4. **Apply configurations**:
```bash
# ArgoCD ingress
kubectl apply -f argocd-ingress.yaml

# User service ingress is automatically deployed by ArgoCD
```

## Testing

### Test ArgoCD Access
```bash
curl -kv https://argocd.local
# or
curl http://argocd.local
```

### Test User Service
```bash
# Health check
curl http://user-service.local/health

# API endpoints
curl http://user-service.local/api/v1/users

# Metrics
curl http://user-service.local/metrics
```

### Test with port-forward (alternative)
```bash
# ArgoCD
kubectl port-forward svc/argocd-server -n argocd 8080:443

# User Service
kubectl port-forward svc/user-service -n production 8081:80
```

## Troubleshooting

### Check Ingress Status
```bash
kubectl get ingress -A
kubectl describe ingress argocd-server -n argocd
kubectl describe ingress user-service -n production
```

### Check Ingress Controller Logs
```bash
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=100 -f
```

### Verify Service Endpoints
```bash
kubectl get endpoints -n argocd argocd-server
kubectl get endpoints -n production user-service
```

## Security Notes

1. **SSL/TLS**: Always use HTTPS in production
2. **Rate Limiting**: Configure in ingress annotations for production
3. **Authentication**: Consider additional auth layer (OAuth2-proxy, etc.)
4. **Network Policies**: Restrict ingress traffic to specific sources

## Next Steps

- [ ] Install cert-manager for automatic SSL certificates
- [ ] Configure external DNS for automatic DNS management
- [ ] Set up OAuth2/OIDC for ArgoCD
- [ ] Configure rate limiting and WAF rules
- [ ] Add monitoring for ingress metrics

# GitHub Container Registry (GHCR) Image Pull Secret

## Opción 1: GHCR Público (Recomendado para desarrollo)

Si el paquete en GHCR es público, no necesitas un imagePullSecret. 

Para hacer público tu paquete en GHCR:
1. Ve a https://github.com/users/Agustiniano/packages/container/user-service/settings
2. En "Danger Zone" → "Change visibility" → selecciona "Public"

## Opción 2: GHCR Privado (Requiere Secret)

Si mantienes el paquete privado, necesitas crear un secret para pull:

### 1. Crear GitHub Personal Access Token

```bash
# 1. Ve a GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
# 2. Generate new token con estos scopes:
#    - read:packages
# 3. Copia el token
```

### 2. Crear el Secret en Kubernetes

```bash
# Crea el secret en el namespace production
kubectl create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username=Agustiniano \
  --docker-password=YOUR_GITHUB_TOKEN \
  --docker-email=your-email@example.com \
  -n production

# Verifica que se creó
kubectl get secret ghcr-pull-secret -n production
```

### 3. Habilitar en ServiceAccount

Descomenta las líneas en `serviceaccount.yaml`:

```yaml
imagePullSecrets:
  - name: ghcr-pull-secret
```

## Opción 3: Sealed Secrets (Recomendado para producción)

Si usas Sealed Secrets en tu cluster:

```bash
# 1. Crea el secret localmente
kubectl create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username=Agustiniano \
  --docker-password=YOUR_GITHUB_TOKEN \
  --docker-email=your-email@example.com \
  --dry-run=client -o yaml > /tmp/ghcr-secret.yaml

# 2. Encripta con kubeseal
kubeseal --format=yaml < /tmp/ghcr-secret.yaml > gitops/environments/base/user-service/sealed-secret-ghcr.yaml

# 3. Agrega el sealed secret a kustomization.yaml
# resources:
#   - sealed-secret-ghcr.yaml

# 4. Limpia el archivo temporal
rm /tmp/ghcr-secret.yaml
```

## Verificación

```bash
# Verifica que el pod puede pullear la imagen
kubectl get pods -n production -l app=user-service
kubectl describe pod -n production -l app=user-service | grep -A 5 "Events:"
```

## Troubleshooting

Si ves errores como "ImagePullBackOff":

```bash
# 1. Verifica que el secret existe
kubectl get secret ghcr-pull-secret -n production

# 2. Verifica que el serviceaccount lo usa
kubectl get sa user-service -n production -o yaml

# 3. Verifica que la imagen existe en GHCR
# Ve a: https://github.com/Agustiniano?tab=packages

# 4. Intenta pullear manualmente
docker pull ghcr.io/agustiniano/user-service:latest
```

# Deployment Strategies - Blue/Green, Canary, Rolling

## 🎯 Estrategias de Deployment

| Estrategia | Risk | Rollback Time | Resource Cost | Use Case |
|------------|------|---------------|---------------|----------|
| **Rolling Update** | Medium | Slow | Low | Features no críticas |
| **Blue/Green** | Low | Instant | High (2x) | Critical updates, database migrations |
| **Canary** | Very Low | Fast | Medium | High-risk changes, A/B testing |
| **Recreate** | High | Slow | Low | Development only |

## 🔵🟢 Blue/Green Deployment

### Concepto

```
┌─────────────────┐
│   Load Balancer │
└────────┬────────┘
         │
    ┌────┴─────┐
    │          │
┌───▼───┐  ┌──────┐
│ BLUE  │  │ GREEN│
│ v1.0  │  │ v1.1 │
│ 100%  │  │  0%  │
└───────┘  └──────┘

After validation:

┌─────────────────┐
│   Load Balancer │
└────────┬────────┘
         │
    ┌────┴─────┐
    │          │
┌───────┐  ┌──▼───┐
│ BLUE  │  │ GREEN│
│ v1.0  │  │ v1.1 │
│  0%   │  │ 100% │
└───────┘  └──────┘
```

### Implementación con Kubernetes

#### 1. Configuración Base

```yaml
# gitops/environments/base/user-service/deployment-blue.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service-blue
  labels:
    app: user-service
    version: blue
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
      version: blue
  template:
    metadata:
      labels:
        app: user-service
        version: blue
    spec:
      containers:
      - name: api
        image: harbor.acme.com/platform/user-service:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: VERSION
          value: "v1.0.0"
        - name: COLOR
          value: "blue"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service-green
  labels:
    app: user-service
    version: green
spec:
  replicas: 0  # Inicialmente sin réplicas
  selector:
    matchLabels:
      app: user-service
      version: green
  template:
    metadata:
      labels:
        app: user-service
        version: green
    spec:
      containers:
      - name: api
        image: harbor.acme.com/platform/user-service:v1.1.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: VERSION
          value: "v1.1.0"
        - name: COLOR
          value: "green"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### 2. Service con Selector Dinámico

```yaml
# gitops/environments/base/user-service/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: user-service
  labels:
    app: user-service
spec:
  selector:
    app: user-service
    version: blue  # Selector activo
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  type: ClusterIP
```

#### 3. Script de Switch

```bash
#!/bin/bash
# scripts/blue-green-switch.sh

set -e

NAMESPACE="${1:-production}"
SERVICE_NAME="${2:-user-service}"
TARGET_VERSION="${3:-green}"

echo "🔄 Starting Blue/Green deployment switch..."
echo "   Namespace: $NAMESPACE"
echo "   Service: $SERVICE_NAME"
echo "   Target: $TARGET_VERSION"

# 1. Verificar que el target deployment esté healthy
echo "📊 Checking $TARGET_VERSION deployment health..."
READY_REPLICAS=$(kubectl get deployment ${SERVICE_NAME}-${TARGET_VERSION} \
  -n $NAMESPACE \
  -o jsonpath='{.status.readyReplicas}')
DESIRED_REPLICAS=$(kubectl get deployment ${SERVICE_NAME}-${TARGET_VERSION} \
  -n $NAMESPACE \
  -o jsonpath='{.spec.replicas}')

if [ "$READY_REPLICAS" != "$DESIRED_REPLICAS" ]; then
  echo "❌ $TARGET_VERSION deployment not ready: $READY_REPLICAS/$DESIRED_REPLICAS"
  exit 1
fi

echo "✅ $TARGET_VERSION deployment is healthy: $READY_REPLICAS/$DESIRED_REPLICAS"

# 2. Ejecutar smoke tests contra el target
echo "🧪 Running smoke tests against $TARGET_VERSION..."
TARGET_POD=$(kubectl get pods -n $NAMESPACE \
  -l app=${SERVICE_NAME},version=${TARGET_VERSION} \
  -o jsonpath='{.items[0].metadata.name}')

kubectl port-forward -n $NAMESPACE pod/$TARGET_POD 8080:8080 &
PF_PID=$!
sleep 3

# Smoke test
HEALTH_STATUS=$(curl -s http://localhost:8080/health | jq -r '.status')
if [ "$HEALTH_STATUS" != "healthy" ]; then
  echo "❌ Smoke test failed"
  kill $PF_PID
  exit 1
fi

kill $PF_PID
echo "✅ Smoke tests passed"

# 3. Patch del service para cambiar selector
echo "🔀 Switching traffic to $TARGET_VERSION..."
kubectl patch service ${SERVICE_NAME} \
  -n $NAMESPACE \
  -p "{\"spec\":{\"selector\":{\"version\":\"${TARGET_VERSION}\"}}}"

echo "✅ Traffic switched to $TARGET_VERSION"

# 4. Verificar que el tráfico fluye correctamente
echo "⏳ Waiting 30 seconds for traffic to stabilize..."
sleep 30

# 5. Escalar down el deployment anterior
OLD_VERSION="blue"
if [ "$TARGET_VERSION" == "blue" ]; then
  OLD_VERSION="green"
fi

echo "⬇️  Scaling down $OLD_VERSION deployment..."
kubectl scale deployment ${SERVICE_NAME}-${OLD_VERSION} \
  -n $NAMESPACE \
  --replicas=0

echo "✅ Blue/Green deployment completed successfully!"
echo ""
echo "📊 Current state:"
kubectl get deployments -n $NAMESPACE -l app=${SERVICE_NAME}
kubectl get service ${SERVICE_NAME} -n $NAMESPACE -o yaml | grep -A 2 selector
```

#### 4. GitHub Actions Workflow

```yaml
# .github/workflows/deploy-blue-green.yml
name: Blue/Green Deployment

on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment'
        required: true
        type: choice
        options:
          - staging
          - production
      service:
        description: 'Service name'
        required: true
        type: string
      version:
        description: 'Version to deploy'
        required: true
        type: string

jobs:
  deploy-green:
    name: Deploy to Green
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          method: kubeconfig
          kubeconfig: ${{ secrets.KUBE_CONFIG }}

      - name: Update green deployment
        run: |
          kubectl set image deployment/${{ inputs.service }}-green \
            api=harbor.acme.com/platform/${{ inputs.service }}:${{ inputs.version }} \
            -n ${{ inputs.environment }}
          
          kubectl scale deployment/${{ inputs.service }}-green \
            --replicas=3 \
            -n ${{ inputs.environment }}

      - name: Wait for green deployment
        run: |
          kubectl rollout status deployment/${{ inputs.service }}-green \
            -n ${{ inputs.environment }} \
            --timeout=5m

      - name: Run smoke tests
        run: |
          # Port forward to green pod
          GREEN_POD=$(kubectl get pods -n ${{ inputs.environment }} \
            -l app=${{ inputs.service }},version=green \
            -o jsonpath='{.items[0].metadata.name}')
          
          kubectl port-forward -n ${{ inputs.environment }} \
            pod/$GREEN_POD 8080:8080 &
          PF_PID=$!
          sleep 5
          
          # Execute tests
          curl -f http://localhost:8080/health || exit 1
          curl -f http://localhost:8080/ready || exit 1
          
          kill $PF_PID

      - name: Run integration tests
        run: |
          npm run test:integration:green

  switch-traffic:
    name: Switch Traffic to Green
    runs-on: ubuntu-latest
    needs: deploy-green
    environment:
      name: ${{ inputs.environment }}-approval
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          method: kubeconfig
          kubeconfig: ${{ secrets.KUBE_CONFIG }}

      - name: Switch service selector
        run: |
          kubectl patch service ${{ inputs.service }} \
            -n ${{ inputs.environment }} \
            -p '{"spec":{"selector":{"version":"green"}}}'

      - name: Monitor for 5 minutes
        run: |
          echo "Monitoring traffic for 5 minutes..."
          for i in {1..30}; do
            echo "Check $i/30"
            
            # Get error rate from Prometheus
            ERROR_RATE=$(curl -s "http://prometheus:9090/api/v1/query?query=rate(http_requests_total{status=~\"5..\",service=\"${{ inputs.service }}\"}[1m])" | jq -r '.data.result[0].value[1]')
            
            if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
              echo "❌ High error rate detected: $ERROR_RATE"
              echo "🔙 Rolling back..."
              kubectl patch service ${{ inputs.service }} \
                -n ${{ inputs.environment }} \
                -p '{"spec":{"selector":{"version":"blue"}}}'
              exit 1
            fi
            
            sleep 10
          done

      - name: Scale down blue
        run: |
          kubectl scale deployment/${{ inputs.service }}-blue \
            --replicas=0 \
            -n ${{ inputs.environment }}

      - name: Send notification
        uses: 8398a7/action-slack@v3
        with:
          status: custom
          custom_payload: |
            {
              text: "✅ Blue/Green deployment successful",
              attachments: [{
                color: 'good',
                fields: [
                  { title: 'Service', value: '${{ inputs.service }}', short: true },
                  { title: 'Version', value: '${{ inputs.version }}', short: true },
                  { title: 'Environment', value: '${{ inputs.environment }}', short: true }
                ]
              }]
            }
          webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

## 🐦 Canary Deployment

### Concepto

```
Initial state:
┌──────────────┐
│ Load Balancer│
└──────┬───────┘
       │
   ┌───┴────┐
   │        │
┌──▼──┐  ┌─────┐
│ v1.0│  │Canary│
│ 95% │  │ v1.1 │
│     │  │  5%  │
└─────┘  └─────┘

Gradual increase:
┌──────────────┐
│ Load Balancer│
└──────┬───────┘
       │
   ┌───┴────┐
   │        │
┌──▼──┐  ┌──▼──┐
│ v1.0│  │ v1.1│
│ 50% │  │ 50% │
└─────┘  └─────┘

Final state:
┌──────────────┐
│ Load Balancer│
└──────┬───────┘
       │
   ┌───┴────┐
   │        │
┌─────┐  ┌──▼──┐
│ v1.0│  │ v1.1│
│  0% │  │100% │
└─────┘  └─────┘
```

### Implementación con Istio

#### 1. VirtualService para Traffic Splitting

```yaml
# gitops/environments/production/user-service/virtual-service.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: user-service
  namespace: production
spec:
  hosts:
  - user-service
  http:
  - match:
    - headers:
        canary:
          exact: "true"
    route:
    - destination:
        host: user-service
        subset: v1-1-0
      weight: 100
  - route:
    - destination:
        host: user-service
        subset: v1-0-0
      weight: 95
    - destination:
        host: user-service
        subset: v1-1-0
      weight: 5
---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: user-service
  namespace: production
spec:
  host: user-service
  subsets:
  - name: v1-0-0
    labels:
      version: v1.0.0
  - name: v1-1-0
    labels:
      version: v1.1.0
```

#### 2. Progressive Delivery con Flagger

```yaml
# gitops/environments/production/user-service/canary.yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: user-service
  namespace: production
spec:
  # Deployment reference
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  
  # Autoscaling reference
  autoscalerRef:
    apiVersion: autoscaling/v2
    kind: HorizontalPodAutoscaler
    name: user-service
  
  # Service configuration
  service:
    port: 80
    targetPort: 8080
  
  # Canary analysis
  analysis:
    # Schedule interval (default 60s)
    interval: 1m
    # Max number of failed metric checks before rollback
    threshold: 5
    # Max traffic percentage routed to canary (default 50)
    maxWeight: 50
    # Canary increment step (default 5)
    stepWeight: 10
    
    # Prometheus metrics
    metrics:
    - name: request-success-rate
      # Minimum req success rate (non 5xx responses)
      # Percentage (0-100)
      thresholdRange:
        min: 99
      interval: 1m
    
    - name: request-duration
      # Maximum req duration P99
      # Milliseconds
      thresholdRange:
        max: 500
      interval: 1m
    
    - name: error-rate
      templateRef:
        name: error-rate
        namespace: istio-system
      thresholdRange:
        max: 1
      interval: 1m
    
    # Webhooks for additional checks
    webhooks:
    - name: acceptance-test
      type: pre-rollout
      url: http://flagger-loadtester.test/
      timeout: 30s
      metadata:
        type: bash
        cmd: "curl -sd 'test' http://user-service-canary:80/token | grep token"
    
    - name: load-test
      type: rollout
      url: http://flagger-loadtester.test/
      metadata:
        cmd: "hey -z 2m -q 10 -c 2 http://user-service-canary.production/"
```

#### 3. Script de Canary Manual

```bash
#!/bin/bash
# scripts/canary-deployment.sh

set -e

NAMESPACE="$1"
SERVICE="$2"
CANARY_VERSION="$3"
STABLE_VERSION="$4"

echo "🐦 Starting Canary deployment..."
echo "   Service: $SERVICE"
echo "   Stable: $STABLE_VERSION"
echo "   Canary: $CANARY_VERSION"

# Stages: 5%, 10%, 25%, 50%, 100%
STAGES=(5 10 25 50 100)

for WEIGHT in "${STAGES[@]}"; do
  STABLE_WEIGHT=$((100 - WEIGHT))
  
  echo ""
  echo "📊 Setting traffic split: Stable ${STABLE_WEIGHT}% | Canary ${WEIGHT}%"
  
  # Update VirtualService
  kubectl patch virtualservice $SERVICE -n $NAMESPACE --type merge -p "
  spec:
    http:
    - route:
      - destination:
          host: $SERVICE
          subset: $STABLE_VERSION
        weight: $STABLE_WEIGHT
      - destination:
          host: $SERVICE
          subset: $CANARY_VERSION
        weight: $WEIGHT
  "
  
  echo "⏳ Monitoring for 5 minutes..."
  
  # Monitor metrics for 5 minutes
  for i in {1..30}; do
    # Get error rate for canary
    ERROR_RATE=$(prometheus-query \
      "rate(http_requests_total{service='$SERVICE',version='$CANARY_VERSION',status=~'5..'}[1m])")
    
    # Get latency P99 for canary
    LATENCY_P99=$(prometheus-query \
      "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{service='$SERVICE',version='$CANARY_VERSION'}[1m]))")
    
    echo "[$i/30] Error rate: $ERROR_RATE | Latency P99: ${LATENCY_P99}ms"
    
    # Check thresholds
    if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
      echo "❌ Error rate too high! Rolling back..."
      kubectl patch virtualservice $SERVICE -n $NAMESPACE --type merge -p "
      spec:
        http:
        - route:
          - destination:
              host: $SERVICE
              subset: $STABLE_VERSION
            weight: 100
      "
      exit 1
    fi
    
    sleep 10
  done
  
  echo "✅ Stage ${WEIGHT}% successful"
done

echo ""
echo "🎉 Canary deployment completed successfully!"
```

Continúo con más estrategias...

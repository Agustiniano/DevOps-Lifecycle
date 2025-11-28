# Crossplane - Cloud Resources as Kubernetes CRDs

## 🎯 ¿Por Qué Crossplane?

**Terraform** es excelente para infraestructura base y de larga duración.
**Crossplane** es ideal para recursos que las aplicaciones crean/destruyen dinámicamente.

### Casos de Uso

| Recurso | Herramienta Recomendada | Razón |
|---------|------------------------|-------|
| VPC, Subnets, EKS | Terraform | Infraestructura base, larga duración |
| Database por microservicio | Crossplane | Ciclo de vida ligado a la app |
| S3 bucket temporal | Crossplane | Creado/destruido por la app |
| Cache Redis para feature | Crossplane | Efímero, app-specific |

## 📦 Instalación de Crossplane

### 1. Instalar Crossplane en Kubernetes

```bash
# Add Crossplane Helm repo
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update

# Install Crossplane
helm install crossplane \
  --namespace crossplane-system \
  --create-namespace \
  crossplane-stable/crossplane \
  --set args='{--enable-composition-revisions}'

# Verificar instalación
kubectl get pods -n crossplane-system
```

### 2. Instalar Provider AWS

```yaml
# crossplane/providers/aws-provider.yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-aws
spec:
  package: xpkg.upbound.io/upbound/provider-aws:v0.40.0
  packagePullPolicy: IfNotPresent
```

```bash
kubectl apply -f crossplane/providers/aws-provider.yaml

# Verificar provider instalado
kubectl get providers
```

### 3. Configurar Credenciales AWS

```bash
# Crear secret con credenciales AWS
kubectl create secret generic aws-creds \
  -n crossplane-system \
  --from-file=credentials=/path/to/aws-credentials

# Crear ProviderConfig
cat <<EOF | kubectl apply -f -
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: aws-creds
      key: credentials
EOF
```

## 🔧 Compositions - Templates Reutilizables

### Composition: PostgreSQL Database

```yaml
# crossplane/compositions/database-postgres.yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xpostgresdatabases.acme.io
spec:
  group: acme.io
  names:
    kind: XPostgresDatabase
    plural: xpostgresdatabases
  claimNames:
    kind: PostgresDatabase
    plural: postgresdatabases
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  properties:
                    storageGB:
                      type: integer
                      description: "Database storage size in GB"
                      default: 20
                    instanceClass:
                      type: string
                      description: "RDS instance class"
                      default: "db.t3.micro"
                      enum:
                        - db.t3.micro
                        - db.t3.small
                        - db.t3.medium
                        - db.r6g.large
                        - db.r6g.xlarge
                    version:
                      type: string
                      description: "PostgreSQL version"
                      default: "15.4"
                    multiAZ:
                      type: boolean
                      description: "Enable Multi-AZ deployment"
                      default: false
                    backupRetentionDays:
                      type: integer
                      description: "Backup retention in days"
                      default: 7
                  required:
                    - storageGB
              required:
                - parameters
---
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: postgres-rds
  labels:
    provider: aws
    type: database
spec:
  writeConnectionSecretsToNamespace: crossplane-system
  
  compositeTypeRef:
    apiVersion: acme.io/v1alpha1
    kind: XPostgresDatabase

  resources:
    # Subnet Group
    - name: subnetgroup
      base:
        apiVersion: rds.aws.upbound.io/v1beta1
        kind: SubnetGroup
        spec:
          forProvider:
            region: us-east-1
            description: "Subnet group for PostgreSQL database"
            subnetIdSelector:
              matchLabels:
                type: database
      patches:
        - fromFieldPath: metadata.name
          toFieldPath: metadata.name
          transforms:
            - type: string
              string:
                fmt: "%s-subnet-group"

    # Security Group
    - name: securitygroup
      base:
        apiVersion: ec2.aws.upbound.io/v1beta1
        kind: SecurityGroup
        spec:
          forProvider:
            region: us-east-1
            description: "Security group for PostgreSQL database"
            vpcIdSelector:
              matchLabels:
                type: eks
      patches:
        - fromFieldPath: metadata.name
          toFieldPath: metadata.name
          transforms:
            - type: string
              string:
                fmt: "%s-sg"

    # Security Group Rule - Allow from EKS
    - name: securitygrouprule
      base:
        apiVersion: ec2.aws.upbound.io/v1beta1
        kind: SecurityGroupRule
        spec:
          forProvider:
            region: us-east-1
            type: ingress
            fromPort: 5432
            toPort: 5432
            protocol: tcp
            sourceSecurityGroupIdSelector:
              matchLabels:
                component: eks-nodes
      patches:
        - fromFieldPath: metadata.name
          toFieldPath: spec.forProvider.securityGroupIdSelector.matchLabels.database
        - type: ToCompositeFieldPath
          fromFieldPath: metadata.name
          toFieldPath: status.securityGroupId

    # RDS Instance
    - name: rdsinstance
      base:
        apiVersion: rds.aws.upbound.io/v1beta1
        kind: Instance
        spec:
          forProvider:
            region: us-east-1
            engine: postgres
            skipFinalSnapshot: false
            publiclyAccessible: false
            storageEncrypted: true
            storageType: gp3
            autoMinorVersionUpgrade: true
            applyImmediately: false
            
            # Master credentials stored in Secrets Manager
            masterUsername: dbadmin
            masterUserPasswordSecretRef:
              key: password
              name: ""  # Patched from claim
              namespace: crossplane-system
            
            # Performance Insights
            performanceInsightsEnabled: true
            performanceInsightsRetentionPeriod: 7
            
            # Monitoring
            enabledCloudwatchLogsExports:
              - postgresql
              - upgrade
            monitoringInterval: 60
            monitoringRoleArnSelector:
              matchLabels:
                role: rds-monitoring
            
            # Backup
            backupWindow: "03:00-04:00"
            maintenanceWindow: "Mon:04:00-Mon:05:00"
            deletionProtection: false  # Set to true in production
            
            # References to other resources
            dbSubnetGroupNameSelector:
              matchControllerRef: true
            vpcSecurityGroupIdSelector:
              matchControllerRef: true

          writeConnectionSecretToRef:
            namespace: crossplane-system

      patches:
        # Basic metadata
        - fromFieldPath: metadata.name
          toFieldPath: metadata.name
        - fromFieldPath: metadata.name
          toFieldPath: spec.writeConnectionSecretToRef.name
          transforms:
            - type: string
              string:
                fmt: "%s-connection"
        
        # Instance configuration
        - fromFieldPath: spec.parameters.storageGB
          toFieldPath: spec.forProvider.allocatedStorage
        - fromFieldPath: spec.parameters.instanceClass
          toFieldPath: spec.forProvider.instanceClass
        - fromFieldPath: spec.parameters.version
          toFieldPath: spec.forProvider.engineVersion
        - fromFieldPath: spec.parameters.multiAZ
          toFieldPath: spec.forProvider.multiAz
        - fromFieldPath: spec.parameters.backupRetentionDays
          toFieldPath: spec.forProvider.backupRetentionPeriod
        
        # Connection secret
        - type: ToCompositeFieldPath
          fromFieldPath: status.atProvider.endpoint
          toFieldPath: status.endpoint
        - type: ToCompositeFieldPath
          fromFieldPath: status.atProvider.address
          toFieldPath: status.address

      connectionDetails:
        - fromConnectionSecretKey: username
        - fromConnectionSecretKey: password
        - fromConnectionSecretKey: endpoint
        - fromConnectionSecretKey: port
```

### Claim: Crear Database para User Service

```yaml
# applications/microservices/user-service/database-claim.yaml
apiVersion: acme.io/v1alpha1
kind: PostgresDatabase
metadata:
  name: user-service-db
  namespace: production
spec:
  parameters:
    storageGB: 100
    instanceClass: db.r6g.large
    version: "15.4"
    multiAZ: true
    backupRetentionDays: 30
  
  # La conexión se escribe en este secret
  writeConnectionSecretToRef:
    name: user-service-db-connection
```

```bash
# Aplicar el claim
kubectl apply -f applications/microservices/user-service/database-claim.yaml

# Verificar status
kubectl get postgresDatabase -n production

# Ver connection secret
kubectl get secret user-service-db-connection -n production -o yaml
```

## 🗄️ Composition: Redis Cache

```yaml
# crossplane/compositions/cache-redis.yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xrediscaches.acme.io
spec:
  group: acme.io
  names:
    kind: XRedisCache
    plural: xrediscaches
  claimNames:
    kind: RedisCache
    plural: rediscaches
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  properties:
                    nodeType:
                      type: string
                      description: "ElastiCache node type"
                      default: "cache.t3.micro"
                    numCacheNodes:
                      type: integer
                      description: "Number of cache nodes"
                      default: 1
                    engineVersion:
                      type: string
                      description: "Redis engine version"
                      default: "7.0"
                    automaticFailoverEnabled:
                      type: boolean
                      default: false
                  required:
                    - nodeType
              required:
                - parameters
---
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: redis-elasticache
  labels:
    provider: aws
    type: cache
spec:
  writeConnectionSecretsToNamespace: crossplane-system
  
  compositeTypeRef:
    apiVersion: acme.io/v1alpha1
    kind: XRedisCache

  resources:
    # Subnet Group
    - name: subnetgroup
      base:
        apiVersion: elasticache.aws.upbound.io/v1beta1
        kind: SubnetGroup
        spec:
          forProvider:
            region: us-east-1
            description: "Subnet group for Redis cluster"
            subnetIdSelector:
              matchLabels:
                type: private

    # Security Group
    - name: securitygroup
      base:
        apiVersion: ec2.aws.upbound.io/v1beta1
        kind: SecurityGroup
        spec:
          forProvider:
            region: us-east-1
            description: "Security group for Redis cluster"
            vpcIdSelector:
              matchLabels:
                type: eks

    # Security Group Rule
    - name: securitygrouprule
      base:
        apiVersion: ec2.aws.upbound.io/v1beta1
        kind: SecurityGroupRule
        spec:
          forProvider:
            region: us-east-1
            type: ingress
            fromPort: 6379
            toPort: 6379
            protocol: tcp
            sourceSecurityGroupIdSelector:
              matchLabels:
                component: eks-nodes

    # ElastiCache Replication Group
    - name: replicationgroup
      base:
        apiVersion: elasticache.aws.upbound.io/v1beta1
        kind: ReplicationGroup
        spec:
          forProvider:
            region: us-east-1
            engine: redis
            atRestEncryptionEnabled: true
            transitEncryptionEnabled: true
            
            # Auth
            authTokenSecretRef:
              key: token
              name: ""  # Patched
              namespace: crossplane-system
            
            # Backup
            snapshotRetentionLimit: 7
            snapshotWindow: "03:00-05:00"
            maintenanceWindow: "sun:05:00-sun:07:00"
            
            # Auto upgrades
            autoMinorVersionUpgrade: true
            
            # Subnet and Security
            cacheSubnetGroupNameSelector:
              matchControllerRef: true
            securityGroupIdSelector:
              matchControllerRef: true

          writeConnectionSecretToRef:
            namespace: crossplane-system

      patches:
        - fromFieldPath: metadata.name
          toFieldPath: metadata.name
        - fromFieldPath: spec.parameters.nodeType
          toFieldPath: spec.forProvider.nodeType
        - fromFieldPath: spec.parameters.numCacheNodes
          toFieldPath: spec.forProvider.numCacheClusters
        - fromFieldPath: spec.parameters.engineVersion
          toFieldPath: spec.forProvider.engineVersion
        - fromFieldPath: spec.parameters.automaticFailoverEnabled
          toFieldPath: spec.forProvider.automaticFailoverEnabled

      connectionDetails:
        - fromConnectionSecretKey: primaryEndpointAddress
        - fromConnectionSecretKey: readerEndpointAddress
        - fromConnectionSecretKey: port
```

## 📦 Composition: S3 Bucket

```yaml
# crossplane/compositions/storage-s3.yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xs3buckets.acme.io
spec:
  group: acme.io
  names:
    kind: XS3Bucket
    plural: xs3buckets
  claimNames:
    kind: S3Bucket
    plural: s3buckets
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  properties:
                    acl:
                      type: string
                      default: "private"
                    versioning:
                      type: boolean
                      default: true
                    lifecycleDays:
                      type: integer
                      description: "Days before transitioning to Glacier"
                      default: 90
              required:
                - parameters
---
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: s3-encrypted-bucket
spec:
  compositeTypeRef:
    apiVersion: acme.io/v1alpha1
    kind: XS3Bucket

  resources:
    # S3 Bucket
    - name: bucket
      base:
        apiVersion: s3.aws.upbound.io/v1beta1
        kind: Bucket
        spec:
          forProvider:
            region: us-east-1
          deletionPolicy: Delete

    # Versioning
    - name: versioning
      base:
        apiVersion: s3.aws.upbound.io/v1beta1
        kind: BucketVersioningV2
        spec:
          forProvider:
            region: us-east-1
            bucketRef:
              name: ""  # Patched
            versioningConfiguration:
              - status: Enabled

    # Server-Side Encryption
    - name: encryption
      base:
        apiVersion: s3.aws.upbound.io/v1beta1
        kind: BucketServerSideEncryptionConfigurationV2
        spec:
          forProvider:
            region: us-east-1
            bucketRef:
              name: ""  # Patched
            rule:
              - applyServerSideEncryptionByDefault:
                  - sseAlgorithm: AES256

    # Public Access Block
    - name: publicaccessblock
      base:
        apiVersion: s3.aws.upbound.io/v1beta1
        kind: BucketPublicAccessBlock
        spec:
          forProvider:
            region: us-east-1
            bucketRef:
              name: ""  # Patched
            blockPublicAcls: true
            blockPublicPolicy: true
            ignorePublicAcls: true
            restrictPublicBuckets: true

    # Lifecycle Configuration
    - name: lifecycle
      base:
        apiVersion: s3.aws.upbound.io/v1beta1
        kind: BucketLifecycleConfigurationV2
        spec:
          forProvider:
            region: us-east-1
            bucketRef:
              name: ""  # Patched
            rule:
              - id: transition-to-glacier
                status: Enabled
                transition:
                  - storageClass: GLACIER
```

## 🔄 Workflow Completo

### 1. Developer crea un claim

```yaml
# user-service/cache-claim.yaml
apiVersion: acme.io/v1alpha1
kind: RedisCache
metadata:
  name: user-service-cache
  namespace: production
spec:
  parameters:
    nodeType: cache.r6g.large
    numCacheNodes: 3
    engineVersion: "7.0"
    automaticFailoverEnabled: true
  writeConnectionSecretToRef:
    name: user-service-cache-connection
```

### 2. Apply el claim

```bash
kubectl apply -f user-service/cache-claim.yaml
```

### 3. Crossplane crea los recursos

```bash
# Monitorear progreso
kubectl get rediscache -n production -w

# Ver detalles
kubectl describe rediscache user-service-cache -n production
```

### 4. Aplicación consume el secret

```yaml
# user-service/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  template:
    spec:
      containers:
        - name: api
          env:
            - name: REDIS_HOST
              valueFrom:
                secretKeyRef:
                  name: user-service-cache-connection
                  key: primaryEndpointAddress
            - name: REDIS_PORT
              valueFrom:
                secretKeyRef:
                  name: user-service-cache-connection
                  key: port
```

## 🎯 Best Practices

1. **Separation of Concerns**: Terraform para infra base, Crossplane para recursos app-specific
2. **Compositions**: Crear compositions reutilizables, no recursos individuales
3. **Naming**: Usar naming conventions consistentes
4. **Secrets**: Siempre usar `writeConnectionSecretToRef`
5. **Labels**: Usar labels para selectors y organización
6. **Deletion Protection**: Habilitar en producción
7. **Backup**: Configurar retention adecuado
8. **Monitoring**: Integrar con Prometheus

## 📊 Monitoreo de Crossplane

```yaml
# ServiceMonitor para Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: crossplane
  namespace: crossplane-system
spec:
  selector:
    matchLabels:
      app: crossplane
  endpoints:
    - port: metrics
      interval: 30s
```

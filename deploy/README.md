# ACS Sippy Kubernetes Deployment

This directory contains Kubernetes manifests for deploying ACS Sippy to a Kubernetes cluster.

## Components

- **acs-sippy Deployment**: Main application (Go backend + embedded React UI)
- **PostgreSQL StatefulSet**: Database for storing CI analytics data
- **CronJob**: Daily BigQuery sync job
- **Service**: ClusterIP service for internal communication
- **Ingress**: External access (requires ingress controller)
- **ConfigMap**: Application configuration from `config/acs.yaml`
- **Secret**: Template for database DSN and BigQuery credentials

## Prerequisites

1. Kubernetes cluster (v1.24+)
2. kubectl configured to access the cluster
3. Ingress controller installed (e.g., nginx-ingress)
4. GCP service account with read access to `acs-san-stackroxci.ci_metrics` dataset

## Before Deployment

### 1. Update Secret Template

Edit `secret-template.yaml` and replace placeholders:

- `database-dsn`: PostgreSQL connection string (update password)
- `bigquery-credentials.json`: Real GCP service account key JSON

**IMPORTANT**: Do NOT commit real credentials to version control!

### 2. Update Ingress

Edit `ingress.yaml`:

- Replace `acs-sippy.example.com` with your actual hostname
- Uncomment TLS section if using HTTPS

### 3. Build and Push Container Image

```bash
# Build the image
docker build -t acs-sippy:latest .

# Tag for your registry
docker tag acs-sippy:latest YOUR_REGISTRY/acs-sippy:VERSION

# Push to registry
docker push YOUR_REGISTRY/acs-sippy:VERSION

# Update kustomization.yaml with the new image
cd deploy
kustomize edit set image acs-sippy=YOUR_REGISTRY/acs-sippy:VERSION
```

## Deployment Methods

### Option 1: Using Kustomize (Recommended)

```bash
# Create namespace
kubectl create namespace acs-sippy

# Deploy all resources
kubectl apply -k deploy/

# Verify deployment
kubectl get all -n acs-sippy
```

### Option 2: Direct kubectl apply

```bash
# Create namespace
kubectl create namespace acs-sippy

# Apply manifests
kubectl apply -f deploy/configmap.yaml
kubectl apply -f deploy/secret-template.yaml
kubectl apply -f deploy/postgres-statefulset.yaml
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/ingress.yaml
kubectl apply -f deploy/cronjob.yaml
```

## Post-Deployment

### 1. Run Database Migration

```bash
# Get pod name
POD=$(kubectl get pod -n acs-sippy -l app=acs-sippy -o jsonpath='{.items[0].metadata.name}')

# Run migration
kubectl exec -n acs-sippy $POD -- /app/acs-sippy migrate
```

### 2. Initial Data Load (Optional)

```bash
# Trigger manual BigQuery sync
kubectl exec -n acs-sippy $POD -- /app/acs-sippy load --loader=bq-sync
```

### 3. Verify Health

```bash
# Port-forward to access locally
kubectl port-forward -n acs-sippy svc/acs-sippy 8080:80

# Check health endpoint
curl http://localhost:8080/api/health
```

### 4. Access the UI

If using Ingress:
```bash
# Get ingress URL
kubectl get ingress -n acs-sippy

# Access via browser
open https://acs-sippy.example.com
```

## Monitoring

### Check Logs

```bash
# Application logs
kubectl logs -n acs-sippy -l app=acs-sippy -f

# CronJob logs (latest run)
kubectl logs -n acs-sippy -l component=bq-sync --tail=100

# PostgreSQL logs
kubectl logs -n acs-sippy -l app=postgres -f
```

### Check CronJob Status

```bash
# List all jobs
kubectl get jobs -n acs-sippy

# Describe CronJob
kubectl describe cronjob acs-sippy-bq-sync -n acs-sippy
```

## Scaling

```bash
# Scale acs-sippy deployment
kubectl scale deployment acs-sippy -n acs-sippy --replicas=3
```

## Resource Requirements

### Minimum
- acs-sippy: 512Mi RAM, 250m CPU
- PostgreSQL: 1Gi RAM, 500m CPU
- Storage: 100Gi for PostgreSQL data

### Recommended for Production
- acs-sippy: 2Gi RAM, 1 CPU (2 replicas)
- PostgreSQL: 4Gi RAM, 2 CPU
- Storage: 500Gi SSD for PostgreSQL data

## Troubleshooting

### Pods not starting

```bash
kubectl describe pod -n acs-sippy <pod-name>
kubectl logs -n acs-sippy <pod-name>
```

### Database connection issues

```bash
# Check PostgreSQL is running
kubectl get pods -n acs-sippy -l app=postgres

# Test connection from acs-sippy pod
kubectl exec -n acs-sippy <pod-name> -- psql $SIPPY_DATABASE_DSN -c "SELECT 1"
```

### CronJob not running

```bash
# Check CronJob schedule
kubectl get cronjob -n acs-sippy

# Manually trigger a job
kubectl create job -n acs-sippy --from=cronjob/acs-sippy-bq-sync manual-sync-$(date +%s)
```

## Cleanup

```bash
# Delete all resources
kubectl delete -k deploy/

# Or delete namespace (removes everything)
kubectl delete namespace acs-sippy
```

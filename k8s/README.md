# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the PDF processing system.

## Prerequisites

1. A running Kubernetes cluster
2. `kubectl` configured to access your cluster
3. Docker images built for custom services:
   - `pdfme-generator:latest`
   - `pdfme-storage:latest`
   - `pdfme-file-watcher:latest`
   - `pdfme-parser:latest`

## Building Docker Images

Before deploying, build and tag your Docker images:

```bash
# Build pdf-generator
docker build -t pdfme-generator:latest ./pdfme

# Build storage-service
docker build -t pdfme-storage:latest ./storage-service

# Build file-watcher
docker build -t pdfme-file-watcher:latest ./file-watcher

# Build parser-service
docker build -t pdfme-parser:latest ./parser
```

If using a remote registry, tag and push the images:

```bash
# Tag and push (replace <registry> with your registry URL)
docker tag pdfme-generator:latest <registry>/pdfme-generator:latest
docker push <registry>/pdfme-generator:latest

docker tag pdfme-storage:latest <registry>/pdfme-storage:latest
docker push <registry>/pdfme-storage:latest

docker tag pdfme-file-watcher:latest <registry>/pdfme-file-watcher:latest
docker push <registry>/pdfme-file-watcher:latest

docker tag pdfme-parser:latest <registry>/pdfme-parser:latest
docker push <registry>/pdfme-parser:latest
```

Then update the image names in the deployment YAMLs accordingly.

## Deployment Order

Deploy in the following order to ensure dependencies are met:

1. **Create Namespace**
   ```bash
   kubectl apply -f namespace.yaml
   ```

2. **Deploy Infrastructure Services** (order matters for dependencies)
   ```bash
   kubectl apply -f postgres.yaml
   kubectl apply -f redis.yaml
   kubectl apply -f rabbitmq.yaml
   kubectl apply -f minio.yaml
   ```

3. **Wait for Infrastructure Services to be Ready**
   ```bash
   kubectl wait --for=condition=ready pod -l app=postgres -n pdfme --timeout=300s
   kubectl wait --for=condition=ready pod -l app=redis -n pdfme --timeout=300s
   kubectl wait --for=condition=ready pod -l app=rabbitmq -n pdfme --timeout=300s
   kubectl wait --for=condition=ready pod -l app=minio -n pdfme --timeout=300s
   ```

4. **Deploy Application Services**
   ```bash
   kubectl apply -f pdf-generator.yaml
   kubectl apply -f storage-service.yaml
   kubectl apply -f file-watcher.yaml
   kubectl apply -f parser-service.yaml
   ```

## Accessing Services

### Port Forwarding (for development)

```bash
# PDF Generator
kubectl port-forward -n pdfme svc/pdf-generator 3000:3000

# Parser Service
kubectl port-forward -n pdfme svc/parser-service 8080:8080

# RabbitMQ Management UI
kubectl port-forward -n pdfme svc/rabbitmq 15672:15672

# MinIO Console
kubectl port-forward -n pdfme svc/minio 9001:9001

# PostgreSQL
kubectl port-forward -n pdfme svc/postgres 5432:5432

# Redis
kubectl port-forward -n pdfme svc/redis 6379:6379
```

### Production Access

For production deployments, consider:
- Using an Ingress controller for HTTP services
- Configuring LoadBalancer services for external access
- Setting up proper DNS entries

## Configuration

### Secrets

Review and update the following secrets in the YAML files before deploying:

- **RabbitMQ** (`rabbitmq.yaml`): username/password
- **MinIO** (`minio.yaml`): root-user/root-password
- **PostgreSQL** (`postgres.yaml`): username/password/database

### Storage

All services use PersistentVolumeClaims. Ensure your cluster has:
- A default StorageClass configured, or
- Manually create PersistentVolumes, or
- Update the PVCs to reference a specific StorageClass

Storage requirements:
- RabbitMQ: 10Gi
- MinIO: 50Gi
- PostgreSQL: 20Gi
- Redis: 10Gi

### Resource Limits

Resource requests and limits are configured with conservative defaults. Adjust based on your workload:
- Review `resources.requests` and `resources.limits` in each deployment
- Monitor actual usage and adjust accordingly

## Monitoring

Check the status of your deployment:

```bash
# View all resources
kubectl get all -n pdfme

# View pods
kubectl get pods -n pdfme

# View services
kubectl get svc -n pdfme

# View persistent volume claims
kubectl get pvc -n pdfme

# Check logs
kubectl logs -n pdfme -l app=pdf-generator
kubectl logs -n pdfme -l app=parser-service
kubectl logs -n pdfme -l app=storage-service
kubectl logs -n pdfme -l app=file-watcher
```

## Troubleshooting

### Pods not starting

```bash
# Describe pod to see events
kubectl describe pod -n pdfme <pod-name>

# Check logs
kubectl logs -n pdfme <pod-name>
```

### Image pull errors

If you see `ImagePullBackOff` or `ErrImagePull`:
1. Verify images are built and available
2. If using a private registry, create an image pull secret
3. Update `imagePullSecrets` in the deployments

### PostgreSQL Init Script

The `postgres.yaml` includes commented sections for mounting an init SQL script. To use:
1. Uncomment the ConfigMap section at the bottom
2. Add your SQL from `init-db-simplified.sql` to the ConfigMap
3. Uncomment the volume and volumeMount sections in the StatefulSet

## Cleanup

To remove all resources:

```bash
kubectl delete -f parser-service.yaml
kubectl delete -f file-watcher.yaml
kubectl delete -f storage-service.yaml
kubectl delete -f pdf-generator.yaml
kubectl delete -f minio.yaml
kubectl delete -f rabbitmq.yaml
kubectl delete -f redis.yaml
kubectl delete -f postgres.yaml
kubectl delete -f namespace.yaml
```

**Note:** This will also delete the PVCs and their data. To preserve data, remove the PVCs from the YAML files before deletion or back them up first.

# orchestrator/deployments/README.md

# Minikube Development Guide

This guide explains how to run the Nestor Orchestrator locally using Minikube for development and testing.

## 🚀 Quick Start

### Prerequisites

Make sure you have these tools installed:

```bash
# macOS
brew install minikube kubectl helm docker

# Verify installation
minikube version
kubectl version --client
helm version
docker --version
```

**Linux users:** Follow the official installation guides for each tool.

### One-Command Setup

```bash
cd orchestrator/
make minikube-setup
```

**That's it!** The script will:
- ✅ Create a Minikube cluster
- ✅ Build the orchestrator Docker image
- ✅ Deploy Redis and DynamoDB Local
- ✅ Deploy the orchestrator
- ✅ Test everything works
- ✅ Show you how to access it

## 📋 Available Commands

| Command | Description |
|---------|-------------|
| `make minikube-setup` | **Complete setup** - cluster + dependencies + orchestrator |
| `make minikube-status` | Show deployment status and pod information |
| `make minikube-logs` | Stream orchestrator logs (follow mode) |
| `make minikube-forward` | Port forward to localhost:8080 |
| `make minikube-shell` | Open interactive shell in orchestrator pod |
| `make minikube-build` | Rebuild and load orchestrator image |
| `make minikube-deploy` | Deploy orchestrator (without cluster setup) |
| `make minikube-test` | Run health checks against deployment |
| `make minikube-cleanup` | Remove deployment (keep cluster) |
| `make minikube-destroy` | **Destroy everything** including cluster |

## 🔧 Development Workflow

### Initial Setup
```bash
# First time setup
cd orchestrator/
make minikube-setup

# Wait for completion, then access via the URL shown
# Example: http://192.168.49.2:32000
```

### Code → Test Cycle
```bash
# 1. Make your code changes
vim internal/api/handlers/catalog.go

# 2. Rebuild and redeploy
make minikube-build

# 3. Test your changes
make minikube-test

# 4. Check logs if needed
make minikube-logs
```

### Debugging
```bash
# Check what's running
make minikube-status

# Follow logs in real-time
make minikube-logs

# Interactive debugging
make minikube-shell
# Inside pod: you can run commands like:
# - ps aux
# - netstat -tlnp
# - env
# - cat /etc/config/config.yaml
```

## 🌐 Accessing the Orchestrator

After `make minikube-setup` completes, you'll see output like:

```
🎉 Nestor Orchestrator is now running!
==================================

📋 Access Information:
  Direct URL:    http://192.168.49.2:32000

🔗 Useful URLs:
  Health:        http://192.168.49.2:32000/health
  Ready:         http://192.168.49.2:32000/ready
  Metrics:       http://192.168.49.2:32000/metrics
  Components:    http://192.168.49.2:32000/api/v1/components
```

### Alternative: Port Forwarding
If you prefer localhost access:

```bash
make minikube-forward
# Then visit: http://localhost:8080
```

## 🧪 Testing the Deployment

### Automated Tests
```bash
make minikube-test
```

### Manual Testing
```bash
# Replace with your actual URL from the setup output
ORCH_URL="http://192.168.49.2:32000"

# Test endpoints
curl $ORCH_URL/health
curl $ORCH_URL/ready
curl $ORCH_URL/metrics

# Test API (should return empty list initially)
curl $ORCH_URL/api/v1/components
```

### Expected Responses
```bash
# Health check
$ curl http://192.168.49.2:32000/health
{"status":"ok","timestamp":"2024-01-15T10:30:45Z"}

# Ready check
$ curl http://192.168.49.2:32000/ready
{"status":"ready","dependencies":{"dynamodb":"ok","redis":"ok"}}

# Components (initially empty)
$ curl http://192.168.49.2:32000/api/v1/components
{"components":[],"total":0}
```

## 🗂️ What Gets Deployed

### Kubernetes Resources
- **Namespace**: `nestor-system`
- **Orchestrator**: 1 pod with NodePort service
- **Redis**: 1 pod (no auth, no persistence)
- **DynamoDB Local**: 1 pod (shared database)
- **Ingress**: Optional (nginx-based)

### Configuration
The deployment uses these settings:
- **Debug logging** enabled
- **DynamoDB endpoint**: `http://nestor-dynamodb-local:8000`
- **Redis URL**: `redis://nestor-redis-master:6379`
- **Resources**: Low CPU/memory for local dev

## 🔍 Troubleshooting

### Common Issues

#### 1. Setup Fails with "Dependencies Missing"
```bash
# Install missing tools (macOS)
brew install minikube kubectl helm docker

# Verify
minikube version
kubectl version --client
```

#### 2. Minikube Won't Start
```bash
# Reset and try again
make minikube-destroy
make minikube-setup

# Or check if Docker is running
docker ps
```

#### 3. Pod Won't Start / ImagePullBackOff
```bash
# Rebuild the image
make minikube-build

# Check pod status
make minikube-status

# Check detailed pod events
kubectl describe pod -l app.kubernetes.io/name=nestor-orchestrator -n nestor-system
```

#### 4. Can't Access the URL
```bash
# Try port forwarding instead
make minikube-forward
# Then: http://localhost:8080

# Or check the correct IP/port
minikube ip -p nestor-dev
kubectl get svc -n nestor-system
```

#### 5. Health Check Fails
```bash
# Check logs for errors
make minikube-logs

# Check configuration
make minikube-shell
cat /etc/config/config.yaml
```

### Debug Commands

```bash
# Check Minikube cluster
minikube status -p nestor-dev

# Check all pods
kubectl get pods -n nestor-system

# Check services and their endpoints
kubectl get svc,endpoints -n nestor-system

# Check recent events
kubectl get events -n nestor-system --sort-by='.metadata.creationTimestamp'

# Check orchestrator pod details
kubectl describe pod -l app.kubernetes.io/name=nestor-orchestrator -n nestor-system
```

## 🧹 Cleanup

### Remove Deployment Only
```bash
make minikube-cleanup
```
This removes the orchestrator, Redis, and DynamoDB but keeps the Minikube cluster running.

### Destroy Everything
```bash
make minikube-destroy
```
This completely removes the Minikube cluster and all resources.

### Start Fresh
```bash
make minikube-destroy
make minikube-setup
```

## ⚙️ Advanced Usage

### Custom Configuration

The script generates `values-minikube.yaml` with default settings. You can modify this file before deploying:

```bash
# Edit the generated values
vim deployments/helm/values-minikube.yaml

# Redeploy with changes
helm upgrade nestor-orchestrator deployments/helm \
  --namespace nestor-system \
  --values deployments/helm/values-minikube.yaml
```

### Using Different Minikube Profile

The script uses the `nestor-dev` profile by default. To use a different profile, edit the script:

```bash
# Edit the script
vim deployments/scripts/minikube-setup.sh

# Change this line:
MINIKUBE_PROFILE="your-profile-name"
```

### Resource Limits

To increase resources for heavy testing:

```bash
# Edit values-minikube.yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi

# Redeploy
make minikube-deploy
```

## 🎯 Next Steps

Once you have the orchestrator running:

1. **Explore the API**: Check out the available endpoints
2. **Add Components**: Create sample component definitions
3. **Test SSE**: Subscribe to real-time events
4. **Develop Features**: Use the local environment for development

### Example: Adding a Test Component

```bash
# Access the orchestrator pod
make minikube-shell

# Create a sample component (inside the pod)
curl -X POST http://localhost:8080/api/v1/components \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "test-s3-bucket",
      "version": "1.0.0",
      "provider": "aws",
      "category": "storage"
    },
    "spec": {
      "required_inputs": [
        {"name": "bucket_name", "type": "string"}
      ]
    }
  }'
```

## 📚 Additional Resources

- [Minikube Documentation](https://minikube.sigs.k8s.io/docs/)
- [Kubectl Reference](https://kubernetes.io/docs/reference/kubectl/)
- [Helm Documentation](https://helm.sh/docs/)
- [Nestor Architecture Guide](../docs/ARCHITECTURE.md)

---

**Need help?** Check the troubleshooting section above or create an issue in the repository.

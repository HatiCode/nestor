# Nestor

**A modern platform engineering solution for self-service infrastructure.**

## 🏗️ Architecture Overview

Nestor consists of three core components that work together to provide a complete platform engineering solution:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│   📚 CATALOG    │    │ 🎼 ORCHESTRATOR │    │ 🎵 COMPOSERS    │
│                 │    │                 │    │                 │
│ Infrastructure  │    │  Deployment     │    │ Team-Specific   │
│ Resource        │◄───┤  Engine &       │◄───┤ Resource        │
│ Definitions     │    │  Dependency     │    │ Composition     │
│                 │    │  Resolution     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
   Platform Team            Platform Team           Product Teams
   Defines primitives      Orchestrates complex      Create team
   (RDS, S3, VPC...)      deployments with deps    abstractions
```

### 📚 **Catalog Service**
- **Purpose**: Central store for low-level infrastructure resource definitions
- **Owned by**: Platform/Infrastructure teams
- **Contains**: Infrastructure primitives (databases, storage, networking, etc.)
- **Features**: Versioning, validation, discovery, real-time updates

### 🎼 **Orchestrator Service**
- **Purpose**: Complex deployment coordination with dependency resolution
- **Owned by**: Platform teams
- **Handles**: Multi-engine deployments (Crossplane, Pulumi, Terraform, Helm)
- **Features**: Dependency resolution, rollback coordination, GitOps integration

### 🎵 **Composer Service**
- **Purpose**: Team-specific resource composition and abstraction layer
- **Owned by**: Product/Development teams
- **Creates**: Business-focused abstractions from catalog primitives
- **Exposes**: Team APIs for CLI and other tools to consume

### 🖥️ **CLI Tool**
- **Purpose**: Developer interface for infrastructure operations
- **Features**: Code annotation parsing, resource composition, deployment management
- **Integration**: Works with Composers to provide self-service infrastructure

## 🚀 Quick Start

### For Platform Teams

1. **Deploy Core Services**
```bash
# Deploy catalog and orchestrator
helm install nestor-catalog deployments/helm/catalog
helm install nestor-orchestrator deployments/helm/orchestrator
```

2. **Add Infrastructure Primitives**
```yaml
# catalog/resources/aws-rds-mysql.yaml
apiVersion: catalog.nestor.io/v1
kind: ResourceDefinition
metadata:
  name: aws-rds-mysql
  version: "1.0.0"
spec:
  provider: aws
  category: database
  resourceType: mysql
  engines:
    - crossplane
    - pulumi
  inputs:
    - name: instanceClass
      type: string
      required: true
    - name: allocatedStorage
      type: integer
      default: 20
```

### For Development Teams

1. **Deploy Team Composer**
```bash
# Each team gets their own composer
helm install team-alpha-composer deployments/helm/composer \
  --set team.name=alpha \
  --set team.namespace=team-alpha
```

2. **Create Team Abstractions**
```yaml
# composer/compositions/web-app.yaml
apiVersion: composer.nestor.io/v1
kind: ComposedResource
metadata:
  name: web-app
  team: alpha
spec:
  description: "Standard web application stack"
  resources:
    - name: database
      catalogRef: aws-rds-mysql
      config:
        instanceClass: "{{ .params.size }}"
    - name: deployment
      catalogRef: k8s-deployment
      config:
        replicas: "{{ .params.replicas }}"
    - name: cache
      catalogRef: redis-cluster
  dependencies:
    - database → deployment
    - cache → deployment
```

3. **Use in Application Code**
```go
// main.go
//nestor:web-app size=small replicas=3
package main

func main() {
    // Your application code
}
```

4. **Deploy with CLI**
```bash
nestor generate  # Parses annotations, creates resources
nestor apply     # Deploys through composer → orchestrator → catalog
```

## 🎯 Key Benefits

### **For Platform Teams**
- **Central Control**: Manage infrastructure primitives and deployment patterns
- **Governance**: Enforce policies, security, and best practices
- **Reusability**: Define once, use across all teams
- **Multi-Engine**: Support different IaC tools based on requirements

### **For Development Teams**
- **Self-Service**: Create and manage infrastructure without platform team bottlenecks
- **Team Abstractions**: Define business-focused resource compositions
- **Code Integration**: Infrastructure definitions live with application code
- **Familiar Workflow**: Use CLI tools similar to kubectl or terraform

### **For Organizations**
- **Reduced Toil**: Eliminate repetitive infrastructure requests
- **Faster Delivery**: Teams can provision resources in minutes, not days
- **Consistency**: Standardized patterns across all teams and environments
- **Cost Optimization**: Resource sharing and right-sizing built-in

## 📋 Use Cases

### **Multi-Team Platform**
```
Platform Team defines:
├── aws-rds-mysql (v1.2.0)
├── k8s-deployment (v2.1.0)
├── redis-cluster (v1.0.0)
└── vpc-setup (v1.5.0)

Team Alpha composes:
├── web-app (database + deployment + cache)
└── api-gateway (load-balancer + certificates)

Team Beta composes:
├── data-pipeline (kafka + spark + s3)
└── ml-training (gpu-nodes + datasets)
```

### **Progressive Delivery**
```bash
# Deploy to staging
nestor apply --env staging

# Run tests, validate
nestor status --env staging

# Promote to production
nestor promote staging → production
```

### **Multi-Cloud Strategy**
```yaml
# Same abstraction, different providers
web-app:
  staging:
    provider: aws
    region: us-west-2
  production:
    provider: gcp
    region: us-central1
```

## 🛠️ Technology Stack

- **Languages**: Go (services), TypeScript (web interfaces)
- **Storage**: DynamoDB (catalog), Redis (caching)
- **Deployment Engines**: Crossplane, Pulumi, Terraform, Helm
- **Container Platform**: Kubernetes
- **GitOps**: ArgoCD integration
- **Observability**: Prometheus, OpenTelemetry

## 📚 Documentation

- [**Architecture Guide**](docs/ARCHITECTURE.md) - Detailed system design
- [**Development Setup**](docs/developer-guide/development.md) - Local development
- [**Platform Team Guide**](docs/platform-teams/) - Setting up catalog and orchestrator
- [**Team Onboarding**](docs/team-guides/) - Getting started with composers
- [**CLI Reference**](docs/cli/) - Complete command reference
- [**API Documentation**](docs/api/) - Service APIs and integration

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
```bash
# Setup development environment
make setup

# Run all tests
make test

# Start local development
make dev

# Check code quality
make check
```

## 🎯 Roadmap

### **Phase 1: Foundation** (Current)
- ✅ Catalog service with resource definitions
- ✅ Orchestrator with Crossplane support
- 🚧 Composer service with team abstractions
- 🚧 CLI with annotation parsing

### **Phase 2: Platform Maturity**
- Multi-engine support (Pulumi, Terraform)
- Advanced dependency resolution
- Policy enforcement framework
- Web-based management interface

### **Phase 3: Enterprise Features**
- Multi-cluster orchestration
- Advanced RBAC and governance
- Cost optimization and monitoring
- Compliance and audit trails

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙋‍♀️ Support

- **Documentation**: [docs.nestor.dev](https://docs.nestor.dev)
- **Community**: [Discord](https://discord.gg/nestor) -> will move to Slack
- **Issues**: [GitHub Issues](https://github.com/HatiCode/nestor/issues)
- **Security**: [security@nestor.dev](mailto:security@nestor.dev)

---

**Built with ❤️ by platform engineers, for platform engineers.**

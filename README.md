# Nestor

**A modern platform engineering solution for self-service infrastructure through code annotations.**

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI Status](https://img.shields.io/github/workflow/status/HatiCode/nestor/CI)](https://github.com/HatiCode/nestor/actions)

## 🏗️ Architecture Overview

Nestor is a cutting-edge platform engineering solution that enables development teams to provision infrastructure through simple code annotations, while platform teams maintain control over infrastructure primitives and deployment patterns.

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│   📚 CATALOG    │    │ 🎼 ORCHESTRATOR │    │ 🎵 COMPOSER     │
│                 │    │                 │    │                 │
│ Infrastructure  │◄───┤  Deployment     │◄───┤ Team-Specific   │
│ Resource        │    │  Engine &       │    │ Resource        │
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
- **Purpose**: Central repository for infrastructure resource definitions
- **Owned by**: Platform/Infrastructure teams
- **Features**: 
  - Git-based resource synchronization
  - Semantic versioning for all resources
  - Real-time updates via Server-Sent Events (SSE)
  - Comprehensive validation and governance

### 🎼 **Orchestrator Service**
- **Purpose**: Complex deployment coordination with dependency resolution
- **Capabilities**:
  - Multi-engine support (Crossplane, Pulumi, Terraform, Helm)
  - Intelligent dependency graph resolution
  - GitOps integration with ArgoCD
  - Rollback coordination and state management

### 🎵 **Composer Service**
- **Purpose**: Team-specific resource composition and abstraction layer
- **Features**:
  - Business-focused resource abstractions
  - Team API exposure for self-service
  - Policy enforcement and quota management
  - Multi-tenant isolation

### 🖥️ **CLI Tool**
- **Purpose**: Developer interface for infrastructure operations
- **Capabilities**:
  - Code annotation parsing (`//nestor:` directives)
  - Resource generation and composition
  - Deployment management and status tracking
  - Cross-platform support (Linux, macOS, Windows)

### 🔧 **Processor Service**
- **Purpose**: Serverless processing for infrastructure events
- **Features**:
  - AWS Lambda, Google Cloud Functions, Azure Functions support
  - Event-driven resource processing
  - Async workflow coordination

## 🚀 Quick Start

### Prerequisites

- Go 1.24.4+ installed
- Docker (optional, for local development)
- Git for version control

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/HatiCode/nestor.git
cd nestor
```

2. **Setup development environment**
```bash
# Run the automated setup script
./scripts/setup.sh

# This will:
# - Install development tools (golangci-lint, etc.)
# - Create necessary directories
# - Setup Go workspace
# - Configure Git hooks
```

3. **Build all components**
```bash
make build-all
```

### For Platform Teams

1. **Deploy Core Services**
```bash
# Deploy catalog service
helm install nestor-catalog deployments/helm/catalog \
  --set storage.type=dynamodb \
  --set git.repositories[0].url=https://github.com/your-org/platform-resources

# Deploy orchestrator
helm install nestor-orchestrator deployments/helm/orchestrator \
  --set engines.crossplane.enabled=true \
  --set engines.pulumi.enabled=true
```

2. **Define Infrastructure Primitives**
```yaml
# catalog/resources/aws-rds-mysql.yaml
apiVersion: catalog.nestor.io/v1
kind: ComponentDefinition
metadata:
  name: aws-rds-mysql
  version: "1.2.0"
  provider: aws
  category: database
spec:
  deploymentEngines:
    - crossplane
    - pulumi
  requiredInputs:
    - name: instanceClass
      type: string
      validation:
        pattern: "db\\.[a-z0-9]+\\.[a-z0-9]+"
    - name: allocatedStorage
      type: integer
      validation:
        min: 20
        max: 16384
```

### For Development Teams

1. **Create Team Composition**
```yaml
# composer/compositions/web-app.yaml
apiVersion: composer.nestor.io/v1
kind: Composition
metadata:
  name: web-app
  team: platform-team
spec:
  resources:
    - name: database
      type: aws-rds-mysql:1.2.0
      config:
        instanceClass: "db.t3.micro"
    - name: cache
      type: redis-cluster:1.0.0
    - name: deployment
      type: k8s-deployment:2.1.0
      dependsOn: [database, cache]
```

2. **Add Code Annotations**
```go
// main.go
//nestor:web-app size=small replicas=3 environment=staging
package main

func main() {
    // Your application code
}
```

3. **Deploy with CLI**
```bash
# Parse annotations and generate resources
nestor generate

# Deploy through the platform
nestor apply --env staging

# Check deployment status
nestor status --deployment-id abc123

# Rollback if needed
nestor rollback --deployment-id abc123
```

## 🎯 Key Benefits

### **For Platform Teams**
- **Central Control**: Manage infrastructure primitives and deployment patterns
- **Governance**: Enforce policies, security, and compliance requirements
- **Reusability**: Define once, use across all teams and environments
- **Multi-Engine**: Support different IaC tools based on specific requirements

### **For Development Teams**
- **Self-Service**: Provision infrastructure without platform team bottlenecks
- **Simplicity**: Infrastructure as code annotations, no YAML complexity
- **Consistency**: Use pre-approved, tested infrastructure patterns
- **Speed**: Deploy in minutes, not days

### **For Organizations**
- **Reduced Toil**: Eliminate repetitive infrastructure tickets
- **Standardization**: Consistent patterns across all teams
- **Cost Control**: Built-in resource optimization and right-sizing
- **Compliance**: Automated policy enforcement and audit trails

## 📋 Component Status

| Component | Version | Status | Description |
|-----------|---------|--------|-------------|
| **Catalog** | v0.1.0 | 🚧 In Progress | Resource definition management |
| **Orchestrator** | v0.1.0 | 🚧 In Progress | Deployment coordination |
| **Composer** | v0.1.0 | 📋 Planned | Team abstractions |
| **CLI** | v0.1.0 | 🚧 In Progress | Developer interface |
| **Processor** | v0.1.0 | 📋 Planned | Event processing |
| **Shared** | v0.1.0 | ✅ Ready | Common utilities |

## 🛠️ Technology Stack

- **Language**: Go 1.24.4
- **Storage**: DynamoDB (catalog), PostgreSQL (composer), Redis (caching)
- **Deployment Engines**: Crossplane, Pulumi, Terraform, Helm
- **Container Platform**: Kubernetes
- **GitOps**: ArgoCD integration
- **Observability**: Prometheus, OpenTelemetry
- **CI/CD**: GitHub Actions with independent component releases

## 📚 Documentation

- [**Architecture Guide**](docs/ARCHITECTURE.md) - Detailed system design and patterns
- [**Developer Guide**](docs/developer-guide/) - Development setup and workflows
- [**CI/CD Pipeline**](docs/developer-guide/ci-cd-pipeline.md) - Release and deployment processes
- [**Release Process**](docs/developer-guide/release-process.md) - Component versioning strategy
- [**API Documentation**](docs/INTERFACES.md) - Service APIs and contracts
- [**Platform Team Guide**](docs/platform-teams/) - Catalog and orchestrator setup
- [**Team Onboarding**](docs/team-guides/) - Getting started with composers

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

---

**Built with ❤️ by platform engineers, for platform engineers.**
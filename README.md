# Casbin CRD Adapter

[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/casbin-crd-adapter)](https://goreportcard.com/report/github.com/casbin/casbin-crd-adapter)
[![CI](https://github.com/casbin/casbin-crd-adapter/workflows/CI/badge.svg)](https://github.com/casbin/casbin-crd-adapter/actions)
[![Coverage Status](https://codecov.io/gh/casbin/casbin-crd-adapter/branch/master/graph/badge.svg)](https://codecov.io/gh/casbin/casbin-crd-adapter)
[![GoDoc](https://godoc.org/github.com/casbin/casbin-crd-adapter?status.svg)](https://godoc.org/github.com/casbin/casbin-crd-adapter)
[![Release](https://img.shields.io/github/release/casbin/casbin-crd-adapter.svg)](https://github.com/casbin/casbin-crd-adapter/releases/latest)
[![Discord](https://img.shields.io/discord/1022748306096537660?logo=discord&label=discord&color=5865F2)](https://discord.gg/S5UjpzGZjN)

A Kubernetes Custom Resource Definition (CRD) adapter for [Casbin](https://github.com/casbin/casbin). With this adapter, Casbin can load policy rules from Kubernetes Custom Resources instead of traditional databases.

## Features

- **Read-Only Adapter**: Designed to keep Kubernetes CRDs as the single source of truth for policies
- **Namespace-Scoped and Cluster-Scoped**: Supports both namespace-scoped and cluster-scoped policy resources
- **Duplicate Handling**: Automatically deduplicates policies when loading from multiple CRs
- **Deterministic Ordering**: Ensures consistent policy loading order across multiple invocations
- **RBAC Support**: Handles both permission rules (p) and role/group bindings (g)
- **No Database Required**: Eliminates the need for external database dependencies at runtime

## Installation

```bash
go get github.com/casbin/casbin-crd-adapter
```

## CRD Schema

The adapter expects CasbinPolicy custom resources with the following structure:

```yaml
apiVersion: casbin.org/v1alpha1
kind: CasbinPolicy
metadata:
  name: example-policy
  namespace: default  # optional, omit for cluster-scoped
spec:
  policyType: p  # or "g" for grouping policies
  rules:
    - values: ["alice", "data1", "read"]
    - values: ["bob", "data2", "write"]
```

### Policy Types

- **p**: Permission policies (who can do what on which resource)
- **g**: Grouping policies (role/group bindings)

## Usage

### Basic Usage (Namespace-Scoped)

```go
package main

import (
	"log"

	"github.com/casbin/casbin/v3"
	crdadapter "github.com/casbin/casbin-crd-adapter"
)

func main() {
	// Create adapter for namespace-scoped policies
	adapter, err := crdadapter.NewAdapter("default")
	if err != nil {
		log.Fatal(err)
	}

	// Create enforcer with the adapter
	e, err := casbin.NewEnforcer("model.conf", adapter)
	if err != nil {
		log.Fatal(err)
	}

	// Use the enforcer
	ok, err := e.Enforce("alice", "data1", "read")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice can read data1: %v", ok)
}
```

### Cluster-Scoped Usage

```go
// Create adapter for cluster-scoped policies
adapter, err := crdadapter.NewAdapter("")
if err != nil {
	log.Fatal(err)
}
```

### Using with Fake Client (for Testing)

```go
import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	crdadapter "github.com/casbin/casbin-crd-adapter"
)

func TestWithFakeClient() {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	
	adapter := crdadapter.NewAdapterWithClient(client, "default")
	
	e, err := casbin.NewEnforcer("model.conf", adapter)
	// ...
}
```

## Example CRD Definitions

### Permission Policy

```yaml
apiVersion: casbin.org/v1alpha1
kind: CasbinPolicy
metadata:
  name: user-permissions
  namespace: default
spec:
  policyType: p
  rules:
    - values: ["alice", "data1", "read"]
    - values: ["alice", "data1", "write"]
    - values: ["bob", "data2", "write"]
```

### Role/Group Bindings

```yaml
apiVersion: casbin.org/v1alpha1
kind: CasbinPolicy
metadata:
  name: role-bindings
  namespace: default
spec:
  policyType: g
  rules:
    - values: ["alice", "admin"]
    - values: ["bob", "developer"]
```

## RBAC Model Example

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

## Write Operations

This adapter is **read-only** by design. All write operations (`SavePolicy`, `AddPolicy`, `RemovePolicy`, `RemoveFilteredPolicy`) will return `ErrWriteNotSupported`.

To modify policies, update the Kubernetes CRD resources directly using `kubectl` or the Kubernetes API:

```bash
# Apply a new policy
kubectl apply -f policy.yaml

# Update existing policy
kubectl edit casbinpolicy example-policy -n default

# Delete a policy
kubectl delete casbinpolicy example-policy -n default
```

## How It Works

1. The adapter connects to the Kubernetes API server (either in-cluster or via kubeconfig)
2. It lists all CasbinPolicy custom resources (filtered by namespace if specified)
3. Policies are sorted deterministically by namespace, name, and policy type
4. Duplicate policies are automatically removed
5. Policies are loaded into the Casbin enforcer

## Testing

Run the test suite:

```bash
go test -v ./...
```

With coverage:

```bash
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## Development

### Prerequisites

- Go 1.23.0 or higher
- Access to a Kubernetes cluster (for integration testing)

### Building

```bash
go build ./...
```

### Running Tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Casbin](https://github.com/casbin/casbin) - An authorization library that supports access control models like ACL, RBAC, ABAC
- [gorm-adapter](https://github.com/casbin/gorm-adapter) - GORM adapter for Casbin
- [ent-adapter](https://github.com/casbin/ent-adapter) - Ent adapter for Casbin

## Support

- [Discord](https://discord.gg/S5UjpzGZjN)
- [Forum](https://forum.casbin.org)
- [GitHub Issues](https://github.com/casbin/casbin-crd-adapter/issues)

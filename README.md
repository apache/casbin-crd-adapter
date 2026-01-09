# casbin-crd-adapter

A Kubernetes Custom Resource Definition (CRD) based persistence adapter for [Casbin](https://casbin.org/). This adapter loads Casbin policies from Kubernetes CRDs instead of a database, making it ideal for cloud-native applications running in Kubernetes.

## Features

- **Read-only adapter**: Loads policies from Kubernetes CRDs
- **Namespace and cluster-scoped support**: Query policies from a specific namespace or across all namespaces
- **Duplicate handling**: Automatically deduplicates policies when multiple CRDs define the same rule
- **Deterministic ordering**: Ensures consistent policy loading order
- **No database dependencies**: Uses Kubernetes as the source of truth for policies
- **RBAC support**: Supports both permission rules (p) and role/group bindings (g)

## Installation

```bash
go get github.com/casbin/casbin-crd-adapter
```

## Quick Start

### 1. Define the CRD

First, apply the CasbinPolicy CRD to your Kubernetes cluster:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: casbinpolicies.casbin.org
spec:
  group: casbin.org
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                policyRules:
                  type: array
                  items:
                    type: object
                    required:
                      - ptype
                      - rule
                    properties:
                      ptype:
                        type: string
                      rule:
                        type: array
                        items:
                          type: string
                groupingRules:
                  type: array
                  items:
                    type: object
                    required:
                      - ptype
                      - rule
                    properties:
                      ptype:
                        type: string
                      rule:
                        type: array
                        items:
                          type: string
  scope: Namespaced
  names:
    plural: casbinpolicies
    singular: casbinpolicy
    kind: CasbinPolicy
    shortNames:
      - cbp
```

### 2. Create a Policy CRD Instance

Create a CasbinPolicy resource:

```yaml
apiVersion: casbin.org/v1alpha1
kind: CasbinPolicy
metadata:
  name: example-policy
  namespace: default
spec:
  policyRules:
    - ptype: p
      rule: ["alice", "data1", "read"]
    - ptype: p
      rule: ["bob", "data2", "write"]
    - ptype: p
      rule: ["admin", "*", "*"]
  groupingRules:
    - ptype: g
      rule: ["alice", "admin"]
```

### 3. Use the Adapter in Your Application

```go
package main

import (
    "log"
    
    "github.com/casbin/casbin-crd-adapter"
    "github.com/casbin/casbin/v2"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

func main() {
    // Create Kubernetes config (in-cluster or from kubeconfig)
    config, err := rest.InClusterConfig()
    if err != nil {
        // Fallback to kubeconfig
        config, err = clientcmd.BuildConfigFromFlags("", "/path/to/kubeconfig")
        if err != nil {
            log.Fatal(err)
        }
    }
    
    // Create the CRD adapter (namespace-scoped)
    adapter, err := adapter.NewAdapter(config, "default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Or create a cluster-scoped adapter (queries all namespaces)
    // adapter, err := adapter.NewAdapter(config, "")
    
    // Create Casbin enforcer with the adapter
    e, err := casbin.NewEnforcer("path/to/model.conf", adapter)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the enforcer
    allowed, err := e.Enforce("alice", "data1", "read")
    if err != nil {
        log.Fatal(err)
    }
    
    if allowed {
        log.Println("Access granted!")
    } else {
        log.Println("Access denied!")
    }
}
```

## Usage Patterns

### Namespace-Scoped Policies

Load policies only from a specific namespace:

```go
adapter, err := adapter.NewAdapter(config, "production")
```

### Cluster-Scoped Policies

Load policies from all namespaces (pass empty string):

```go
adapter, err := adapter.NewAdapter(config, "")
```

### Using with a Fake Client (for testing)

```go
import (
    "github.com/casbin/casbin-crd-adapter"
    "k8s.io/apimachinery/pkg/runtime"
    dynamicfake "k8s.io/client-go/dynamic/fake"
)

scheme := runtime.NewScheme()
_ = v1alpha1.AddToScheme(scheme)

fakeClient := dynamicfake.NewSimpleDynamicClient(scheme)
adapter := adapter.NewAdapterFromClient(fakeClient, "default")
```

## Policy Format

### Permission Rules (p, p2, etc.)

Permission rules define who can access what resources:

```yaml
policyRules:
  - ptype: p
    rule: ["subject", "object", "action"]
  - ptype: p2
    rule: ["subject", "object", "action"]
```

### Grouping Rules (g, g2, etc.)

Grouping rules define role/group memberships:

```yaml
groupingRules:
  - ptype: g
    rule: ["user", "role"]
  - ptype: g2
    rule: ["user", "group"]
```

## Casbin Model Example

```conf
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

## Limitations

This adapter is **read-only** by design. Write operations (`SavePolicy`, `AddPolicy`, `RemovePolicy`, `RemoveFilteredPolicy`) are not supported and will return an error. This keeps the CRDs as the single source of truth for policies.

To modify policies, update the CRD resources directly using `kubectl` or the Kubernetes API.

## API Reference

### NewAdapter

```go
func NewAdapter(config *rest.Config, namespace string) (*Adapter, error)
```

Creates a new CRD adapter from a Kubernetes rest config.

- `config`: Kubernetes client configuration
- `namespace`: Namespace to query (empty string for cluster-wide)

### NewAdapterFromClient

```go
func NewAdapterFromClient(client dynamic.Interface, namespace string) *Adapter
```

Creates a new CRD adapter from an existing dynamic client. Useful for testing with fake clients.

- `client`: Kubernetes dynamic client
- `namespace`: Namespace to query (empty string for cluster-wide)

### LoadPolicy

```go
func (a *Adapter) LoadPolicy(model model.Model) error
```

Loads all policies from CRDs into the Casbin model. Implements the `persist.Adapter` interface.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Related Projects

- [Casbin](https://github.com/casbin/casbin) - An authorization library that supports access control models
- [casbin-server](https://github.com/casbin/casbin-server) - Casbin as a Service
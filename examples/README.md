# Examples

This directory contains examples demonstrating how to use the Casbin CRD adapter.

## Files

- `crd.yaml` - CustomResourceDefinition for CasbinPolicy
- `rbac-policy.yaml` - Example RBAC policy with role assignments
- `rbac_model.conf` - Casbin RBAC model configuration
- `main.go` - Example Go application using the adapter

## Setup

1. **Apply the CRD to your Kubernetes cluster:**

```bash
kubectl apply -f crd.yaml
```

2. **Create a sample policy:**

```bash
kubectl apply -f rbac-policy.yaml
```

3. **Verify the policy was created:**

```bash
kubectl get casbinpolicies
kubectl describe casbinpolicy rbac-policy
```

## Running the Example

### Prerequisites

- Go 1.25 or later
- Access to a Kubernetes cluster
- kubectl configured to connect to your cluster

### Run the example:

```bash
cd examples
go run main.go
```

Expected output:
```
Testing access control policies:
================================
alice    | data1      | read       | ALLOWED
alice    | data1      | write      | ALLOWED
alice    | data2      | read       | ALLOWED
bob      | data1      | read       | ALLOWED
bob      | data1      | write      | DENIED
charlie  | data1      | read       | ALLOWED
charlie  | data1      | write      | DENIED
```

## Understanding the Example

The example demonstrates a typical RBAC setup:

- **Roles:**
  - `admin` - Can read and write all data
  - `user` - Can read data1 and data2
  - `guest` - Can only read data1

- **Users:**
  - Alice is an admin
  - Bob is a regular user
  - Charlie is a guest

- **Resources:**
  - data1, data2

- **Actions:**
  - read, write

The enforcer checks whether each user can perform specific actions on resources based on their assigned roles.

## Creating Your Own Policies

To create your own policy, create a YAML file following this structure:

```yaml
apiVersion: casbin.org/v1alpha1
kind: CasbinPolicy
metadata:
  name: my-policy
  namespace: default
spec:
  policyRules:
    - ptype: p
      rule: ["role", "resource", "action"]
  groupingRules:
    - ptype: g
      rule: ["user", "role"]
```

Apply it with:
```bash
kubectl apply -f my-policy.yaml
```

## Multiple Policies

You can create multiple CasbinPolicy resources. The adapter will load all policies from the configured namespace (or all namespaces if cluster-wide). Duplicate rules are automatically deduplicated.

## Modifying Policies

Since the adapter is read-only, modify policies using kubectl:

```bash
kubectl edit casbinpolicy rbac-policy
```

Or update the YAML file and reapply:

```bash
kubectl apply -f rbac-policy.yaml
```

After updating, restart your application or reload the enforcer to pick up the changes.

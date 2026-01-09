package adapter

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/casbin/casbin-crd-adapter/pkg/apis/casbin/v1alpha1"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var (
	// ErrNotSupported is returned when attempting unsupported write operations
	ErrNotSupported = errors.New("write operations are not supported by the CRD adapter")
)

// Adapter represents the Kubernetes CRD adapter for Casbin
// It implements the persist.Adapter interface for read-only operations
type Adapter struct {
	client    dynamic.Interface
	namespace string
	gvr       schema.GroupVersionResource
}

// NewAdapter creates a new CRD adapter instance
// If namespace is empty, it will list cluster-scoped resources across all namespaces
func NewAdapter(config *rest.Config, namespace string) (*Adapter, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupName,
		Version:  v1alpha1.Version,
		Resource: "casbinpolicies",
	}

	return &Adapter{
		client:    client,
		namespace: namespace,
		gvr:       gvr,
	}, nil
}

// NewAdapterFromClient creates a new CRD adapter from an existing dynamic client
// This is useful for testing with fake clients
func NewAdapterFromClient(client dynamic.Interface, namespace string) *Adapter {
	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupName,
		Version:  v1alpha1.Version,
		Resource: "casbinpolicies",
	}

	return &Adapter{
		client:    client,
		namespace: namespace,
		gvr:       gvr,
	}
}

// LoadPolicy loads all policy rules from Kubernetes CRDs into the model
func (a *Adapter) LoadPolicy(model model.Model) error {
	ctx := context.Background()

	// List all CasbinPolicy resources
	var unstructuredList *unstructured.UnstructuredList
	var err error

	if a.namespace == "" {
		// List cluster-wide
		unstructuredList, err = a.client.Resource(a.gvr).List(ctx, metav1.ListOptions{})
	} else {
		// List namespace-scoped
		unstructuredList, err = a.client.Resource(a.gvr).Namespace(a.namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to list CasbinPolicy resources: %w", err)
	}

	// Convert to typed objects
	policies := make([]v1alpha1.CasbinPolicy, 0, len(unstructuredList.Items))
	for _, item := range unstructuredList.Items {
		policy := v1alpha1.CasbinPolicy{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &policy); err != nil {
			return fmt.Errorf("failed to convert unstructured to CasbinPolicy: %w", err)
		}
		policies = append(policies, policy)
	}

	// Sort policies by namespace and name for deterministic ordering
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].Namespace != policies[j].Namespace {
			return policies[i].Namespace < policies[j].Namespace
		}
		return policies[i].Name < policies[j].Name
	})

	// Use a map to track unique policy lines and prevent duplicates
	seen := make(map[string]bool)

	// Load all policy rules into the model
	for _, policy := range policies {
		// Load policy rules (p, p2, etc.)
		for _, rule := range policy.Spec.PolicyRules {
			line := formatPolicyLine(rule.PType, rule.Rule)
			if !seen[line] {
				seen[line] = true
				if err := persist.LoadPolicyLine(line, model); err != nil {
					return fmt.Errorf("failed to load policy line %q: %w", line, err)
				}
			}
		}

		// Load grouping rules (g, g2, etc.)
		for _, rule := range policy.Spec.GroupingRules {
			line := formatPolicyLine(rule.PType, rule.Rule)
			if !seen[line] {
				seen[line] = true
				if err := persist.LoadPolicyLine(line, model); err != nil {
					return fmt.Errorf("failed to load grouping line %q: %w", line, err)
				}
			}
		}
	}

	return nil
}

// formatPolicyLine formats a policy rule into a CSV line
// e.g., "p, alice, data1, read" or "g, alice, admin"
func formatPolicyLine(ptype string, rule []string) string {
	parts := make([]string, 0, len(rule)+1)
	parts = append(parts, ptype)
	parts = append(parts, rule...)
	return strings.Join(parts, ", ")
}

// SavePolicy is not supported by the CRD adapter
// CRDs should be modified directly as the source of truth
func (a *Adapter) SavePolicy(model model.Model) error {
	return ErrNotSupported
}

// AddPolicy is not supported by the CRD adapter
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return ErrNotSupported
}

// RemovePolicy is not supported by the CRD adapter
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return ErrNotSupported
}

// RemoveFilteredPolicy is not supported by the CRD adapter
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return ErrNotSupported
}

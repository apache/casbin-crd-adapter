package adapter

import (
	"testing"

	"github.com/casbin/casbin-crd-adapter/pkg/apis/casbin/v1alpha1"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

// Test model for RBAC
const rbacModel = `
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
`

func TestNewAdapterFromClient(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	client := dynamicfake.NewSimpleDynamicClient(scheme)
	adapter := NewAdapterFromClient(client, "default")
	
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
	
	if adapter.namespace != "default" {
		t.Errorf("Expected namespace 'default', got %q", adapter.namespace)
	}
}

func TestLoadPolicy_NamespaceScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create test policy
	policy := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"alice", "data1", "read"}},
				{PType: "p", Rule: []string{"bob", "data2", "write"}},
			},
			GroupingRules: []v1alpha1.GroupingRule{
				{PType: "g", Rule: []string{"alice", "admin"}},
			},
		},
	}
	
	// Convert to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(policy)
	if err != nil {
		t.Fatalf("Failed to convert to unstructured: %v", err)
	}
	
	client := dynamicfake.NewSimpleDynamicClient(scheme, &unstructured.Unstructured{Object: unstructuredObj})
	adapter := NewAdapterFromClient(client, "default")
	
	// Load policy into model
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	err = adapter.LoadPolicy(m)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}
	
	// Verify policy was loaded
	policies, err := m.GetPolicy("p", "p")
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}
	
	// Verify grouping was loaded
	grouping, err := m.GetPolicy("g", "g")
	if err != nil {
		t.Fatalf("Failed to get grouping: %v", err)
	}
	if len(grouping) != 1 {
		t.Errorf("Expected 1 grouping rule, got %d", len(grouping))
	}
}

func TestLoadPolicy_ClusterScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create policies in different namespaces
	policy1 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-ns1",
			Namespace: "namespace1",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"user1", "data1", "read"}},
			},
		},
	}
	
	policy2 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-ns2",
			Namespace: "namespace2",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"user2", "data2", "write"}},
			},
		},
	}
	
	// Convert to unstructured
	obj1, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy1)
	obj2, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy2)
	
	client := dynamicfake.NewSimpleDynamicClient(scheme,
		&unstructured.Unstructured{Object: obj1},
		&unstructured.Unstructured{Object: obj2},
	)
	
	// Use empty namespace for cluster-wide listing
	adapter := NewAdapterFromClient(client, "")
	
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	err = adapter.LoadPolicy(m)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}
	
	// Verify both policies were loaded
	policies, err := m.GetPolicy("p", "p")
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("Expected 2 policies from different namespaces, got %d", len(policies))
	}
}

func TestLoadPolicy_DuplicateHandling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create two policies with duplicate rules
	policy1 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy1",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"alice", "data1", "read"}},
				{PType: "p", Rule: []string{"bob", "data2", "write"}},
			},
		},
	}
	
	policy2 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy2",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				// Duplicate of the first rule in policy1
				{PType: "p", Rule: []string{"alice", "data1", "read"}},
				{PType: "p", Rule: []string{"charlie", "data3", "read"}},
			},
		},
	}
	
	obj1, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy1)
	obj2, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy2)
	
	client := dynamicfake.NewSimpleDynamicClient(scheme,
		&unstructured.Unstructured{Object: obj1},
		&unstructured.Unstructured{Object: obj2},
	)
	adapter := NewAdapterFromClient(client, "default")
	
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	err = adapter.LoadPolicy(m)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}
	
	// Verify duplicate was handled - should have 3 unique policies
	policies, err := m.GetPolicy("p", "p")
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}
	if len(policies) != 3 {
		t.Errorf("Expected 3 unique policies (duplicates removed), got %d", len(policies))
	}
}

func TestLoadPolicy_DeterministicOrdering(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create policies with names that would be unordered
	policy1 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "z-policy",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"user1", "data1", "read"}},
			},
		},
	}
	
	policy2 := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-policy",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"user2", "data2", "write"}},
			},
		},
	}
	
	obj1, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy1)
	obj2, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy2)
	
	// Test multiple times to ensure consistent ordering
	for i := 0; i < 3; i++ {
		client := dynamicfake.NewSimpleDynamicClient(scheme,
			&unstructured.Unstructured{Object: obj1},
			&unstructured.Unstructured{Object: obj2},
		)
		adapter := NewAdapterFromClient(client, "default")
		
		m, err := model.NewModelFromString(rbacModel)
		if err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}
		
		err = adapter.LoadPolicy(m)
		if err != nil {
			t.Fatalf("Failed to load policy: %v", err)
		}
		
		policies, err := m.GetPolicy("p", "p")
		if err != nil {
			t.Fatalf("Failed to get policies: %v", err)
		}
		if len(policies) != 2 {
			t.Errorf("Expected 2 policies, got %d", len(policies))
		}
		
		// Verify ordering is consistent (should be sorted by name)
		// Since a-policy comes before z-policy alphabetically,
		// user2 should come before user1
		if len(policies) >= 2 {
			if policies[0][0] != "user2" {
				t.Errorf("Run %d: Expected first policy subject to be 'user2' (from a-policy), got %q", i+1, policies[0][0])
			}
		}
	}
}

func TestEnforcerIntegration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create a complete RBAC setup
	policy := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbac-policy",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules: []v1alpha1.PolicyRule{
				{PType: "p", Rule: []string{"admin", "data1", "read"}},
				{PType: "p", Rule: []string{"admin", "data1", "write"}},
				{PType: "p", Rule: []string{"user", "data1", "read"}},
			},
			GroupingRules: []v1alpha1.GroupingRule{
				{PType: "g", Rule: []string{"alice", "admin"}},
				{PType: "g", Rule: []string{"bob", "user"}},
			},
		},
	}
	
	obj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy)
	client := dynamicfake.NewSimpleDynamicClient(scheme, &unstructured.Unstructured{Object: obj})
	adapter := NewAdapterFromClient(client, "default")
	
	// Create enforcer with the adapter
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	
	// Test enforcement
	tests := []struct {
		sub    string
		obj    string
		act    string
		expect bool
	}{
		{"alice", "data1", "read", true},   // alice is admin, admin can read
		{"alice", "data1", "write", true},  // alice is admin, admin can write
		{"bob", "data1", "read", true},     // bob is user, user can read
		{"bob", "data1", "write", false},   // bob is user, user cannot write
		{"charlie", "data1", "read", false}, // charlie has no role
	}
	
	for _, tt := range tests {
		result, err := e.Enforce(tt.sub, tt.obj, tt.act)
		if err != nil {
			t.Errorf("Enforce(%q, %q, %q) error: %v", tt.sub, tt.obj, tt.act, err)
		}
		if result != tt.expect {
			t.Errorf("Enforce(%q, %q, %q) = %v, want %v", tt.sub, tt.obj, tt.act, result, tt.expect)
		}
	}
}

func TestWriteOperationsNotSupported(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	client := dynamicfake.NewSimpleDynamicClient(scheme)
	adapter := NewAdapterFromClient(client, "default")
	
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	// Test SavePolicy
	err = adapter.SavePolicy(m)
	if err != ErrNotSupported {
		t.Errorf("SavePolicy: expected ErrNotSupported, got %v", err)
	}
	
	// Test AddPolicy
	err = adapter.AddPolicy("p", "p", []string{"alice", "data1", "read"})
	if err != ErrNotSupported {
		t.Errorf("AddPolicy: expected ErrNotSupported, got %v", err)
	}
	
	// Test RemovePolicy
	err = adapter.RemovePolicy("p", "p", []string{"alice", "data1", "read"})
	if err != ErrNotSupported {
		t.Errorf("RemovePolicy: expected ErrNotSupported, got %v", err)
	}
	
	// Test RemoveFilteredPolicy
	err = adapter.RemoveFilteredPolicy("p", "p", 0, "alice")
	if err != ErrNotSupported {
		t.Errorf("RemoveFilteredPolicy: expected ErrNotSupported, got %v", err)
	}
}

func TestLoadPolicy_EmptyPolicies(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create an empty policy
	policy := &v1alpha1.CasbinPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "casbin.org/v1alpha1",
			Kind:       "CasbinPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "empty-policy",
			Namespace: "default",
		},
		Spec: v1alpha1.CasbinPolicySpec{
			PolicyRules:   []v1alpha1.PolicyRule{},
			GroupingRules: []v1alpha1.GroupingRule{},
		},
	}
	
	obj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(policy)
	client := dynamicfake.NewSimpleDynamicClient(scheme, &unstructured.Unstructured{Object: obj})
	adapter := NewAdapterFromClient(client, "default")
	
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	err = adapter.LoadPolicy(m)
	if err != nil {
		t.Fatalf("Failed to load empty policy: %v", err)
	}
	
	// Verify no policies were loaded
	policies, err := m.GetPolicy("p", "p")
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies, got %d", len(policies))
	}
}

func TestLoadPolicy_NoPolicies(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	
	// Create client with no policies
	client := dynamicfake.NewSimpleDynamicClient(scheme)
	adapter := NewAdapterFromClient(client, "default")
	
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	
	err = adapter.LoadPolicy(m)
	if err != nil {
		t.Fatalf("Failed to load with no policies: %v", err)
	}
	
	// Verify no policies were loaded
	policies, err := m.GetPolicy("p", "p")
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies, got %d", len(policies))
	}
}

func TestFormatPolicyLine(t *testing.T) {
	tests := []struct {
		ptype string
		rule  []string
		want  string
	}{
		{"p", []string{"alice", "data1", "read"}, "p, alice, data1, read"},
		{"g", []string{"alice", "admin"}, "g, alice, admin"},
		{"p2", []string{"bob", "resource", "write"}, "p2, bob, resource, write"},
	}
	
	for _, tt := range tests {
		got := formatPolicyLine(tt.ptype, tt.rule)
		if got != tt.want {
			t.Errorf("formatPolicyLine(%q, %v) = %q, want %q", tt.ptype, tt.rule, got, tt.want)
		}
	}
}

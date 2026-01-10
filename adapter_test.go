// Copyright 2026 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crdadapter

import (
	"testing"

	"github.com/casbin/casbin/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

func createFakeClient(objects ...runtime.Object) dynamic.Interface {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{
		Group:    "casbin.org",
		Version:  "v1alpha1",
		Resource: "casbinpolicies",
	}
	return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr: "CasbinPolicyList",
	}, objects...)
}

func createCasbinPolicyCR(name, namespace, ptype string, rules [][]string) *unstructured.Unstructured {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "casbin.org/v1alpha1",
			"kind":       "CasbinPolicy",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"policyType": ptype,
				"rules":      []interface{}{},
			},
		},
	}

	if namespace != "" {
		cr.Object["metadata"].(map[string]interface{})["namespace"] = namespace
	}

	rulesList := []interface{}{}
	for _, rule := range rules {
		// Convert []string to []interface{} for deep copy compatibility
		values := make([]interface{}, len(rule))
		for i, v := range rule {
			values[i] = v
		}
		rulesList = append(rulesList, map[string]interface{}{
			"values": values,
		})
	}
	cr.Object["spec"].(map[string]interface{})["rules"] = rulesList

	return cr
}

func TestNewAdapterWithClient(t *testing.T) {
	client := createFakeClient()

	adapter := NewAdapterWithClient(client, "default")
	if adapter == nil {
		t.Fatal("adapter should not be nil")
	}
	if adapter.namespace != "default" {
		t.Errorf("expected namespace 'default', got '%s'", adapter.namespace)
	}
}

func TestLoadPolicyNamespaceScoped(t *testing.T) {
	// Create CRs with policies
	cr1 := createCasbinPolicyCR("policy1", "default", "p", [][]string{
		{"alice", "data1", "read"},
		{"bob", "data2", "write"},
	})

	cr2 := createCasbinPolicyCR("policy2", "default", "g", [][]string{
		{"alice", "admin"},
	})

	client := createFakeClient(cr1, cr2)
	adapter := NewAdapterWithClient(client, "default")

	e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	// Check loaded policies
	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("failed to get policies: %v", err)
	}

	if len(policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(policies))
	}

	// Check grouping policies
	grouping, err := e.GetGroupingPolicy()
	if err != nil {
		t.Fatalf("failed to get grouping policies: %v", err)
	}

	if len(grouping) != 1 {
		t.Errorf("expected 1 grouping policy, got %d", len(grouping))
	}
}

func TestLoadPolicyClusterScoped(t *testing.T) {
	// Create cluster-scoped CRs
	cr1 := createCasbinPolicyCR("cluster-policy1", "", "p", [][]string{
		{"alice", "data1", "read"},
	})

	client := createFakeClient(cr1)
	adapter := NewAdapterWithClient(client, "")

	e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("failed to get policies: %v", err)
	}

	if len(policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(policies))
	}
}

func TestLoadPolicyDuplicateHandling(t *testing.T) {
	// Create CRs with duplicate policies
	cr1 := createCasbinPolicyCR("policy1", "default", "p", [][]string{
		{"alice", "data1", "read"},
		{"bob", "data2", "write"},
	})

	cr2 := createCasbinPolicyCR("policy2", "default", "p", [][]string{
		{"alice", "data1", "read"}, // duplicate
		{"charlie", "data3", "read"},
	})

	client := createFakeClient(cr1, cr2)
	adapter := NewAdapterWithClient(client, "default")

	e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("failed to get policies: %v", err)
	}

	// Should only have 3 unique policies
	if len(policies) != 3 {
		t.Errorf("expected 3 unique policies, got %d", len(policies))
	}
}

func TestLoadPolicyDeterministicOrdering(t *testing.T) {
	cr1 := createCasbinPolicyCR("zpolicy", "ns2", "p", [][]string{
		{"user1", "res1", "read"},
	})

	cr2 := createCasbinPolicyCR("apolicy", "ns1", "p", [][]string{
		{"user2", "res2", "write"},
	})

	cr3 := createCasbinPolicyCR("bpolicy", "ns1", "p", [][]string{
		{"user3", "res3", "read"},
	})

	client := createFakeClient(cr1, cr2, cr3)
	adapter := NewAdapterWithClient(client, "")

	// Load multiple times and check consistency
	for i := 0; i < 5; i++ {
		e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
		if err != nil {
			t.Fatalf("failed to create enforcer: %v", err)
		}

		policies, err := e.GetPolicy()
		if err != nil {
			t.Fatalf("failed to get policies: %v", err)
		}

		// Policies should be ordered by namespace, then name
		if len(policies) != 3 {
			t.Errorf("expected 3 policies, got %d", len(policies))
		}

		// First should be from ns1/apolicy
		if policies[0][0] != "user2" {
			t.Errorf("ordering incorrect: first policy should be user2, got %s", policies[0][0])
		}

		// Second should be from ns1/bpolicy
		if policies[1][0] != "user3" {
			t.Errorf("ordering incorrect: second policy should be user3, got %s", policies[1][0])
		}

		// Third should be from ns2/zpolicy
		if policies[2][0] != "user1" {
			t.Errorf("ordering incorrect: third policy should be user1, got %s", policies[2][0])
		}
	}
}

func TestSavePolicyNotSupported(t *testing.T) {
	client := createFakeClient()
	adapter := NewAdapterWithClient(client, "default")

	e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	err = e.SavePolicy()
	if err != ErrWriteNotSupported {
		t.Errorf("expected ErrWriteNotSupported, got %v", err)
	}
}

func TestAddPolicyNotSupported(t *testing.T) {
	client := createFakeClient()
	adapter := NewAdapterWithClient(client, "default")

	err := adapter.AddPolicy("p", "p", []string{"alice", "data1", "read"})
	if err != ErrWriteNotSupported {
		t.Errorf("expected ErrWriteNotSupported, got %v", err)
	}
}

func TestRemovePolicyNotSupported(t *testing.T) {
	client := createFakeClient()
	adapter := NewAdapterWithClient(client, "default")

	err := adapter.RemovePolicy("p", "p", []string{"alice", "data1", "read"})
	if err != ErrWriteNotSupported {
		t.Errorf("expected ErrWriteNotSupported, got %v", err)
	}
}

func TestRemoveFilteredPolicyNotSupported(t *testing.T) {
	client := createFakeClient()
	adapter := NewAdapterWithClient(client, "default")

	err := adapter.RemoveFilteredPolicy("p", "p", 0, "alice")
	if err != ErrWriteNotSupported {
		t.Errorf("expected ErrWriteNotSupported, got %v", err)
	}
}

func TestEmptyPolicies(t *testing.T) {
	client := createFakeClient()
	adapter := NewAdapterWithClient(client, "default")

	e, err := casbin.NewEnforcer("testdata/rbac_model.conf", adapter)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	policies, err := e.GetPolicy()
	if err != nil {
		t.Fatalf("failed to get policies: %v", err)
	}

	if len(policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(policies))
	}
}

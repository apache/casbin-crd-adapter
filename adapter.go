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
	"context"
	"errors"
	"sort"

	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ErrWriteNotSupported is returned when write operations are attempted
	ErrWriteNotSupported = errors.New("write operations are not supported by CRD adapter")
)

// CasbinPolicyGVR is the GroupVersionResource for CasbinPolicy CRD
var CasbinPolicyGVR = schema.GroupVersionResource{
	Group:    "casbin.org",
	Version:  "v1alpha1",
	Resource: "casbinpolicies",
}

// Adapter represents the Kubernetes CRD adapter for policy storage
type Adapter struct {
	client    dynamic.Interface
	namespace string
}

// NewAdapter creates a new CRD adapter
// If namespace is empty, it will use cluster-scoped resources
func NewAdapter(namespace string) (*Adapter, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Adapter{
		client:    client,
		namespace: namespace,
	}, nil
}

// NewAdapterWithClient creates a new CRD adapter with a provided dynamic client
// This is useful for testing with a fake client
func NewAdapterWithClient(client dynamic.Interface, namespace string) *Adapter {
	return &Adapter{
		client:    client,
		namespace: namespace,
	}
}

// LoadPolicy loads all policy rules from Kubernetes CRDs
func (a *Adapter) LoadPolicy(model model.Model) error {
	list, err := a.fetchPolicies()
	if err != nil {
		return err
	}

	policyLines := a.parsePolicyLines(list)
	sortPolicyLines(policyLines)

	return a.loadPolicyLines(policyLines, model)
}

// fetchPolicies retrieves policy CRs from Kubernetes
func (a *Adapter) fetchPolicies() (*unstructured.UnstructuredList, error) {
	ctx := context.Background()

	if a.namespace != "" {
		// Namespace-scoped
		return a.client.Resource(CasbinPolicyGVR).Namespace(a.namespace).List(ctx, metav1.ListOptions{})
	}
	// Cluster-scoped
	return a.client.Resource(CasbinPolicyGVR).List(ctx, metav1.ListOptions{})
}

// parsePolicyLines extracts policy lines from unstructured CRs
func (a *Adapter) parsePolicyLines(list *unstructured.UnstructuredList) []policyLine {
	var policyLines []policyLine

	for _, item := range list.Items {
		spec, found, err := unstructured.NestedMap(item.Object, "spec")
		if err != nil || !found {
			continue
		}

		ptype := a.getPolicyType(spec)
		rules, _, _ := unstructured.NestedSlice(spec, "rules")

		for _, rule := range rules {
			if line := a.parsePolicyRule(rule, ptype, item.GetName(), item.GetNamespace()); line != nil {
				policyLines = append(policyLines, *line)
			}
		}
	}

	return policyLines
}

// getPolicyType extracts and validates the policy type
func (a *Adapter) getPolicyType(spec map[string]interface{}) string {
	ptype, _, _ := unstructured.NestedString(spec, "policyType")
	if ptype == "" {
		ptype = "p" // default to permission policy
	}
	return ptype
}

// parsePolicyRule parses a single policy rule
func (a *Adapter) parsePolicyRule(rule interface{}, ptype, name, namespace string) *policyLine {
	ruleMap, ok := rule.(map[string]interface{})
	if !ok {
		return nil
	}

	ruleFields, found, _ := unstructured.NestedStringSlice(ruleMap, "values")
	if !found || len(ruleFields) == 0 {
		return nil
	}

	return &policyLine{
		ptype:     ptype,
		values:    ruleFields,
		name:      name,
		namespace: namespace,
	}
}

// sortPolicyLines sorts policy lines for deterministic ordering
func sortPolicyLines(policyLines []policyLine) {
	sort.Slice(policyLines, func(i, j int) bool {
		return comparePolicyLines(policyLines[i], policyLines[j])
	})
}

// comparePolicyLines compares two policy lines for sorting
func comparePolicyLines(a, b policyLine) bool {
	// First by namespace
	if a.namespace != b.namespace {
		return a.namespace < b.namespace
	}
	// Then by name
	if a.name != b.name {
		return a.name < b.name
	}
	// Then by ptype
	if a.ptype != b.ptype {
		return a.ptype < b.ptype
	}
	// Then by values
	return compareStringSlices(a.values, b.values)
}

// compareStringSlices compares two string slices lexicographically
func compareStringSlices(a, b []string) bool {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for k := 0; k < minLen; k++ {
		if a[k] != b[k] {
			return a[k] < b[k]
		}
	}
	return len(a) < len(b)
}

// loadPolicyLines loads deduplicated policy lines into the model
func (a *Adapter) loadPolicyLines(policyLines []policyLine, model model.Model) error {
	seen := make(map[string]bool)

	for _, line := range policyLines {
		key := buildPolicyKey(line)

		if !seen[key] {
			seen[key] = true
			if err := a.loadSinglePolicy(line, model); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildPolicyKey creates a unique key for deduplication
func buildPolicyKey(line policyLine) string {
	key := line.ptype
	for _, v := range line.values {
		key += ":" + v
	}
	return key
}

// loadSinglePolicy loads a single policy into the model
func (a *Adapter) loadSinglePolicy(line policyLine, model model.Model) error {
	p := make([]string, len(line.values)+1)
	p[0] = line.ptype
	copy(p[1:], line.values)
	return persist.LoadPolicyArray(p, model)
}

// SavePolicy is not supported in read-only CRD adapter
func (a *Adapter) SavePolicy(model model.Model) error {
	return ErrWriteNotSupported
}

// AddPolicy is not supported in read-only CRD adapter
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return ErrWriteNotSupported
}

// RemovePolicy is not supported in read-only CRD adapter
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return ErrWriteNotSupported
}

// RemoveFilteredPolicy is not supported in read-only CRD adapter
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return ErrWriteNotSupported
}

// policyLine represents a single policy line with metadata for sorting
type policyLine struct {
	ptype     string
	values    []string
	name      string
	namespace string
}

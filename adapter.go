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
	client     dynamic.Interface
	namespace  string
	isFiltered bool
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
	ctx := context.Background()

	var list *unstructured.UnstructuredList
	var err error

	if a.namespace != "" {
		// Namespace-scoped
		list, err = a.client.Resource(CasbinPolicyGVR).Namespace(a.namespace).List(ctx, metav1.ListOptions{})
	} else {
		// Cluster-scoped
		list, err = a.client.Resource(CasbinPolicyGVR).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return err
	}

	// Collect all policy lines
	var policyLines []policyLine

	for _, item := range list.Items {
		spec, found, err := unstructured.NestedMap(item.Object, "spec")
		if err != nil || !found {
			continue
		}

		// Get policy type (p or g)
		ptype, _, _ := unstructured.NestedString(spec, "policyType")
		if ptype == "" {
			ptype = "p" // default to permission policy
		}

		// Get rules array
		rules, _, _ := unstructured.NestedSlice(spec, "rules")

		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract rule fields
			var ruleFields []string
			if vals, found, _ := unstructured.NestedStringSlice(ruleMap, "values"); found {
				ruleFields = vals
			}

			if len(ruleFields) > 0 {
				policyLines = append(policyLines, policyLine{
					ptype:  ptype,
					values: ruleFields,
					// For deterministic ordering
					name:      item.GetName(),
					namespace: item.GetNamespace(),
				})
			}
		}
	}

	// Sort for deterministic ordering
	sort.Slice(policyLines, func(i, j int) bool {
		// First by namespace
		if policyLines[i].namespace != policyLines[j].namespace {
			return policyLines[i].namespace < policyLines[j].namespace
		}
		// Then by name
		if policyLines[i].name != policyLines[j].name {
			return policyLines[i].name < policyLines[j].name
		}
		// Then by ptype
		if policyLines[i].ptype != policyLines[j].ptype {
			return policyLines[i].ptype < policyLines[j].ptype
		}
		// Then by values
		for k := 0; k < len(policyLines[i].values) && k < len(policyLines[j].values); k++ {
			if policyLines[i].values[k] != policyLines[j].values[k] {
				return policyLines[i].values[k] < policyLines[j].values[k]
			}
		}
		return len(policyLines[i].values) < len(policyLines[j].values)
	})

	// Remove duplicates while maintaining order
	seen := make(map[string]bool)
	for _, line := range policyLines {
		key := line.ptype
		for _, v := range line.values {
			key += ":" + v
		}

		if !seen[key] {
			seen[key] = true
			// Build the policy array for Casbin
			p := make([]string, len(line.values)+1)
			p[0] = line.ptype
			copy(p[1:], line.values)

			if err := persist.LoadPolicyArray(p, model); err != nil {
				return err
			}
		}
	}

	return nil
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

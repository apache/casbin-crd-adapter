package main

import (
	"log"

	adapter "github.com/casbin/casbin-crd-adapter"
	"github.com/casbin/casbin/v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Load Kubernetes configuration
	// Try in-cluster config first, then fall back to kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		// Running outside cluster, use kubeconfig
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Failed to create Kubernetes config: %v", err)
		}
	}

	// Create the CRD adapter for the "default" namespace
	// Use "" for cluster-wide policies across all namespaces
	a, err := adapter.NewAdapter(config, "default")
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Create Casbin enforcer with the RBAC model and CRD adapter
	e, err := casbin.NewEnforcer("examples/rbac_model.conf", a)
	if err != nil {
		log.Fatalf("Failed to create enforcer: %v", err)
	}

	// Test some access control scenarios
	testCases := []struct {
		user     string
		resource string
		action   string
	}{
		{"alice", "data1", "read"},
		{"alice", "data1", "write"},
		{"alice", "data2", "read"},
		{"bob", "data1", "read"},
		{"bob", "data1", "write"},
		{"charlie", "data1", "read"},
		{"charlie", "data1", "write"},
	}

	log.Println("Testing access control policies:")
	log.Println("================================")

	for _, tc := range testCases {
		allowed, err := e.Enforce(tc.user, tc.resource, tc.action)
		if err != nil {
			log.Printf("Error checking %s trying to %s %s: %v", tc.user, tc.action, tc.resource, err)
			continue
		}

		result := "DENIED"
		if allowed {
			result = "ALLOWED"
		}
		log.Printf("%-8s | %-10s | %-10s | %s", tc.user, tc.resource, tc.action, result)
	}
}

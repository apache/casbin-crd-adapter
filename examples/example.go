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

package main

import (
	"log"

	crdadapter "github.com/casbin/casbin-crd-adapter"
	"github.com/casbin/casbin/v3"
)

func main() {
	// Create adapter for namespace-scoped policies in the "default" namespace
	// Use empty string "" for cluster-scoped policies
	adapter, err := crdadapter.NewAdapter("default")
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Create enforcer with RBAC model and CRD adapter
	e, err := casbin.NewEnforcer("examples/rbac_model.conf", adapter)
	if err != nil {
		log.Fatalf("Failed to create enforcer: %v", err)
	}

	// Example 1: Check direct permission
	ok, err := e.Enforce("alice", "data1", "read")
	if err != nil {
		log.Fatalf("Enforce failed: %v", err)
	}
	log.Printf("alice can read data1: %v", ok)

	// Example 2: Check permission through role
	// If alice has the "admin" role and "admin" can read data2
	ok, err = e.Enforce("alice", "data2", "read")
	if err != nil {
		log.Fatalf("Enforce failed: %v", err)
	}
	log.Printf("alice can read data2: %v", ok)

	// Example 3: Get all policies
	policies, err := e.GetPolicy()
	if err != nil {
		log.Fatalf("Failed to get policies: %v", err)
	}
	log.Printf("All policies: %v", policies)

	// Example 4: Get all roles
	roles, err := e.GetGroupingPolicy()
	if err != nil {
		log.Fatalf("Failed to get roles: %v", err)
	}
	log.Printf("All roles: %v", roles)

	// Note: Write operations are not supported
	// err = e.AddPolicy("charlie", "data3", "read")
	// This will return ErrWriteNotSupported
}

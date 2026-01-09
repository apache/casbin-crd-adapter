package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CasbinPolicy is a specification for a Casbin policy
type CasbinPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CasbinPolicySpec `json:"spec"`
}

// CasbinPolicySpec defines the desired state of CasbinPolicy
type CasbinPolicySpec struct {
	// PolicyRules defines the permission rules (e.g., p, alice, data1, read)
	// +optional
	PolicyRules []PolicyRule `json:"policyRules,omitempty"`

	// GroupingRules defines the role/group bindings (e.g., g, alice, admin)
	// +optional
	GroupingRules []GroupingRule `json:"groupingRules,omitempty"`
}

// PolicyRule represents a single policy rule
type PolicyRule struct {
	// PType is the policy type (e.g., "p", "p2")
	PType string `json:"ptype"`

	// Rule contains the rule elements (e.g., ["alice", "data1", "read"])
	Rule []string `json:"rule"`
}

// GroupingRule represents a single role/group binding
type GroupingRule struct {
	// PType is the grouping policy type (e.g., "g", "g2")
	PType string `json:"ptype"`

	// Rule contains the grouping elements (e.g., ["alice", "admin"])
	Rule []string `json:"rule"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CasbinPolicyList is a list of CasbinPolicy resources
type CasbinPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CasbinPolicy `json:"items"`
}

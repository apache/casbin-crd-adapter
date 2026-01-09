package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyObject implements runtime.Object interface
func (in *CasbinPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy creates a deep copy of CasbinPolicy
func (in *CasbinPolicy) DeepCopy() *CasbinPolicy {
	if in == nil {
		return nil
	}
	out := new(CasbinPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *CasbinPolicy) DeepCopyInto(out *CasbinPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopyInto copies all properties of CasbinPolicySpec into another object
func (in *CasbinPolicySpec) DeepCopyInto(out *CasbinPolicySpec) {
	*out = *in
	if in.PolicyRules != nil {
		in, out := &in.PolicyRules, &out.PolicyRules
		*out = make([]PolicyRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.GroupingRules != nil {
		in, out := &in.GroupingRules, &out.GroupingRules
		*out = make([]GroupingRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of CasbinPolicySpec
func (in *CasbinPolicySpec) DeepCopy() *CasbinPolicySpec {
	if in == nil {
		return nil
	}
	out := new(CasbinPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of PolicyRule into another object
func (in *PolicyRule) DeepCopyInto(out *PolicyRule) {
	*out = *in
	if in.Rule != nil {
		in, out := &in.Rule, &out.Rule
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a deep copy of PolicyRule
func (in *PolicyRule) DeepCopy() *PolicyRule {
	if in == nil {
		return nil
	}
	out := new(PolicyRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of GroupingRule into another object
func (in *GroupingRule) DeepCopyInto(out *GroupingRule) {
	*out = *in
	if in.Rule != nil {
		in, out := &in.Rule, &out.Rule
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a deep copy of GroupingRule
func (in *GroupingRule) DeepCopy() *GroupingRule {
	if in == nil {
		return nil
	}
	out := new(GroupingRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object interface
func (in *CasbinPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy creates a deep copy of CasbinPolicyList
func (in *CasbinPolicyList) DeepCopy() *CasbinPolicyList {
	if in == nil {
		return nil
	}
	out := new(CasbinPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of CasbinPolicyList into another object
func (in *CasbinPolicyList) DeepCopyInto(out *CasbinPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CasbinPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

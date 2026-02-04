package v1alpha1

import (
	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/api/server"
)

// Type conversion helpers between server and v1alpha1 packages

func policyServerToV1Alpha1(p server.Policy) v1alpha1.Policy {
	out := v1alpha1.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
	if p.PolicyType != nil {
		t := v1alpha1.PolicyPolicyType(*p.PolicyType)
		out.PolicyType = &t
	}
	return out
}

func policyV1Alpha1ToServer(p v1alpha1.Policy) server.Policy {
	out := server.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
	if p.PolicyType != nil {
		t := server.PolicyPolicyType(*p.PolicyType)
		out.PolicyType = &t
	}
	return out
}

func listResponseV1Alpha1ToServer(r v1alpha1.PolicyList) server.PolicyList {
	policies := make([]server.Policy, len(r.Policies))
	for i, p := range r.Policies {
		policies[i] = policyV1Alpha1ToServer(p)
	}
	return server.PolicyList{
		NextPageToken: r.NextPageToken,
		Policies:      policies,
	}
}

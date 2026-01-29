package v1alpha1

import (
	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/api/server"
)

// Type conversion helpers between server and v1alpha1 packages

func policyServerToV1Alpha1(p server.Policy) v1alpha1.Policy {
	return v1alpha1.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		PolicyType:    v1alpha1.PolicyPolicyType(p.PolicyType),
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
}

func policyV1Alpha1ToServer(p v1alpha1.Policy) server.Policy {
	return server.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		PolicyType:    server.PolicyPolicyType(p.PolicyType),
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
}

func listResponseV1Alpha1ToServer(r v1alpha1.ListPoliciesResponse) server.ListPoliciesResponse {
	policies := make([]server.Policy, len(r.Policies))
	for i, p := range r.Policies {
		policies[i] = policyV1Alpha1ToServer(p)
	}
	return server.ListPoliciesResponse{
		NextPageToken: r.NextPageToken,
		Policies:      policies,
	}
}

// policyUpdateServerToV1Alpha1 converts server PolicyUpdate (PATCH body) to api/v1alpha1 PolicyUpdate.
func policyUpdateServerToV1Alpha1(p server.PolicyUpdate) v1alpha1.PolicyUpdate {
	return v1alpha1.PolicyUpdate{
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		LabelSelector: p.LabelSelector,
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
	}
}

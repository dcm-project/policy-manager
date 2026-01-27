package service

import (
	"fmt"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/store/model"
)

const (
	DefaultPriority = 500
)

// APIToDBModel converts an API Policy model to a database Policy model.
// RegoCode is stripped as it's not stored in the database.
// Optional fields are handled with proper defaults.
func APIToDBModel(api v1alpha1.Policy, id string) model.Policy {
	db := model.Policy{
		ID:          id,
		DisplayName: api.DisplayName,
		PolicyType:  string(api.PolicyType),
	}

	// Handle optional description
	if api.Description != nil {
		db.Description = *api.Description
	}

	// Handle optional enabled field (default: true)
	if api.Enabled != nil {
		db.Enabled = *api.Enabled
	} else {
		db.Enabled = true
	}

	// Handle optional priority (default: 500)
	if api.Priority != nil {
		db.Priority = *api.Priority
	} else {
		db.Priority = DefaultPriority
	}

	// Handle optional label selector
	if api.LabelSelector != nil {
		db.LabelSelector = *api.LabelSelector
	}

	return db
}

// DBToAPIModel converts a database Policy model to an API Policy model.
// Path field is set, RegoCode is returned as empty string, and timestamps are included.
func DBToAPIModel(db *model.Policy) v1alpha1.Policy {
	path := fmt.Sprintf("policies/%s", db.ID)
	regoCode := "" // RegoCode not stored in database, return empty

	api := v1alpha1.Policy{
		Id:          &db.ID,
		Path:        &path,
		DisplayName: db.DisplayName,
		PolicyType:  v1alpha1.PolicyPolicyType(db.PolicyType),
		Priority:    &db.Priority,
		Enabled:     &db.Enabled,
		CreateTime:  &db.CreateTime,
		UpdateTime:  &db.UpdateTime,
		RegoCode:    regoCode,
	}

	// Handle optional description
	if db.Description != "" {
		api.Description = &db.Description
	}

	// Handle optional label selector
	if len(db.LabelSelector) > 0 {
		api.LabelSelector = &db.LabelSelector
	}

	return api
}

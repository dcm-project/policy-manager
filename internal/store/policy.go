package store

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/dcm-project/policy-manager/internal/store/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPolicyNotFound             = errors.New("policy not found")
	ErrPolicyIDTaken              = errors.New("policy ID already taken")
	ErrDisplayNamePolicyTypeTaken = errors.New("display_name and policy_type combination already taken")
	ErrPriorityPolicyTypeTaken    = errors.New("priority and policy_type combination already taken")
)

// PolicyFilter contains optional fields for filtering policy queries.
// nil fields are ignored (not filtered).
type PolicyFilter struct {
	PolicyType *string
	Enabled    *bool
}

// PolicyListOptions contains options for listing policies.
type PolicyListOptions struct {
	Filter    *PolicyFilter
	OrderBy   string
	PageToken *string
	PageSize  int
}

// PolicyListResult contains the result of a List operation.
type PolicyListResult struct {
	Policies      model.PolicyList
	NextPageToken string
}

type Policy interface {
	List(ctx context.Context, opts *PolicyListOptions) (*PolicyListResult, error)
	Create(ctx context.Context, policy model.Policy) (*model.Policy, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, policy model.Policy) (*model.Policy, error)
	Get(ctx context.Context, id string) (*model.Policy, error)
}

type PolicyStore struct {
	db *gorm.DB
}

var _ Policy = (*PolicyStore)(nil)

func NewPolicy(db *gorm.DB) Policy {
	return &PolicyStore{db: db}
}

func (s *PolicyStore) List(ctx context.Context, opts *PolicyListOptions) (*PolicyListResult, error) {
	var policies model.PolicyList
	query := s.db.WithContext(ctx)

	// Default page size
	pageSize := 50
	if opts != nil && opts.PageSize > 0 {
		pageSize = opts.PageSize
	}

	// Decode page token to get offset
	offset := 0
	if opts != nil && opts.PageToken != nil && *opts.PageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(*opts.PageToken)
		if err == nil {
			if parsedOffset, err := strconv.Atoi(string(decoded)); err == nil {
				offset = parsedOffset
			}
		}
	}

	if opts != nil {
		if opts.Filter != nil {
			if opts.Filter.PolicyType != nil {
				query = query.Where("policy_type = ?", *opts.Filter.PolicyType)
			}
			if opts.Filter.Enabled != nil {
				query = query.Where("enabled = ?", *opts.Filter.Enabled)
			}
		}

		// Apply ordering
		if opts.OrderBy != "" {
			query = query.Order(opts.OrderBy)
		} else {
			// Default order by policy_type, priority, id ascending
			query = query.Order("policy_type ASC, priority ASC, id ASC")
		}
	} else {
		// Default order when no options provided
		query = query.Order("policy_type ASC, priority ASC, id ASC")
	}

	// Query with limit+1 to detect if there are more results
	query = query.Limit(pageSize + 1).Offset(offset)

	if err := query.Find(&policies).Error; err != nil {
		return nil, err
	}

	// Generate next page token if there are more results
	result := &PolicyListResult{
		Policies: policies,
	}

	if len(policies) > pageSize {
		// Trim to requested page size
		result.Policies = policies[:pageSize]
		// Encode next offset as page token
		nextOffset := offset + pageSize
		result.NextPageToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(nextOffset)))
	}

	return result, nil
}

// mapUniqueConstraintError maps a DB unique constraint violation to a store sentinel error.
// It checks for gorm.ErrDuplicatedKey and then the constraint/index name in the error message
// (Postgres includes constraint name; SQLite may include column names like "priority" or "display_name").
func mapUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		// Raw driver error (e.g. tests without TranslateError)
		if !strings.Contains(strings.ToLower(err.Error()), "unique") &&
			!strings.Contains(err.Error(), "duplicate key") {
			return err
		}
	}
	msg := err.Error()
	// Prefer index/constraint name; fallback to column names for SQLite (check display_name before priority to avoid misclassification)
	if strings.Contains(msg, "idx_display_name_policy_type") ||
		(strings.Contains(msg, "display_name") && strings.Contains(msg, "policy_type")) {
		return ErrDisplayNamePolicyTypeTaken
	}
	if strings.Contains(msg, "idx_priority_policy_type") ||
		(strings.Contains(msg, "priority") && strings.Contains(msg, "policy_type")) {
		return ErrPriorityPolicyTypeTaken
	}
	return err
}

func (s *PolicyStore) Create(ctx context.Context, policy model.Policy) (*model.Policy, error) {
	if err := s.db.WithContext(ctx).Clauses(clause.Returning{}).Select("*").Create(&policy).Error; err != nil {
		return nil, mapUniqueConstraintError(err)
	}
	return &policy, nil
}

func (s *PolicyStore) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Policy{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

func (s *PolicyStore) Update(ctx context.Context, policy model.Policy) (*model.Policy, error) {
	// Use Select to update all mutable fields including zero values
	// Immutable fields (id, policy_type, create_time) are not updated
	result := s.db.WithContext(ctx).Model(&policy).
		Select("display_name", "description", "label_selector", "priority", "enabled").
		Clauses(clause.Returning{}).
		Updates(&policy)
	if result.Error != nil {
		return nil, mapUniqueConstraintError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrPolicyNotFound
	}
	return &policy, nil
}

func (s *PolicyStore) Get(ctx context.Context, id string) (*model.Policy, error) {
	var policy model.Policy
	if err := s.db.WithContext(ctx).First(&policy, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return &policy, nil
}

package store

import (
	"context"
	"errors"

	"github.com/dcm-project/policy-manager/internal/store/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrPolicyIDTaken  = errors.New("policy ID already taken")
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
	Limit     int
	Offset    int
}

type Policy interface {
	List(ctx context.Context, opts *PolicyListOptions) (model.PolicyList, error)
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

func (s *PolicyStore) List(ctx context.Context, opts *PolicyListOptions) (model.PolicyList, error) {
	var policies model.PolicyList
	query := s.db.WithContext(ctx)

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
			// Default order by priority ascending
			query = query.Order("priority ASC")
		}

		// Apply pagination
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
	} else {
		// Default order when no options provided
		query = query.Order("priority ASC")
	}

	if err := query.Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

func (s *PolicyStore) Create(ctx context.Context, policy model.Policy) (*model.Policy, error) {
	if err := s.db.WithContext(ctx).Clauses(clause.Returning{}).Select("*").Create(&policy).Error; err != nil {
		return nil, err
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
	result := s.db.WithContext(ctx).Model(&policy).Clauses(clause.Returning{}).Updates(&policy)
	if result.Error != nil {
		return nil, result.Error
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

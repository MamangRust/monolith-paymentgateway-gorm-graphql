// Package role adapts the Role service gRPC API into the shared domain model.
// It is the only place that talks to the role query service for role lookups;
// role assignment lives in pkg/adapter/user_role.
package role

import (
	"context"
	"time"

	pbroles "github.com/MamangRust/monolith-payment-gateway-pb/role"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	role_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/role_errors/repository"
)

// QueryRepository is the contract consumers depend on for role reads.
type QueryRepository interface {
	FindByID(ctx context.Context, roleID int) (*models.Role, error)
}

// Repository implements QueryRepository on top of the generated role query
// client.
type Repository struct {
	query pbroles.RoleServiceClient
	guard *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// NewAdapter wraps the generated role query client. Resilience
// (timeout/circuit-breaker/bulkhead) is applied per call through the optional
// dependency guard; a nil guard is a passthrough.
func NewAdapter(query pbroles.RoleServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{query: query}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// New wraps the generated role query client. It is the canonical constructor;
// modules build it inside their repositories with the guard options they were
// given (see GuardOptions patterns in service/*/repository).
func New(query pbroles.RoleServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(query, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface.
func NewQueryAdapter(query pbroles.RoleServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, opts...)
}

func (r *Repository) FindByID(ctx context.Context, roleID int) (*models.Role, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pbroles.ApiResponseRole, error) {
		return r.query.FindByIdRole(ctx, &pbroles.FindByIdRoleRequest{RoleId: int32(roleID)})
	})
	if err != nil {
		return nil, role_errors.ErrRoleNotFound.WithInternal(err)
	}
	if res == nil {
		return nil, nil
	}
	return roleToModel(res.Data), nil
}

// FindByName resolves a role by its name via the role query service.
func (r *Repository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pbroles.ApiResponseRole, error) {
		return r.query.FindByName(ctx, &pbroles.FindByNameRoleRequest{Name: name})
	})
	if err != nil {
		return nil, role_errors.ErrRoleNotFound.WithInternal(err)
	}
	if res == nil {
		return nil, nil
	}
	return roleToModel(res.Data), nil
}

func roleToModel(role *pbroles.RoleResponse) *models.Role {
	if role == nil {
		return nil
	}
	return &models.Role{
		RoleID:    role.Id,
		RoleName:  role.Name,
		CreatedAt: parseTime(role.CreatedAt),
		UpdatedAt: parseTime(role.UpdatedAt),
	}
}

// parseTime accepts the loose RFC3339-ish timestamp strings the role service
// emits and returns the zero time for empty or unparseable values.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

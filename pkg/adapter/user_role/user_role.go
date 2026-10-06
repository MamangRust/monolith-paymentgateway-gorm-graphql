// Package user_role adapts the user-role gRPC service into the shared domain
// model. It owns the only place that assigns or removes roles for a user and
// resolves the roles attached to a user. The user_role service is served by the
// role gRPC server but lives in its own pb/user_role package.
package user_role

import (
	"context"
	"time"

	pbroles "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbuserroles "github.com/MamangRust/monolith-payment-gateway-pb/user_role"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	userrole_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/user_role_errors/repository"
	"google.golang.org/protobuf/types/known/emptypb"
)

// QueryRepository is the read path consumers use to resolve a user's roles.
type QueryRepository interface {
	FindByUserId(ctx context.Context, userID int) ([]*models.Role, error)
}

// CommandRepository is the write path consumers use to (un)assign roles.
type CommandRepository interface {
	AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error)
	RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error
}

// Repository implements QueryRepository and CommandRepository on top of the
// generated user_role client.
type Repository struct {
	client pbuserroles.UserRoleServiceClient
	guard  *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (r *Repository) SetGuard(g *resilience.DependencyGuard) { r.guard = g }

// NewAdapter wraps the generated user_role client. Resilience (timeout/circuit-breaker/
// bulkhead) is applied per call through the optional dependency guard; a nil
// guard is a passthrough.
func NewAdapter(client pbuserroles.UserRoleServiceClient, opts ...adapter.GuardOption) *Repository {
	r := &Repository{client: client}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// New wraps the generated user_role client for user-role operations. It is the
// canonical constructor; modules build it inside their repositories with the
// guard options they were given.
func New(client pbuserroles.UserRoleServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(client, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface.
func NewQueryAdapter(query pbuserroles.UserRoleServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, opts...)
}

// NewCommandAdapter returns the adapter restricted to the CommandRepository
// surface for consumers that only (un)assign roles.
func NewCommandAdapter(command pbuserroles.UserRoleServiceClient, opts ...adapter.GuardOption) CommandRepository {
	return NewAdapter(command, opts...)
}

// FindByUserId implements QueryRepository. The user_role service returns the
// resolved roles, so we map the role responses straight into the domain model.
func (r *Repository) FindByUserId(ctx context.Context, userID int) ([]*models.Role, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pbroles.ApiResponsesRole, error) {
		return r.client.FindByUserId(ctx, &pbuserroles.FindByIdUserRoleRequest{UserId: int32(userID)})
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	roles := make([]*models.Role, 0, len(res.Data))
	for _, item := range res.Data {
		if item == nil {
			continue
		}
		roles = append(roles, roleToModel(item))
	}
	return roles, nil
}

// AssignRoleToUser implements CommandRepository. The user_role service treats a
// repeated assignment as a no-op and returns the existing mapping, so this call
// is idempotent.
func (r *Repository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error) {
	res, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*pbuserroles.ApiResponseUserRole, error) {
		return r.client.AssignRoleToUser(ctx, &pbuserroles.AssignRoleToUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
	})
	if err != nil || res == nil || res.Data == nil {
		return nil, userrole_errors.ErrAssignRoleToUser.WithInternal(err)
	}
	return userRoleToModel(res.Data), nil
}

// RemoveRoleFromUser implements CommandRepository.
func (r *Repository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	_, err := adapter.Call(r.guard, ctx, func(ctx context.Context) (*emptypb.Empty, error) {
		return r.client.RemoveRoleFromUser(ctx, &pbuserroles.RemoveRoleFromUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
	})
	if err != nil {
		return userrole_errors.ErrRemoveRole.WithInternal(err)
	}
	return nil
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

func userRoleToModel(ur *pbuserroles.UserRoleResponse) *models.UserRole {
	if ur == nil {
		return nil
	}
	return &models.UserRole{
		UserRoleID: ur.UserRoleId,
		UserID:     ur.UserId,
		RoleID:     ur.RoleId,
	}
}

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

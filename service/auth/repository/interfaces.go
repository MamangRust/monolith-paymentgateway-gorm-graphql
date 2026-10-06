package repository

import (
	"context"

	roleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/role"
	useradapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user"
	userroleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user_role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
)

// UserRepository delegates to the shared user adapter, whose RPCs are served by
// the user gRPC service.
type UserRepository = *useradapter.Repository

// UserRoleRepository delegates to the shared user_role adapter, whose RPCs are
// served piggybacked on the role gRPC server.
type UserRoleRepository = *userroleadapter.Repository

// RoleRepository delegates to the shared role adapter (query side).
type RoleRepository = *roleadapter.Repository

type ResetTokenRepository interface {
	FindByToken(ctx context.Context, code string) (*models.ResetTokenRow, error)
	CreateResetToken(ctx context.Context, req *requests.CreateResetTokenRequest) (*models.ResetTokenRow, error)
	DeleteResetToken(ctx context.Context, user_id int) error
}

type RefreshTokenRepository interface {
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	FindByUserId(ctx context.Context, user_id int) (*models.RefreshToken, error)
	CreateRefreshToken(ctx context.Context, req *requests.CreateRefreshToken) (*models.RefreshToken, error)
	UpdateRefreshToken(ctx context.Context, req *requests.UpdateRefreshToken) (*models.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteRefreshTokenByUserId(ctx context.Context, user_id int) error
}

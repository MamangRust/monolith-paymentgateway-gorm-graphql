package repository

import (
	"context"

	userroleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user_role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
)

type UserQueryRepository interface {
	FindAllUsers(ctx context.Context, req *requests.FindAllUsers) ([]*models.UserRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllUsers) ([]*models.UserActiveRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllUsers) ([]*models.UserTrashedRow, error)
	FindById(ctx context.Context, user_id int) (*models.UserByIDRow, error)
	FindByEmail(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error)
	FindByEmailAndVerify(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error)
	FindByVerificationCode(ctx context.Context, verificationCode string) (*models.UserByVerificationCodeRow, error)
}

type UserCommandRepository interface {
	CreateUser(ctx context.Context, request *requests.CreateUserRequest) (*models.CreateUserRow, error)
	CreateUserFromRegister(ctx context.Context, request *requests.RegisterRequest) (*models.CreateUserRow, error)
	UpdateUser(ctx context.Context, request *requests.UpdateUserRequest) (*models.UpdateUserRow, error)
	UpdateUserIsVerified(ctx context.Context, user_id int, is_verified bool) (*models.UserIsVerifiedRow, error)
	UpdateUserPassword(ctx context.Context, user_id int, password string) (*models.UserPasswordRow, error)
	TrashedUser(ctx context.Context, user_id int) (*models.TrashUserRow, error)
	RestoreUser(ctx context.Context, user_id int) (*models.RestoreUserRow, error)
	DeleteUserPermanent(ctx context.Context, user_id int) (bool, error)
	RestoreAllUser(ctx context.Context) (bool, error)
	DeleteAllUserPermanent(ctx context.Context) (bool, error)
}

type RoleRepository interface {
	FindByID(ctx context.Context, role_id int) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
}

// UserRoleRepository adapts the user-role gRPC service. service/user uses it to
// assign the default role to a newly created user (mirrors auth register). It is
// satisfied by userroleadapter.CommandRepository.
type UserRoleRepository = userroleadapter.CommandRepository

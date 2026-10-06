package repository

import (
	pbroles "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbuserroles "github.com/MamangRust/monolith-payment-gateway-pb/user_role"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	roleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/role"
	userroleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user_role"
	"gorm.io/gorm"
)

type GuardOptions struct {
	UserRole []adapter.GuardOption
	Role     []adapter.GuardOption
}

type Repositories struct {
	UserCommand UserCommandRepository
	UserQuery   UserQueryRepository
	Role        RoleRepository
	UserRole    UserRoleRepository
}

type Deps struct {
	Db                *gorm.DB
	RoleQueryClient   pbroles.RoleServiceClient
	RoleCommandClient pbroles.RoleCommandServiceClient
	UserRoleClient    pbuserroles.UserRoleServiceClient
	Guard             GuardOptions
}

func NewRepositories(deps *Deps) *Repositories {
	return &Repositories{
		UserCommand: NewUserCommandRepository(deps.Db),
		UserQuery:   NewUserQueryRepository(deps.Db),
		UserRole:    userroleadapter.New(deps.UserRoleClient, deps.Guard.UserRole...),
		Role:        roleadapter.New(deps.RoleQueryClient, deps.Guard.Role...),
	}
}

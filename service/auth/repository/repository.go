package repository

import (
	pbroles "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbusers "github.com/MamangRust/monolith-payment-gateway-pb/user"
	pbuserroles "github.com/MamangRust/monolith-payment-gateway-pb/user_role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	roleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/role"
	useradapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user"
	userroleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user_role"
	"gorm.io/gorm"
)

// Repositories bundles all auth repositories. It uses named fields (not
// embedding) because UserRepository and RoleRepository both declare FindById,
// and RefreshTokenRepository and ResetTokenRepository both declare FindByToken.
type Repositories struct {
	User         UserRepository
	RefreshToken RefreshTokenRepository
	UserRole     UserRoleRepository
	Role         RoleRepository
	ResetToken   ResetTokenRepository
}

// GuardOptions collects the dependency guards for each remote dependency.
type GuardOptions struct {
	User []adapter.GuardOption
	Role []adapter.GuardOption
}

// Compile-time assertions: the shared adapters satisfy auth's repository
// contracts directly.
var (
	_ UserRepository     = (*useradapter.Repository)(nil)
	_ RoleRepository     = (*roleadapter.Repository)(nil)
	_ UserRoleRepository = (*userroleadapter.Repository)(nil)
)

type Deps struct {
	Db                *gorm.DB
	UserQueryClient   pbusers.UserQueryServiceClient
	UserCommandClient pbusers.UserCommandServiceClient
	RoleQueryClient   pbroles.RoleServiceClient
	RoleCommandClient pbroles.RoleCommandServiceClient
	UserRoleClient    pbuserroles.UserRoleServiceClient
	Guard             GuardOptions
}

func NewRepositories(
	deps *Deps,
) *Repositories {
	return &Repositories{
		User:         useradapter.New(deps.UserQueryClient, deps.UserCommandClient, deps.Guard.User...),
		RefreshToken: NewRefreshTokenRepository(deps.Db),
		UserRole:     userroleadapter.New(deps.UserRoleClient, deps.Guard.Role...),
		Role:         roleadapter.New(deps.RoleQueryClient, deps.Guard.Role...),
		ResetToken:   NewResetTokenRepository(deps.Db),
	}
}

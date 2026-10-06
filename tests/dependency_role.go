package tests

import (
	"context"
	"fmt"
	"net"

	pbrole "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbuserroles "github.com/MamangRust/monolith-payment-gateway-pb/user_role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	role_handler "github.com/MamangRust/monolith-payment-gateway-role/handler"
	role_repository "github.com/MamangRust/monolith-payment-gateway-role/repository"
	role_service "github.com/MamangRust/monolith-payment-gateway-role/service"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

// InProcessRoleService starts an in-process role gRPC server backed by gormDB.
// It exposes the role query/command services plus the user-role service that is
// served piggybacked on the same server, and hands back a user-role client. It
// lets auth/user tests exercise the real user_role adapter without an external
// process.
type InProcessRoleService struct {
	Role        pbrole.RoleServiceClient
	RoleCommand pbrole.RoleCommandServiceClient
	UserRole    pbuserroles.UserRoleServiceClient

	conn *grpc.ClientConn
	lis  *bufconn.Listener
	srv  *grpc.Server
}

// StartInProcessRoleService wires and dials the in-process role server.
func StartInProcessRoleService(gormDB *gorm.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface) (*InProcessRoleService, error) {
	roleSvc := role_service.NewService(&role_service.Deps{
		Repositories: role_repository.NewRepositories(gormDB),
		Logger:       log,
		Cache:        cacheStore,
	})
	roleH := role_handler.NewHandler(roleSvc)

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pbrole.RegisterRoleServiceServer(srv, roleH.RoleQuery)
	pbrole.RegisterRoleCommandServiceServer(srv, roleH.RoleCommand)
	pbuserroles.RegisterUserRoleServiceServer(srv, roleH.UserRole)

	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		return nil, fmt.Errorf("dial role service: %w", err)
	}

	return &InProcessRoleService{
		Role:        pbrole.NewRoleServiceClient(conn),
		RoleCommand: pbrole.NewRoleCommandServiceClient(conn),
		UserRole:    pbuserroles.NewUserRoleServiceClient(conn),
		conn:        conn,
		lis:         lis,
		srv:         srv,
	}, nil
}

// Close tears down the client connection and the in-process server.
func (r *InProcessRoleService) Close() {
	if r.conn != nil {
		_ = r.conn.Close()
	}
	if r.srv != nil {
		r.srv.Stop()
	}
}

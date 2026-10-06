package tests

import (
	"context"
	"fmt"
	"net"

	pbuser "github.com/MamangRust/monolith-payment-gateway-pb/user"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	"github.com/MamangRust/monolith-payment-gateway-pkg/hash"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"github.com/MamangRust/monolith-payment-gateway-shared/observability"
	user_handler "github.com/MamangRust/monolith-payment-gateway-user/handler"
	user_repository "github.com/MamangRust/monolith-payment-gateway-user/repository"
	user_service "github.com/MamangRust/monolith-payment-gateway-user/service"
	"github.com/redis/go-redis/v9"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

// UserClient is an in-process gRPC client for the user service, backed by the
// same database as the suite. It lets the card/merchant repositories exercise
// the real user adapter (adapter + dependency guard) without an external
// process.
type UserClient struct {
	Query   pbuser.UserQueryServiceClient
	Command pbuser.UserCommandServiceClient

	log  logger.LoggerInterface
	conn *grpc.ClientConn
	lis  *bufconn.Listener
	srv  *grpc.Server
}

// Guard returns the project-default guard options for the user dependency,
// mirroring the production wiring in apps/server.go.
func (u *UserClient) Guard() []adapter.GuardOption {
	return Guard("user", u.log)
}

// NewUserClient starts an in-process user gRPC service backed by gormDB, using a
// fresh logger and the suite's Redis, then dials it over bufconn. Callers only
// need the database handle and the suite.
func NewUserClient(gormDB *gorm.DB, ts *TestSuite) (*UserClient, error) {
	opts, err := redis.ParseURL(ts.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	redisClient := redis.NewClient(opts)

	log, err := logger.NewLogger("test-user", sdklog.NewLoggerProvider())
	if err != nil {
		return nil, fmt.Errorf("create test logger: %w", err)
	}
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(redisClient, log, cacheMetrics)

	return newUserClient(gormDB, cacheStore, log)
}

// newUserClient starts the in-process user service with explicit cache/logger
// dependencies; it is shared with the cross-service dependency harness.
func newUserClient(gormDB *gorm.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface) (*UserClient, error) {
	userSvc := user_service.NewService(&user_service.Deps{
		Cache:        cacheStore,
		Repositories: user_repository.NewRepositories(&user_repository.Deps{Db: gormDB}),
		Hash:         hash.NewHashingPassword(),
		Logger:       log,
	})

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	userH := user_handler.NewHandler(userSvc)
	pbuser.RegisterUserQueryServiceServer(srv, userH)
	pbuser.RegisterUserCommandServiceServer(srv, userH)

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
		return nil, fmt.Errorf("dial user service: %w", err)
	}

	return &UserClient{
		Query:   pbuser.NewUserQueryServiceClient(conn),
		Command: pbuser.NewUserCommandServiceClient(conn),
		log:     log,
		conn:    conn,
		lis:     lis,
		srv:     srv,
	}, nil
}

// Close tears down the client connection and the in-process server.
func (u *UserClient) Close() {
	if u.conn != nil {
		_ = u.conn.Close()
	}
	if u.srv != nil {
		u.srv.Stop()
	}
}

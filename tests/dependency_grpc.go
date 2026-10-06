package tests

import (
	"context"
	"fmt"
	"net"
	"time"

	card_handler "github.com/MamangRust/monolith-payment-gateway-card/handler"
	card_repository "github.com/MamangRust/monolith-payment-gateway-card/repository"
	card_service "github.com/MamangRust/monolith-payment-gateway-card/service"
	merchant_handler "github.com/MamangRust/monolith-payment-gateway-merchant/handler"
	merchant_repository "github.com/MamangRust/monolith-payment-gateway-merchant/repository"
	merchant_service "github.com/MamangRust/monolith-payment-gateway-merchant/service"
	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbmerchant "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	pbuser "github.com/MamangRust/monolith-payment-gateway-pb/user"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	saldo_handler "github.com/MamangRust/monolith-payment-gateway-saldo/handler"
	saldo_repository "github.com/MamangRust/monolith-payment-gateway-saldo/repository"
	saldo_service "github.com/MamangRust/monolith-payment-gateway-saldo/service"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

// DependencyClients holds in-process gRPC clients for the services that the
// topup/transaction/transfer/withdraw modules call over the network. The server
// is backed by the same database the suite uses, so the production wiring
// (adapter + dependency guard) is exercised end to end without external
// processes.
type DependencyClients struct {
	CardQuery     pbcard.CardQueryServiceClient
	CardCommand   pbcard.CardCommandServiceClient
	SaldoQuery    pbsaldo.SaldoQueryServiceClient
	SaldoCommand  pbsaldo.SaldoCommandServiceClient
	MerchantQuery pbmerchant.MerchantQueryServiceClient
	UserQuery     pbuser.UserQueryServiceClient

	user *UserClient

	conn *grpc.ClientConn
	lis  *bufconn.Listener
	srv  *grpc.Server
}

// NewDependencyClients starts an in-process gRPC server exposing the card,
// saldo and merchant services backed by gormDB, and dials it over bufconn. A
// separate in-process user service is started so the card/merchant repositories
// reach users through the real gRPC adapter.
func NewDependencyClients(gormDB *gorm.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface) (*DependencyClients, error) {
	userClient, err := newUserClient(gormDB, cacheStore, log)
	if err != nil {
		return nil, err
	}

	cardSvc := card_service.NewService(&card_service.Deps{
		Cache: cacheStore,
		Repositories: card_repository.NewRepositories(gormDB, userClient.Query,
			card_repository.GuardOptions{User: Guard("user", log)}),
		Logger: log,
		Kafka:  nil,
	})
	merchantSvc := merchant_service.NewService(&merchant_service.Deps{
		Cache: cacheStore,
		Repositories: merchant_repository.NewRepositories(gormDB, userClient.Query,
			merchant_repository.GuardOptions{User: Guard("user", log)}),
		Logger: log,
	})

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	cardH := card_handler.NewHandler(cardSvc)
	pbcard.RegisterCardQueryServiceServer(srv, cardH)
	pbcard.RegisterCardCommandServiceServer(srv, cardH)

	pbmerchant.RegisterMerchantQueryServiceServer(srv, merchant_handler.NewHandler(merchantSvc))

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		return nil, fmt.Errorf("dial dependency services: %w", err)
	}

	cardQuery := pbcard.NewCardQueryServiceClient(conn)
	cardCommand := pbcard.NewCardCommandServiceClient(conn)

	saldoSvc := saldo_service.NewService(&saldo_service.Deps{
		Cache: cacheStore,
		Repositories: saldo_repository.NewRepositories(gormDB, cardQuery, cardCommand,
			saldo_repository.GuardOptions{Card: Guard("card", log)}),
		Logger: log,
	})

	saldoH := saldo_handler.NewHandler(saldoSvc)
	pbsaldo.RegisterSaldoQueryServiceServer(srv, saldoH)
	pbsaldo.RegisterSaldoCommandServiceServer(srv, saldoH)

	go func() {
		_ = srv.Serve(lis)
	}()

	return &DependencyClients{
		CardQuery:     pbcard.NewCardQueryServiceClient(conn),
		CardCommand:   pbcard.NewCardCommandServiceClient(conn),
		SaldoQuery:    pbsaldo.NewSaldoQueryServiceClient(conn),
		SaldoCommand:  pbsaldo.NewSaldoCommandServiceClient(conn),
		MerchantQuery: pbmerchant.NewMerchantQueryServiceClient(conn),
		UserQuery:     userClient.Query,
		user:          userClient,
		conn:          conn,
		lis:           lis,
		srv:           srv,
	}, nil
}

// Close tears down the client connection and the in-process server.
func (d *DependencyClients) Close() {
	if d.conn != nil {
		_ = d.conn.Close()
	}
	if d.srv != nil {
		d.srv.Stop()
	}
	if d.user != nil {
		d.user.Close()
	}
}

// Guard builds the project-default dependency guard options for a named
// dependency, mirroring the production wiring in apps/server.go.
func Guard(name string, log logger.LoggerInterface) []adapter.GuardOption {
	return []adapter.GuardOption{
		adapter.WithDependencyGuard(resilience.NewDependencyGuard(name, 5, 30, 100, 3*time.Second, log)),
	}
}

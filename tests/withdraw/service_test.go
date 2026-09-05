package withdraw_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	card_repo_impl "github.com/MamangRust/monolith-payment-gateway-card/repository"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	saldo_repo_impl "github.com/MamangRust/monolith-payment-gateway-saldo/repository"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/observability"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	user_repo_impl "github.com/MamangRust/monolith-payment-gateway-user/repository"
	"github.com/MamangRust/monolith-payment-gateway-withdraw/repository"
	"github.com/MamangRust/monolith-payment-gateway-withdraw/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"gorm.io/gorm"
)

type WithdrawServiceTestSuite struct {
	suite.Suite
	gormDB          *gorm.DB
	ts              *tests.TestSuite
	withdrawService service.Service
	withdrawID      int
	cardNumber      string
	userID          int
}

func (s *WithdrawServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	redisClient := redis.NewClient(opts)

	cardRepo := &realCardRepo{query: card_repo_impl.NewCardQueryRepository(gormDB)}
	saldoRepo := &realSaldoRepo{repo: saldo_repo_impl.NewRepositories(gormDB)}
	repos := repository.NewRepositories(gormDB, cardRepo, saldoRepo)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(redisClient, log, cacheMetrics)

	s.withdrawService = service.NewService(&service.Deps{
		Kafka:        nil,
		Repositories: repos,
		Logger:       log,
		Cache:        cacheStore,
	})

	// Seed User
	userRepo := user_repo_impl.NewUserCommandRepository(gormDB)
	user, err := userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Withdraw",
		LastName:  "Tester",
		Email:     fmt.Sprintf("withdraw.tester.%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)

	// Seed Card
	cardCmdRepo := card_repo_impl.NewCardCommandRepository(gormDB)
	card, err := cardCmdRepo.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.cardNumber = card.CardNumber

	// Seed Saldo
	saldoCmdRepo := saldo_repo_impl.NewSaldoCommandRepository(gormDB)
	_, err = saldoCmdRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)
}

func (s *WithdrawServiceTestSuite) TearDownSuite() {
	s.ts.Teardown()
}

func (s *WithdrawServiceTestSuite) Test1_WithdrawLifecycle() {
	ctx := context.Background()

	// Create Withdraw
	createReq := &requests.CreateWithdrawRequest{
		CardNumber:     s.cardNumber,
		WithdrawAmount: 100000,
		WithdrawTime:   time.Now(),
	}
	res, err := s.withdrawService.Create(ctx, createReq)
	s.NoError(err)
	s.NotNil(res)
	s.withdrawID = int(res.WithdrawID)
	s.Equal("success", res.Status)

	// FindById
	found, err := s.withdrawService.FindById(ctx, s.withdrawID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(int32(s.withdrawID), found.WithdrawID)

	// Update Withdraw
	updateReq := &requests.UpdateWithdrawRequest{
		WithdrawID:     &s.withdrawID,
		CardNumber:     s.cardNumber,
		WithdrawAmount: 200000,
		WithdrawTime:   time.Now(),
	}
	updated, err := s.withdrawService.Update(ctx, updateReq)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(200000), updated.WithdrawAmount)
}

func (s *WithdrawServiceTestSuite) Test2_QueryOperations() {
	ctx := context.Background()

	// FindAll
	all, total, err := s.withdrawService.FindAll(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.NotNil(all)
	s.GreaterOrEqual(*total, 1)

	// FindByActive
	active, totalActive, err := s.withdrawService.FindByActive(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.NotNil(active)
	s.GreaterOrEqual(*totalActive, 1)

	// FindByCardNumber
	byCard, totalCard, err := s.withdrawService.FindAllByCardNumber(ctx, &requests.FindAllWithdrawCardNumber{
		CardNumber: s.cardNumber,
		Page:       1,
		PageSize:   10,
	})
	s.NoError(err)
	s.NotNil(byCard)
	s.GreaterOrEqual(*totalCard, 1)
}

func (s *WithdrawServiceTestSuite) Test3_TrashAndRestore() {
	ctx := context.Background()
	s.Require().NotZero(s.withdrawID)

	// Trash
	trashed, err := s.withdrawService.TrashedWithdraw(ctx, s.withdrawID)
	s.NoError(err)
	s.NotNil(trashed)

	// FindByTrashed
	trashedList, totalTrashed, err := s.withdrawService.FindByTrashed(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
	})
	s.NoError(err)
	s.NotNil(trashedList)
	s.GreaterOrEqual(*totalTrashed, 1)

	// Restore
	restored, err := s.withdrawService.RestoreWithdraw(ctx, s.withdrawID)
	s.NoError(err)
	s.NotNil(restored)
}

func (s *WithdrawServiceTestSuite) Test4_BulkOperations() {
	ctx := context.Background()

	// Restore All
	ok, err := s.withdrawService.RestoreAllWithdraw(ctx)
	s.NoError(err)
	s.True(ok)

	// Delete All Permanent
	ok, err = s.withdrawService.DeleteAllWithdrawPermanent(ctx)
	s.NoError(err)
	s.True(ok)
}

func TestWithdrawServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(WithdrawServiceTestSuite))
}

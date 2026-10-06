package saldo_test

import (
	"context"
	"testing"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	tests "github.com/MamangRust/monolith-payment-gateway-test"

	"github.com/MamangRust/monolith-payment-gateway-saldo/repository"
	"github.com/MamangRust/monolith-payment-gateway-saldo/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SaldoStatsServiceTestSuite struct {
	suite.Suite
	gormDB     *gorm.DB
	ts         *tests.TestSuite
	svc        service.Service
	userID     int32
	cardNumber string
	testYear   int
}

func (s *SaldoStatsServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	s.gormDB = gormDB
	repos := repository.NewRepositories(gormDB, nil, nil)

	zapLog := zap.NewNop()
	myLogger := &logger.Logger{Log: zapLog}

	redisOption, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	redisClient := redis.NewClient(redisOption)
	cacheStore := cache.NewCacheStore(redisClient, myLogger, &dummyCacheMetrics{})

	s.svc = service.NewService(&service.Deps{
		Repositories: repos,
		Logger:       myLogger,
		Cache:        cacheStore,
	})

	s.testYear = time.Now().Year()

	ctx := context.Background()
	err = s.gormDB.WithContext(ctx).Raw("INSERT INTO users (firstname, lastname, email, password, verification_code, is_verified) VALUES ('SaldoService', 'Stats', 'saldo_svc_stats@example.com', 'pass', '123', true) RETURNING user_id").Scan(&s.userID).Error
	s.Require().NoError(err)

	s.cardNumber = "4444555566667777"
	err = s.gormDB.WithContext(ctx).Exec("INSERT INTO cards (user_id, card_number, card_type, cvv, card_provider, expire_date) VALUES (?, ?, 'debit', '123', 'visa', '2030-01-01')", s.userID, s.cardNumber).Error
	s.Require().NoError(err)

	// Seed Saldo with balance history
	err = s.gormDB.WithContext(ctx).Exec("INSERT INTO saldos (card_number, total_balance) VALUES (?, ?)", s.cardNumber, 1000000).Error
	s.Require().NoError(err)
}

func (s *SaldoStatsServiceTestSuite) TearDownSuite() {
	s.ts.Teardown()
}

func (s *SaldoStatsServiceTestSuite) TestSaldoStatsService() {
	ctx := context.Background()

	// Monthly balances
	resMonthly, err := s.svc.FindMonthlySaldoBalances(ctx, s.testYear)
	s.NoError(err)
	s.NotEmpty(resMonthly)

	// Yearly balances
	resYearly, err := s.svc.FindYearlySaldoBalances(ctx, s.testYear)
	s.NoError(err)
	s.NotEmpty(resYearly)
}

func TestSaldoStatsServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoStatsServiceTestSuite))
}

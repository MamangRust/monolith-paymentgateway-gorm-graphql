package saldo_test

import (
	"context"
	"testing"
	"time"

	repository "github.com/MamangRust/monolith-payment-gateway-saldo/repository/stats"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type SaldoStatsRepositoryTestSuite struct {
	suite.Suite
	gormDB   *gorm.DB
	ts       *tests.TestSuite
	repo     repository.SaldoStatsRepository
	testYear int
}

func (s *SaldoStatsRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	s.gormDB = gormDB
	s.repo = repository.NewSaldoStatsRepository(gormDB)
	s.testYear = time.Now().Year()
}

func (s *SaldoStatsRepositoryTestSuite) TearDownSuite() {
	s.ts.Teardown()
}

func (s *SaldoStatsRepositoryTestSuite) TestBalanceStats() {
	ctx := context.Background()

	// Seed data
	var userID int32
	err := s.gormDB.WithContext(ctx).Raw("INSERT INTO users (firstname, lastname, email, password, verification_code, is_verified) VALUES ('Saldo', 'Stats', 'saldo_stats_balance@example.com', 'pass', '123', true) RETURNING user_id").Scan(&userID).Error
	s.Require().NoError(err)

	cardNumber := "1111222233334444"
	err = s.gormDB.WithContext(ctx).Exec("INSERT INTO cards (user_id, card_number, card_type, cvv, card_provider, expire_date) VALUES (?, ?, 'debit', '123', 'visa', '2030-01-01')", userID, cardNumber).Error
	s.Require().NoError(err)

	err = s.gormDB.WithContext(ctx).Exec("INSERT INTO saldos (card_number, total_balance, created_at) VALUES (?, ?, ?)",
		cardNumber, 50000, time.Date(s.testYear, 1, 15, 10, 0, 0, 0, time.UTC)).Error
	s.Require().NoError(err)

	// Monthly Global
	res, err := s.repo.GetMonthlySaldoBalances(ctx, s.testYear)
	s.NoError(err)
	s.NotEmpty(res)

	// Yearly Global (5 years)
	yearlyRes, err := s.repo.GetYearlySaldoBalances(ctx, s.testYear)
	s.NoError(err)
	s.Len(yearlyRes, 5)

}

func (s *SaldoStatsRepositoryTestSuite) TestTotalStats() {
	ctx := context.Background()

	// Monthly Total Balance (2 periods)
	req := &requests.MonthTotalSaldoBalance{
		Year:  s.testYear,
		Month: int(time.Now().Month()),
	}
	res, err := s.repo.GetMonthlyTotalSaldoBalance(ctx, req)
	s.NoError(err)
	s.Len(res, 2)

	// Yearly Total Balance (2 years)
	yearlyRes, err := s.repo.GetYearTotalSaldoBalance(ctx, s.testYear)
	s.NoError(err)
	s.NotEmpty(yearlyRes)
}

func TestSaldoStatsRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoStatsRepositoryTestSuite))
}

package withdraw_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	card_repo "github.com/MamangRust/monolith-payment-gateway-card/repository"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	user_repo "github.com/MamangRust/monolith-payment-gateway-user/repository"
	"github.com/MamangRust/monolith-payment-gateway-withdraw/repository"

	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type WithdrawRepositoryTestSuite struct {
	suite.Suite
	gormDB     *gorm.DB
	ts         *tests.TestSuite
	repo       repository.Repositories
	cardRepo   *card_repo.Repositories
	userRepo   *user_repo.Repositories
	userClient *tests.UserClient
	userID     int
}

func (s *WithdrawRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	s.Require().NoError(err)

	userClient, err := tests.NewUserClient(gormDB, s.ts)
	s.Require().NoError(err)
	s.userClient = userClient

	s.userRepo = user_repo.NewRepositories(&user_repo.Deps{Db: gormDB})
	s.cardRepo = card_repo.NewRepositories(gormDB, userClient.Query, card_repo.GuardOptions{User: userClient.Guard()})
	// Withdraw repository methods also mostly use db queries directly
	s.repo = repository.NewRepositories(gormDB, nil, nil, nil, nil, 0)

	// Create user
	user, err := s.userRepo.UserCommand.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Withdraw",
		LastName:  "Tester",
		Email:     fmt.Sprintf("withdraw.tester-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)
}

func (s *WithdrawRepositoryTestSuite) TearDownSuite() {
	if s.userClient != nil {
		s.userClient.Close()
	}
	s.ts.Teardown()
}

func (s *WithdrawRepositoryTestSuite) createSeedWithdraw() (*models.WithdrawAllFieldsRow, error) {
	card, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "123",
		CardProvider: "Visa",
	})
	if err != nil {
		return nil, err
	}

	return s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 100000,
		WithdrawTime:   time.Now(),
	})
}

func (s *WithdrawRepositoryTestSuite) TestCreateWithdraw() {
	ctx := context.Background()

	card, _ := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "123",
		CardProvider: "Visa",
	})

	req := &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 100000,
		WithdrawTime:   time.Now(),
	}

	res, err := s.repo.CreateWithdraw(ctx, req)
	s.NoError(err)
	s.NotNil(res)
}

func (s *WithdrawRepositoryTestSuite) TestFindAllWithdraws() {
	_, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindAll(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestFindById() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.repo.FindById(ctx, int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(withdraw.WithdrawID, found.WithdrawID)
}

func (s *WithdrawRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindByActive(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestFindByTrashed() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedWithdraw(ctx, int(withdraw.WithdrawID))
	s.Require().NoError(err)

	res, err := s.repo.FindByTrashed(ctx, &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestUpdateWithdraw() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(withdraw.WithdrawID)
	req := &requests.UpdateWithdrawRequest{
		WithdrawID:     &id,
		CardNumber:     withdraw.CardNumber,
		WithdrawAmount: 200000,
		WithdrawTime:   time.Now(),
	}

	res, err := s.repo.UpdateWithdraw(ctx, req)
	s.NoError(err)
	s.NotNil(res)
}

func (s *WithdrawRepositoryTestSuite) TestTrashWithdraw() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.TrashedWithdraw(ctx, int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(trashed)
}

func (s *WithdrawRepositoryTestSuite) TestRestoreWithdraw() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedWithdraw(ctx, int(withdraw.WithdrawID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreWithdraw(ctx, int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(restored)
}

func (s *WithdrawRepositoryTestSuite) TestDeleteWithdrawPermanent() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedWithdraw(ctx, int(withdraw.WithdrawID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteWithdrawPermanent(ctx, int(withdraw.WithdrawID))
	s.NoError(err)
	s.True(success)
}

func (s *WithdrawRepositoryTestSuite) TestRestoreAllWithdraw() {
	withdraw, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedWithdraw(ctx, int(withdraw.WithdrawID))
	s.Require().NoError(err)

	success, err := s.repo.RestoreAllWithdraw(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *WithdrawRepositoryTestSuite) TestDeleteAllWithdrawPermanent() {
	_, err := s.createSeedWithdraw()
	s.Require().NoError(err)
	ctx := context.Background()

	success, err := s.repo.DeleteAllWithdrawPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestWithdrawRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(WithdrawRepositoryTestSuite))
}

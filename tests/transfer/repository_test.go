package transfer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	card_repo "github.com/MamangRust/monolith-payment-gateway-card/repository"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	"github.com/MamangRust/monolith-payment-gateway-transfer/repository"
	user_repo "github.com/MamangRust/monolith-payment-gateway-user/repository"

	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TransferRepositoryTestSuite struct {
	suite.Suite
	gormDB   *gorm.DB
	ts       *tests.TestSuite
	repo     repository.Repositories
	cardRepo *card_repo.Repositories
	userRepo user_repo.Repositories
	userID   int
}

func (s *TransferRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	s.Require().NoError(err)

	s.userRepo = user_repo.NewRepositories(gormDB)
	s.cardRepo = card_repo.NewRepositories(gormDB)
	s.repo = repository.NewRepositories(gormDB, nil, nil)

	// Create user
	user, err := s.userRepo.UserCommand().CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Transfer",
		LastName:  "Tester",
		Email:     fmt.Sprintf("transfer.tester-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)
}

func (s *TransferRepositoryTestSuite) TearDownSuite() {
	s.ts.Teardown()
}

func (s *TransferRepositoryTestSuite) createSeedTransfer() (*models.TransferAllFieldsRow, error) {
	fromCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "111",
		CardProvider: "Visa",
	})
	if err != nil {
		return nil, err
	}

	toCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "222",
		CardProvider: "MasterCard",
	})
	if err != nil {
		return nil, err
	}

	res, err := s.repo.CreateTransferAtomic(context.Background(), &requests.CreateTransferRequest{
		TransferFrom:   fromCard.CardNumber,
		TransferTo:     toCard.CardNumber,
		TransferAmount: 100000,
	})
	if err != nil {
		return nil, err
	}
	return res.Row, nil
}

func (s *TransferRepositoryTestSuite) TestCreateTransfer() {
	ctx := context.Background()

	fromCard, _ := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "111",
		CardProvider: "Visa",
	})

	toCard, _ := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "222",
		CardProvider: "MasterCard",
	})

	req := &requests.CreateTransferRequest{
		TransferFrom:   fromCard.CardNumber,
		TransferTo:     toCard.CardNumber,
		TransferAmount: 100000,
	}

	res, err := s.repo.CreateTransferAtomic(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.NotNil(res.Row)
}

func (s *TransferRepositoryTestSuite) TestFindAllTransfers() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindAll(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindById() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.repo.FindById(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(transfer.TransferID, found.TransferID)
}

func (s *TransferRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindByActive(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindByTrashed() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	res, err := s.repo.FindByTrashed(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestUpdateTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(transfer.TransferID)
	req := &requests.UpdateTransferRequest{
		TransferID:     &id,
		TransferFrom:   transfer.TransferFrom,
		TransferTo:     transfer.TransferTo,
		TransferAmount: 200000,
	}

	res, err := s.repo.UpdateTransfer(ctx, req)
	s.NoError(err)
	s.NotNil(res)
}

func (s *TransferRepositoryTestSuite) TestTrashTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(trashed)
}

func (s *TransferRepositoryTestSuite) TestRestoreTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreTransfer(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(restored)
}

func (s *TransferRepositoryTestSuite) TestDeleteTransferPermanent() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteTransferPermanent(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.True(success)
}

func (s *TransferRepositoryTestSuite) TestRestoreAllTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	success, err := s.repo.RestoreAllTransfer(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *TransferRepositoryTestSuite) TestDeleteAllTransferPermanent() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	success, err := s.repo.DeleteAllTransferPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestTransferRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransferRepositoryTestSuite))
}

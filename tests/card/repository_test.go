package card_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-card/repository"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	tests "github.com/MamangRust/monolith-payment-gateway-test"
	user_repo "github.com/MamangRust/monolith-payment-gateway-user/repository"

	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type CardRepositoryTestSuite struct {
	suite.Suite
	gormDB     *gorm.DB
	ts         *tests.TestSuite
	repo       *repository.Repositories
	userRepo   *user_repo.Repositories
	userClient *tests.UserClient
	userID     int
}

func (s *CardRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	gormDB, err := s.ts.GormDB()
	s.Require().NoError(err)
	s.Require().NoError(err)

	userClient, err := tests.NewUserClient(gormDB, s.ts)
	s.Require().NoError(err)
	s.userClient = userClient

	s.repo = repository.NewRepositories(gormDB, userClient.Query, repository.GuardOptions{User: userClient.Guard()})
	s.userRepo = user_repo.NewRepositories(&user_repo.Deps{Db: gormDB})

	// Create a user for card ownership
	user, err := s.userRepo.UserCommand.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Card",
		LastName:  "Owner",
		Email:     fmt.Sprintf("card.owner-%d-%d@example.com", time.Now().UnixNano(), time.Now().UnixNano()%10000),
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)
}

func (s *CardRepositoryTestSuite) TearDownSuite() {
	if s.userClient != nil {
		s.userClient.Close()
	}
	s.ts.Teardown()
}

func (s *CardRepositoryTestSuite) createSeedCard() (*models.CardCreateRow, error) {
	return s.repo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "123",
		CardProvider: "Visa",
	})
}

func (s *CardRepositoryTestSuite) TestCreateCard() {
	ctx := context.Background()
	req := &requests.CreateCardRequest{
		UserID:       s.userID,
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(5, 0, 0),
		CVV:          "123",
		CardProvider: "Visa",
	}

	res, err := s.repo.CardCommand.CreateCard(ctx, req)
	s.NoError(err)
	s.NotNil(res)
}

func (s *CardRepositoryTestSuite) TestFindAllCards() {
	_, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.CardQuery.FindAllCards(ctx, &requests.FindAllCards{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *CardRepositoryTestSuite) TestFindById() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.repo.CardQuery.FindById(ctx, int(card.CardID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(card.CardID, found.CardID)
}

func (s *CardRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.CardQuery.FindByActive(ctx, &requests.FindAllCards{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *CardRepositoryTestSuite) TestFindByTrashed() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.CardCommand.TrashedCard(ctx, int(card.CardID))
	s.Require().NoError(err)

	res, err := s.repo.CardQuery.FindByTrashed(ctx, &requests.FindAllCards{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *CardRepositoryTestSuite) TestUpdateCard() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	req := &requests.UpdateCardRequest{
		CardID:       int(card.CardID),
		UserID:       s.userID,
		CardType:     "credit",
		ExpireDate:   time.Now().AddDate(6, 0, 0),
		CVV:          "456",
		CardProvider: "MasterCard",
	}

	res, err := s.repo.CardCommand.UpdateCard(ctx, req)
	s.NoError(err)
	s.NotNil(res)
}

func (s *CardRepositoryTestSuite) TestTrashCard() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.CardCommand.TrashedCard(ctx, int(card.CardID))
	s.NoError(err)
	s.NotNil(trashed)
}

func (s *CardRepositoryTestSuite) TestRestoreCard() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.CardCommand.TrashedCard(ctx, int(card.CardID))
	s.Require().NoError(err)

	restored, err := s.repo.CardCommand.RestoreCard(ctx, int(card.CardID))
	s.NoError(err)
	s.NotNil(restored)
}

func (s *CardRepositoryTestSuite) TestDeleteCardPermanent() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.CardCommand.TrashedCard(ctx, int(card.CardID))
	s.Require().NoError(err)

	success, err := s.repo.CardCommand.DeleteCardPermanent(ctx, int(card.CardID))
	s.NoError(err)
	s.True(success)
}

func (s *CardRepositoryTestSuite) TestRestoreAllCard() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.CardCommand.TrashedCard(ctx, int(card.CardID))
	s.Require().NoError(err)

	success, err := s.repo.CardCommand.RestoreAllCard(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *CardRepositoryTestSuite) TestDeleteAllCardPermanent() {
	_, err := s.createSeedCard()
	s.Require().NoError(err)
	ctx := context.Background()

	success, err := s.repo.CardCommand.DeleteAllCardPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestCardRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CardRepositoryTestSuite))
}

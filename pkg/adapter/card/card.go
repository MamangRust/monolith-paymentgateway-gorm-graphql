// Package card adapts the Card service gRPC API into the shared domain model.
package card

import (
	"context"
	"time"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryRepository is the contract consumers depend on for card reads.
type QueryRepository interface {
	FindCardByUserId(ctx context.Context, user_id int) (*models.CardAllFieldsRow, error)
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*models.CardByEmailRow, error)
	FindCardByCardNumber(ctx context.Context, card_number string) (*models.CardAllFieldsRow, error)
}

// CommandRepository is the contract consumers depend on for card writes.
type CommandRepository interface {
	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*models.CardUpdateRow, error)
	AddRewardPoints(ctx context.Context, cardID int, points int) (*models.AddRewardPointsRow, error)
	UpdateCardOutstandingBalance(ctx context.Context, cardID int, outstandingBalance int) (*models.UpdateOutstandingBalanceRow, error)
	UpdateCardStatus(ctx context.Context, cardID int, status string) (*models.UpdateCardStatusRow, error)
}

// Repository implements both QueryRepository and CommandRepository on top of the
// generated card query/command clients.
type Repository struct {
	query   pbcard.CardQueryServiceClient
	command pbcard.CardCommandServiceClient
	guard   *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (a *Repository) SetGuard(g *resilience.DependencyGuard) { a.guard = g }

// NewAdapter wraps the generated card clients. Resilience (timeout/circuit-breaker/
// bulkhead) is applied per call through the optional dependency guard; a nil
// guard is a passthrough.
func NewAdapter(query pbcard.CardQueryServiceClient, command pbcard.CardCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	a := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// New wraps the generated card query/command clients. It is the canonical
// constructor; modules build it inside their repositories with the guard
// options they were given.
func New(query pbcard.CardQueryServiceClient, command pbcard.CardCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(query, command, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface
// for consumers that only read.
func NewQueryAdapter(query pbcard.CardQueryServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, nil, opts...)
}

// NewCommandAdapter returns the adapter restricted to the CommandRepository
// surface for consumers that only write.
func NewCommandAdapter(command pbcard.CardCommandServiceClient, opts ...adapter.GuardOption) CommandRepository {
	return NewAdapter(nil, command, opts...)
}

func (a *Repository) FindCardByUserId(ctx context.Context, user_id int) (*models.CardAllFieldsRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.ApiResponseCard, error) {
		return a.query.FindByUserIdCard(ctx, &pbcard.FindByUserIdCardRequest{
			UserId: int32(user_id),
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.CardAllFieldsRow{
		CardID:       res.Data.Id,
		UserID:       res.Data.UserId,
		CardNumber:   res.Data.CardNumber,
		CardType:     res.Data.CardType,
		ExpireDate:   parseTime(res.Data.ExpireDate),
		Cvv:          res.Data.Cvv,
		CardProvider: res.Data.CardProvider,
		Status:       res.Data.Status,
	}, nil
}

func (a *Repository) FindUserCardByCardNumber(ctx context.Context, card_number string) (*models.CardByEmailRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.CardWithEmailResponse, error) {
		return a.query.FindUserCardByCardNumber(ctx, &pbcard.FindByCardNumberRequest{
			CardNumber: card_number,
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.CardByEmailRow{
		CardID:       res.Id,
		UserID:       res.UserId,
		CardNumber:   res.CardNumber,
		CardType:     res.CardType,
		ExpireDate:   parseTime(res.ExpireDate),
		Cvv:          res.Cvv,
		CardProvider: res.CardProvider,
		Email:        res.Email,
	}, nil
}

func (a *Repository) FindCardByCardNumber(ctx context.Context, card_number string) (*models.CardAllFieldsRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.ApiResponseCard, error) {
		return a.query.FindByCardNumber(ctx, &pbcard.FindByCardNumberRequest{
			CardNumber: card_number,
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.CardAllFieldsRow{
		CardID:       res.Data.Id,
		UserID:       res.Data.UserId,
		CardNumber:   res.Data.CardNumber,
		CardType:     res.Data.CardType,
		ExpireDate:   parseTime(res.Data.ExpireDate),
		Cvv:          res.Data.Cvv,
		CardProvider: res.Data.CardProvider,
		Status:       res.Data.Status,
	}, nil
}

func (a *Repository) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*models.CardUpdateRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.ApiResponseCard, error) {
		return a.command.UpdateCard(ctx, &pbcard.UpdateCardRequest{
			CardId:       int32(request.CardID),
			UserId:       int32(request.UserID),
			CardType:     request.CardType,
			ExpireDate:   timestamppb.New(request.ExpireDate),
			Cvv:          request.CVV,
			CardProvider: request.CardProvider,
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.CardUpdateRow{
		CardID:       res.Data.Id,
		UserID:       res.Data.UserId,
		CardNumber:   res.Data.CardNumber,
		CardType:     res.Data.CardType,
		ExpireDate:   parseTime(res.Data.ExpireDate),
		Cvv:          res.Data.Cvv,
		CardProvider: res.Data.CardProvider,
	}, nil
}

func (a *Repository) AddRewardPoints(ctx context.Context, cardID int, points int) (*models.AddRewardPointsRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.ApiResponseCard, error) {
		return a.command.UpdateCreditLimit(ctx, &pbcard.UpdateCreditLimitRequest{
			CardId:      int32(cardID),
			CreditLimit: int32(points),
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.AddRewardPointsRow{
		CardID:             res.Data.Id,
		UserID:             res.Data.UserId,
		CardNumber:         res.Data.CardNumber,
		CardType:           res.Data.CardType,
		ExpireDate:         parseTime(res.Data.ExpireDate),
		Cvv:                res.Data.Cvv,
		CardProvider:       res.Data.CardProvider,
		Status:             res.Data.Status,
		CreditLimit:        res.Data.CreditLimit,
		OutstandingBalance: res.Data.OutstandingBalance,
		RewardPoints:       res.Data.RewardPoints,
	}, nil
}

func (a *Repository) UpdateCardOutstandingBalance(ctx context.Context, cardID int, outstandingBalance int) (*models.UpdateOutstandingBalanceRow, error) {
	return &models.UpdateOutstandingBalanceRow{}, nil
}

func (a *Repository) UpdateCardStatus(ctx context.Context, cardID int, status string) (*models.UpdateCardStatusRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbcard.ApiResponseCard, error) {
		return a.command.ToggleCardStatus(ctx, &pbcard.ToggleCardStatusRequest{
			CardId: int32(cardID),
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.UpdateCardStatusRow{
		CardID:             res.Data.Id,
		UserID:             res.Data.UserId,
		CardNumber:         res.Data.CardNumber,
		CardType:           res.Data.CardType,
		ExpireDate:         parseTime(res.Data.ExpireDate),
		Cvv:                res.Data.Cvv,
		CardProvider:       res.Data.CardProvider,
		Status:             res.Data.Status,
		CreditLimit:        res.Data.CreditLimit,
		OutstandingBalance: res.Data.OutstandingBalance,
		RewardPoints:       res.Data.RewardPoints,
	}, nil
}

// parseTime accepts the RFC3339 timestamps the card service emits and returns
// the zero time for empty or unparseable values.
func parseTime(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

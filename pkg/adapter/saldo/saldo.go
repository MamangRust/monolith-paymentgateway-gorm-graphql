// Package saldo adapts the Saldo service gRPC API into the shared domain model.
package saldo

import (
	"context"
	"time"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
)

// QueryRepository is the contract consumers depend on for saldo reads.
type QueryRepository interface {
	FindByCardNumber(ctx context.Context, card_number string) (*models.Saldo, error)
}

// CommandRepository is the contract consumers depend on for saldo writes.
type CommandRepository interface {
	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*models.UpdateSaldoBalanceRow, error)
	UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*models.UpdateSaldoWithdrawRow, error)
}

// Repository implements both QueryRepository and CommandRepository on top of the
// generated saldo query/command clients.
type Repository struct {
	query   pbsaldo.SaldoQueryServiceClient
	command pbsaldo.SaldoCommandServiceClient
	guard   *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (a *Repository) SetGuard(g *resilience.DependencyGuard) { a.guard = g }

// NewAdapter wraps the generated saldo clients. Resilience (timeout/circuit-breaker/
// bulkhead) is applied per call through the optional dependency guard; a nil
// guard is a passthrough.
func NewAdapter(query pbsaldo.SaldoQueryServiceClient, command pbsaldo.SaldoCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	a := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// New wraps the generated saldo query/command clients. It is the canonical
// constructor; modules build it inside their repositories with the guard
// options they were given.
func New(query pbsaldo.SaldoQueryServiceClient, command pbsaldo.SaldoCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(query, command, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface
// for consumers that only read.
func NewQueryAdapter(query pbsaldo.SaldoQueryServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, nil, opts...)
}

// NewCommandAdapter returns the adapter restricted to the CommandRepository
// surface for consumers that only write.
func NewCommandAdapter(command pbsaldo.SaldoCommandServiceClient, opts ...adapter.GuardOption) CommandRepository {
	return NewAdapter(nil, command, opts...)
}

func (a *Repository) FindByCardNumber(ctx context.Context, card_number string) (*models.Saldo, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbsaldo.ApiResponseSaldo, error) {
		return a.query.FindByCardNumber(ctx, &pbcard.FindByCardNumberRequest{
			CardNumber: card_number,
		})
	})
	if err != nil {
		return nil, err
	}

	return toSaldo(res.Data), nil
}

func (a *Repository) UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*models.UpdateSaldoBalanceRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbsaldo.ApiResponseSaldo, error) {
		return a.command.UpdateSaldoBalance(ctx, &pbsaldo.UpdateSaldoBalanceRequest{
			CardNumber:   request.CardNumber,
			TotalBalance: int32(request.TotalBalance),
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.UpdateSaldoBalanceRow{
		SaldoID:      res.Data.SaldoId,
		CardNumber:   res.Data.CardNumber,
		TotalBalance: res.Data.TotalBalance,
	}, nil
}

func (a *Repository) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*models.UpdateSaldoWithdrawRow, error) {
	withdrawTime := ""
	if request.WithdrawTime != nil {
		withdrawTime = request.WithdrawTime.Format(time.RFC3339)
	}

	withdrawAmount := int32(0)
	if request.WithdrawAmount != nil {
		withdrawAmount = int32(*request.WithdrawAmount)
	}

	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbsaldo.ApiResponseSaldo, error) {
		return a.command.UpdateSaldoWithdraw(ctx, &pbsaldo.UpdateSaldoWithdrawRequest{
			CardNumber:     request.CardNumber,
			TotalBalance:   int32(request.TotalBalance),
			WithdrawTime:   withdrawTime,
			WithdrawAmount: withdrawAmount,
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.UpdateSaldoWithdrawRow{
		SaldoID:        res.Data.SaldoId,
		CardNumber:     res.Data.CardNumber,
		TotalBalance:   res.Data.TotalBalance,
		WithdrawAmount: &res.Data.WithdrawAmount,
		WithdrawTime:   parseTimePtr(res.Data.WithdrawTime),
		CreatedAt:      parseTime(res.Data.CreatedAt),
		UpdatedAt:      parseTime(res.Data.UpdatedAt),
	}, nil
}

func toSaldo(s *pbsaldo.SaldoResponse) *models.Saldo {
	if s == nil {
		return nil
	}

	saldo := &models.Saldo{
		SaldoID:      s.SaldoId,
		CardNumber:   s.CardNumber,
		TotalBalance: s.TotalBalance,
	}

	if s.WithdrawAmount != 0 {
		wa := s.WithdrawAmount
		saldo.WithdrawAmount = &wa
	}

	saldo.WithdrawTime = parseTimePtr(s.WithdrawTime)
	saldo.CreatedAt = parseTime(s.CreatedAt)
	saldo.UpdatedAt = parseTime(s.UpdatedAt)

	return saldo
}

// parseTime accepts the RFC3339 timestamps the saldo service emits and returns
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

// parseTimePtr is parseTime for optional timestamps.
func parseTimePtr(ts string) *time.Time {
	if ts == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil
	}
	return &t
}

package repository

import (
	"gorm.io/gorm"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	cardadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/card"
	saldoadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/saldo"
	withdrawstatsrepository "github.com/MamangRust/monolith-payment-gateway-withdraw/repository/stats"
	withdrawstatsbycardrepository "github.com/MamangRust/monolith-payment-gateway-withdraw/repository/statsbycard"
)

// GuardOptions carries the resilience guard options for each outbound
// dependency. Callers build them with adapter.WithDependencyGuard (typically
// around resilience.NewDependencyGuard) so the repository owns the wiring.
type GuardOptions struct {
	Card  []adapter.GuardOption
	Saldo []adapter.GuardOption
}

type Repositories interface {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
	withdrawstatsrepository.WithdrawStatsRepository
	withdrawstatsbycardrepository.WithdrawStatsByCardRepository
}

type repositories struct {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
	withdrawstatsrepository.WithdrawStatsRepository
	withdrawstatsbycardrepository.WithdrawStatsByCardRepository
}

func NewRepositories(
	db *gorm.DB,
	cardQuery pbcard.CardQueryServiceClient,
	cardCommand pbcard.CardCommandServiceClient,
	saldoQuery pbsaldo.SaldoQueryServiceClient,
	saldoCommand pbsaldo.SaldoCommandServiceClient,
	dailyLimit int64,
	guards ...GuardOptions,
) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		CardRepository:                cardadapter.NewAdapter(cardQuery, cardCommand, g.Card...),
		SaldoRepository:               saldoadapter.NewAdapter(saldoQuery, saldoCommand, g.Saldo...),
		WithdrawQueryRepository:       NewWithdrawQueryRepository(db),
		WithdrawCommandRepository:     NewWithdrawCommandRepository(db, dailyLimit),
		WithdrawStatsRepository:       withdrawstatsrepository.NewWithdrawStatsRepository(db),
		WithdrawStatsByCardRepository: withdrawstatsbycardrepository.NewWithdrawStatsByCardRepository(db),
	}
}

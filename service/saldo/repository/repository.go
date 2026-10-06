package repository

import (
	"gorm.io/gorm"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	cardadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/card"
	saldostatsrepository "github.com/MamangRust/monolith-payment-gateway-saldo/repository/stats"
)

// GuardOptions configures the guarded gRPC adapters used by the saldo
// repositories (e.g. the card adapter).
type GuardOptions struct {
	Card []adapter.GuardOption
}

// Repositories is a struct containing all saldo repositories.
type Repositories interface {
	SaldoQueryRepository
	SaldoCommandRepository
	saldostatsrepository.SaldoStatsRepository
	CardRepository
}

type repositories struct {
	SaldoQueryRepository
	SaldoCommandRepository
	saldostatsrepository.SaldoStatsRepository
	CardRepository
}

func NewRepositories(
	db *gorm.DB,
	cardQuery pbcard.CardQueryServiceClient,
	cardCommand pbcard.CardCommandServiceClient,
	guards ...GuardOptions,
) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		SaldoQueryRepository:   NewSaldoQueryRepository(db),
		SaldoCommandRepository: NewSaldoCommandRepository(db),
		SaldoStatsRepository:   saldostatsrepository.NewSaldoStatsRepository(db),
		CardRepository:         cardadapter.NewAdapter(cardQuery, cardCommand, g.Card...),
	}
}

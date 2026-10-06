package repository

import (
	"gorm.io/gorm"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	cardadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/card"
	saldoadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/saldo"
	topupstatsrepository "github.com/MamangRust/monolith-payment-gateway-topup/repository/stats"
	topupstatsbycardrepository "github.com/MamangRust/monolith-payment-gateway-topup/repository/statsbycard"
)

// GuardOptions carries the resilience guard options for each outbound
// dependency. Callers build them with adapter.WithDependencyGuard (typically
// around resilience.NewDependencyGuard) so the repository owns the wiring.
type GuardOptions struct {
	Card  []adapter.GuardOption
	Saldo []adapter.GuardOption
}

type Repositories interface {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
	topupstatsrepository.TopupStatsRepository
	topupstatsbycardrepository.TopupStatsByCardRepository
}

type repositories struct {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
	topupstatsrepository.TopupStatsRepository
	topupstatsbycardrepository.TopupStatsByCardRepository
}

func NewRepositories(
	db *gorm.DB,
	cardQuery pbcard.CardQueryServiceClient,
	cardCommand pbcard.CardCommandServiceClient,
	saldoQuery pbsaldo.SaldoQueryServiceClient,
	saldoCommand pbsaldo.SaldoCommandServiceClient,
	guards ...GuardOptions,
) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		TopupQueryRepository:       NewTopupQueryRepository(db),
		TopupCommandRepository:     NewTopupCommandRepository(db),
		TopupStatsRepository:       topupstatsrepository.NewTopupStatsRepository(db),
		TopupStatsByCardRepository: topupstatsbycardrepository.NewTopupStatsByCardRepository(db),
		CardRepository:             cardadapter.NewAdapter(cardQuery, cardCommand, g.Card...),
		SaldoRepository:            saldoadapter.NewAdapter(saldoQuery, saldoCommand, g.Saldo...),
	}
}

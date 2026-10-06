package repository

import (
	"gorm.io/gorm"

	pbcard "github.com/MamangRust/monolith-payment-gateway-pb/card"
	pbmerchant "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	pbsaldo "github.com/MamangRust/monolith-payment-gateway-pb/saldo"
	"github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	cardadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/card"
	merchantadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/merchant"
	saldoadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/saldo"
	transactionstatsrepository "github.com/MamangRust/monolith-payment-gateway-transaction/repository/stats"
	transactionbycardrepository "github.com/MamangRust/monolith-payment-gateway-transaction/repository/statsbycard"
)

// GuardOptions carries the resilience guard options for each outbound
// dependency. Callers build them with adapter.WithDependencyGuard (typically
// around resilience.NewDependencyGuard) so the repository owns the wiring.
type GuardOptions struct {
	Saldo    []adapter.GuardOption
	Card     []adapter.GuardOption
	Merchant []adapter.GuardOption
}

type Repositories interface {
	SaldoRepository
	MerchantRepository
	CardRepository
	TransactionQueryRepository
	TransactionCommandRepository
	transactionstatsrepository.TransactionStatsRepository
	transactionbycardrepository.TransactionStatsByCardRepository
}

type repositories struct {
	SaldoRepository
	MerchantRepository
	CardRepository
	TransactionQueryRepository
	TransactionCommandRepository
	transactionstatsrepository.TransactionStatsRepository
	transactionbycardrepository.TransactionStatsByCardRepository
}

func NewRepositories(
	db *gorm.DB,
	saldoQuery pbsaldo.SaldoQueryServiceClient,
	saldoCommand pbsaldo.SaldoCommandServiceClient,
	cardQuery pbcard.CardQueryServiceClient,
	cardCommand pbcard.CardCommandServiceClient,
	merchantQuery pbmerchant.MerchantQueryServiceClient,
	guards ...GuardOptions,
) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		SaldoRepository:                  saldoadapter.NewAdapter(saldoQuery, saldoCommand, g.Saldo...),
		MerchantRepository:               merchantadapter.NewAdapter(merchantQuery, g.Merchant...),
		CardRepository:                   cardadapter.NewAdapter(cardQuery, cardCommand, g.Card...),
		TransactionQueryRepository:       NewTransactionQueryRepository(db),
		TransactionCommandRepository:     NewTransactionCommandRepository(db),
		TransactionStatsRepository:       transactionstatsrepository.NewTransactionStatsRepository(db),
		TransactionStatsByCardRepository: transactionbycardrepository.NewTransactionStatsByCardRepository(db),
	}
}

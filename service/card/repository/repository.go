package repository

import (
	repositorydashboard "github.com/MamangRust/monolith-payment-gateway-card/repository/dashboard"
	repositorystats "github.com/MamangRust/monolith-payment-gateway-card/repository/stats"
	repositorystatsbycard "github.com/MamangRust/monolith-payment-gateway-card/repository/statsbycard"
	pbuser "github.com/MamangRust/monolith-payment-gateway-pb/user"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	useradapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user"
	"gorm.io/gorm"
)

// GuardOptions carries the resilience guard options for each outbound
// dependency. Callers build them with adapter.WithDependencyGuard (typically
// around resilience.NewDependencyGuard) so the repository owns the wiring.
type GuardOptions struct {
	User []adapter.GuardOption
}

// Repositories contains all the repositories used in the application
type Repositories struct {
	CardCommand         CardCommandRepository
	CardQuery           CardQueryRepository
	CardDashboard       repositorydashboard.CardDashboardRepository
	CardStatistic       repositorystats.CardStatsRepository
	CardStatisticByCard repositorystatsbycard.CardStatsByCardRepository
	User                UserRepository
	CardAuthTransaction CardAuthTransactionRepository
	CardPayment         CardPaymentRepository
	CardReward          CardRewardRepository
	BillingCycle        BillingCycleRepository
}

func NewRepositories(db *gorm.DB, userQueryClient pbuser.UserQueryServiceClient, guards ...GuardOptions) *Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &Repositories{
		CardQuery:           NewCardQueryRepository(db),
		CardCommand:         NewCardCommandRepository(db),
		CardDashboard:       repositorydashboard.NewCardDashboardRepository(db),
		CardStatistic:       repositorystats.NewCardStatsRepository(db),
		CardStatisticByCard: repositorystatsbycard.NewCardStatsByCardRepository(db),
		User:                useradapter.NewQueryAdapter(userQueryClient, g.User...),
		CardAuthTransaction: NewCardAuthTransactionRepository(db),
		CardPayment:         NewCardPaymentRepository(db),
		CardReward:          NewCardRewardRepository(db),
		BillingCycle:        NewBillingCycleRepository(db),
	}
}

package repository

import (
	merchantstatsrepository "github.com/MamangRust/monolith-payment-gateway-merchant/repository/stats"
	merchantstatsapikeyrepository "github.com/MamangRust/monolith-payment-gateway-merchant/repository/statsbyapikey"
	merchantstatsmerchantrepository "github.com/MamangRust/monolith-payment-gateway-merchant/repository/statsbymerchant"
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

type Repositories interface {
	MerchantQueryRepository
	MerchantCommandRepository
	MerchantDocumentQueryRepository
	MerchantDocumentCommandRepository
	MerchantTransactionRepository
	merchantstatsrepository.MerchantStatsRepository
	merchantstatsapikeyrepository.MerchantStatsByApiKeyRepository
	merchantstatsmerchantrepository.MerchantStatsByMerchantRepository
	UserRepository
}

type repositories struct {
	MerchantQueryRepository
	MerchantCommandRepository
	MerchantDocumentQueryRepository
	MerchantDocumentCommandRepository
	MerchantTransactionRepository
	merchantstatsrepository.MerchantStatsRepository
	merchantstatsapikeyrepository.MerchantStatsByApiKeyRepository
	merchantstatsmerchantrepository.MerchantStatsByMerchantRepository
	UserRepository
}

func NewRepositories(db *gorm.DB, userQueryClient pbuser.UserQueryServiceClient, guards ...GuardOptions) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		NewMerchantQueryRepository(db),
		NewMerchantCommandRepository(db),
		NewMerchantDocumentQueryRepository(db),
		NewMerchantDocumentCommandRepository(db),
		NewMerchantTransactionRepository(db),
		merchantstatsrepository.NewMerchantStatsRepository(db),
		merchantstatsapikeyrepository.NewMerchantStatsByApiKeyRepository(db),
		merchantstatsmerchantrepository.NewMerchantStatsByMerchantRepository(db),
		useradapter.NewQueryAdapter(userQueryClient, g.User...),
	}
}

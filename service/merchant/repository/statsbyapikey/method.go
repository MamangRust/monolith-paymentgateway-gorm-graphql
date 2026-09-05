package merchantstatsapikeyrepository

import (
	"context"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	merchant_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/merchant_errors/repository"
	"gorm.io/gorm"
)

type merchantStatsMethodByApiKeyRepository struct {
	db *gorm.DB
}

func NewMerchantStatsMethodByApiKeyRepository(db *gorm.DB) MerchantStatsMethodByApiKeyRepository {
	return &merchantStatsMethodByApiKeyRepository{db: db}
}

func (r *merchantStatsMethodByApiKeyRepository) GetMonthlyPaymentMethodByApikey(ctx context.Context, req *requests.MonthYearPaymentMethodApiKey) ([]*models.MerchantMonthlyPaymentMethodRow, error) {
	yearStart := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantMonthlyPaymentMethodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(t.transaction_time, 'Mon') AS month, t.payment_method, COALESCE(SUM(t.amount), 0)::int AS total_amount
		FROM transactions t JOIN merchants m ON m.merchant_id = t.merchant_id AND m.deleted_at IS NULL
		WHERE m.api_key = ? AND t.transaction_time >= ? AND t.transaction_time < ? AND t.deleted_at IS NULL
		GROUP BY TO_CHAR(t.transaction_time, 'Mon'), EXTRACT(MONTH FROM t.transaction_time), t.payment_method
		ORDER BY EXTRACT(MONTH FROM t.transaction_time), t.payment_method
	`, req.Apikey, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetMonthlyPaymentMethodByApikeyFailed.WithInternal(err)
	}
	return res, nil
}

func (r *merchantStatsMethodByApiKeyRepository) GetYearlyPaymentMethodByApikey(ctx context.Context, req *requests.MonthYearPaymentMethodApiKey) ([]*models.MerchantYearlyPaymentMethodRow, error) {
	yearStart := time.Date(req.Year-4, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantYearlyPaymentMethodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(YEAR FROM t.transaction_time)::text AS year, t.payment_method, COALESCE(SUM(t.amount), 0)::bigint AS total_amount
		FROM transactions t JOIN merchants m ON m.merchant_id = t.merchant_id AND m.deleted_at IS NULL
		WHERE m.api_key = ? AND t.transaction_time >= ? AND t.transaction_time < ? AND t.deleted_at IS NULL
		GROUP BY EXTRACT(YEAR FROM t.transaction_time), t.payment_method ORDER BY EXTRACT(YEAR FROM t.transaction_time), t.payment_method
	`, req.Apikey, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetYearlyPaymentMethodByApikeyFailed.WithInternal(err)
	}
	return res, nil
}

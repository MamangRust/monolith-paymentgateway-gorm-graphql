package merchantstatsrepository

import (
	"context"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	merchant_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/merchant_errors/repository"
	"gorm.io/gorm"
)

type merchantStatsAmountRepository struct {
	db *gorm.DB
}

func NewMerchantStatsAmountRepository(db *gorm.DB) MerchantStatsAmountRepository {
	return &merchantStatsAmountRepository{db: db}
}

func (r *merchantStatsAmountRepository) GetMonthlyAmountMerchant(ctx context.Context, year int) ([]*models.MerchantMonthlyAmountRow, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantMonthlyAmountRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(transaction_time, 'Mon') AS month, COALESCE(SUM(amount), 0)::int AS total_amount
		FROM transactions WHERE transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY TO_CHAR(transaction_time, 'Mon'), EXTRACT(MONTH FROM transaction_time)
		ORDER BY EXTRACT(MONTH FROM transaction_time)
	`, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetMonthlyAmountMerchantFailed.WithInternal(err)
	}
	return res, nil
}

func (r *merchantStatsAmountRepository) GetYearlyAmountMerchant(ctx context.Context, year int) ([]*models.MerchantYearlyAmountRow, error) {
	var res []*models.MerchantYearlyAmountRow
	yearStart := time.Date(year-4, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(YEAR FROM transaction_time)::text AS year, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM transactions WHERE transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY EXTRACT(YEAR FROM transaction_time) ORDER BY EXTRACT(YEAR FROM transaction_time)
	`, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetYearlyAmountMerchantFailed.WithInternal(err)
	}
	return res, nil
}

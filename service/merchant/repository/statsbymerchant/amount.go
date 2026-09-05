package merchantstatsmerchantrepository

import (
	"context"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	merchant_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/merchant_errors/repository"
	"gorm.io/gorm"
)

type merchantStatsAmountByMerchantRepository struct {
	db *gorm.DB
}

func NewMerchantStatsAmountByMerchantRepository(db *gorm.DB) MerchantStatsAmountByMerchantRepository {
	return &merchantStatsAmountByMerchantRepository{db: db}
}

func (r *merchantStatsAmountByMerchantRepository) GetMonthlyAmountByMerchants(ctx context.Context, req *requests.MonthYearAmountMerchant) ([]*models.MerchantMonthlyAmountRow, error) {
	yearStart := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantMonthlyAmountRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(transaction_time, 'Mon') AS month, COALESCE(SUM(amount), 0)::int AS total_amount
		FROM transactions WHERE merchant_id = ? AND transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY TO_CHAR(transaction_time, 'Mon'), EXTRACT(MONTH FROM transaction_time)
		ORDER BY EXTRACT(MONTH FROM transaction_time)
	`, req.MerchantID, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetMonthlyAmountByMerchantsFailed.WithInternal(err)
	}
	return res, nil
}

func (r *merchantStatsAmountByMerchantRepository) GetYearlyAmountByMerchants(ctx context.Context, req *requests.MonthYearAmountMerchant) ([]*models.MerchantYearlyAmountRow, error) {
	yearStart := time.Date(req.Year-4, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantYearlyAmountRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(YEAR FROM transaction_time)::text AS year, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM transactions WHERE merchant_id = ? AND transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY EXTRACT(YEAR FROM transaction_time) ORDER BY EXTRACT(YEAR FROM transaction_time)
	`, req.MerchantID, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetYearlyAmountByMerchantsFailed.WithInternal(err)
	}
	return res, nil
}

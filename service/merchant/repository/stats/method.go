package merchantstatsrepository

import (
	"context"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	merchant_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/merchant_errors/repository"
	"gorm.io/gorm"
)

type merchantStatsMethodRepository struct {
	db *gorm.DB
}

func NewMerchantStatsMethodRepository(db *gorm.DB) MerchantStatsMethodRepository {
	return &merchantStatsMethodRepository{db: db}
}

func (r *merchantStatsMethodRepository) GetMonthlyPaymentMethodsMerchant(ctx context.Context, year int) ([]*models.MerchantMonthlyPaymentMethodRow, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantMonthlyPaymentMethodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(created_at, 'Mon') AS month, payment_method, COALESCE(SUM(amount), 0)::int AS total_amount
		FROM transactions WHERE created_at >= ? AND created_at < ? AND deleted_at IS NULL
		GROUP BY TO_CHAR(created_at, 'Mon'), EXTRACT(MONTH FROM created_at), payment_method
		ORDER BY EXTRACT(MONTH FROM created_at), payment_method
	`, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetMonthlyPaymentMethodsMerchantFailed.WithInternal(err)
	}
	return res, nil
}

func (r *merchantStatsMethodRepository) GetYearlyPaymentMethodMerchant(ctx context.Context, year int) ([]*models.MerchantYearlyPaymentMethodRow, error) {
	var res []*models.MerchantYearlyPaymentMethodRow
	yearStart := time.Date(year-4, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(YEAR FROM created_at)::text AS year, payment_method, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM transactions WHERE created_at >= ? AND created_at < ? AND deleted_at IS NULL
		GROUP BY EXTRACT(YEAR FROM created_at), payment_method ORDER BY EXTRACT(YEAR FROM created_at), payment_method
	`, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetYearlyPaymentMethodMerchantFailed.WithInternal(err)
	}
	return res, nil
}

package merchantstatsmerchantrepository

import (
	"context"
	"time"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	merchant_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/merchant_errors/repository"
	"gorm.io/gorm"
)

type merchantStatsMethodByMerchantRepository struct {
	db *gorm.DB
}

func NewMerchantStatsMethodByMerchantRepository(db *gorm.DB) MerchantStatsMethodByMerchantRepository {
	return &merchantStatsMethodByMerchantRepository{db: db}
}

func (r *merchantStatsMethodByMerchantRepository) GetMonthlyPaymentMethodByMerchants(ctx context.Context, req *requests.MonthYearPaymentMethodMerchant) ([]*models.MerchantMonthlyPaymentMethodRow, error) {
	yearStart := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantMonthlyPaymentMethodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(transaction_time, 'Mon') AS month, payment_method, COALESCE(SUM(amount), 0)::int AS total_amount
		FROM transactions WHERE merchant_id = ? AND transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY TO_CHAR(transaction_time, 'Mon'), EXTRACT(MONTH FROM transaction_time), payment_method
		ORDER BY EXTRACT(MONTH FROM transaction_time), payment_method
	`, req.MerchantID, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetMonthlyPaymentMethodByMerchantsFailed.WithInternal(err)
	}
	return res, nil
}

func (r *merchantStatsMethodByMerchantRepository) GetYearlyPaymentMethodByMerchants(ctx context.Context, req *requests.MonthYearPaymentMethodMerchant) ([]*models.MerchantYearlyPaymentMethodRow, error) {
	yearStart := time.Date(req.Year-4, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var res []*models.MerchantYearlyPaymentMethodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXTRACT(YEAR FROM transaction_time)::text AS year, payment_method, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM transactions WHERE merchant_id = ? AND transaction_time >= ? AND transaction_time < ? AND deleted_at IS NULL
		GROUP BY EXTRACT(YEAR FROM transaction_time), payment_method ORDER BY EXTRACT(YEAR FROM transaction_time), payment_method
	`, req.MerchantID, yearStart, yearEnd).Scan(&res).Error
	if err != nil {
		return nil, merchant_errors.ErrGetYearlyPaymentMethodByMerchantsFailed.WithInternal(err)
	}
	return res, nil
}

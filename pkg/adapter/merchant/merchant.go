// Package merchant adapts the Merchant service gRPC API into the shared domain model.
package merchant

import (
	"context"

	pbmerchant "github.com/MamangRust/monolith-payment-gateway-pb/merchant"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
)

// QueryRepository is the contract consumers depend on for merchant reads.
type QueryRepository interface {
	FindByApiKey(ctx context.Context, api_key string) (*models.MerchantAllFieldsRow, error)
}

// Repository implements QueryRepository on top of the generated merchant query
// client.
type Repository struct {
	query pbmerchant.MerchantQueryServiceClient
	guard *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (a *Repository) SetGuard(g *resilience.DependencyGuard) { a.guard = g }

// NewAdapter wraps the generated merchant query client. Resilience (timeout/
// circuit-breaker/bulkhead) is applied per call through the optional dependency
// guard; a nil guard is a passthrough.
func NewAdapter(query pbmerchant.MerchantQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	a := &Repository{query: query}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// New wraps the generated merchant query client. It is the canonical
// constructor; modules build it inside their repositories with the guard
// options they were given.
func New(query pbmerchant.MerchantQueryServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(query, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface
// for consumers that only read.
func NewQueryAdapter(query pbmerchant.MerchantQueryServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, opts...)
}

func (a *Repository) FindByApiKey(ctx context.Context, api_key string) (*models.MerchantAllFieldsRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbmerchant.ApiResponseMerchant, error) {
		return a.query.FindByApiKey(ctx, &pbmerchant.FindByApiKeyRequest{
			ApiKey: api_key,
		})
	})
	if err != nil {
		return nil, err
	}

	return &models.MerchantAllFieldsRow{
		MerchantID: res.Data.Id,
		Name:       res.Data.Name,
		ApiKey:     res.Data.ApiKey,
		UserID:     res.Data.UserId,
		Status:     res.Data.Status,
	}, nil
}

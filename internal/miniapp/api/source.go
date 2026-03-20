package api

import (
	"context"

	"mrtang-pim/internal/miniapp/model"
)

type Source interface {
	FetchDataset(ctx context.Context) (*model.Dataset, error)
}

type TargetSyncSource interface {
	FetchTargetSyncDataset(ctx context.Context, entityType string, scopeKey string) (*model.Dataset, error)
}

type TargetSyncProductSource interface {
	FetchTargetSyncProductsFromSections(ctx context.Context, sections []model.CategorySection, scopeKey string) (*model.Dataset, error)
}

type ProductResolverSource interface {
	ResolveProduct(ctx context.Context, spuID string, skuID string) (*model.ProductPage, error)
}

type StatusSource interface {
	RawAuthStatus() model.RawAuthStatus
}

type ActionSource interface {
	ExecuteCartOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error)
	ExecuteOrderOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error)
	ExecuteFreightScenario(ctx context.Context, scenario string, requestBody any) (*model.ScenarioAction, error)
}

package api

import (
	"context"

	"mrtang-pim/internal/miniapp/model"
)

type Source interface {
	FetchDataset(ctx context.Context) (*model.Dataset, error)
}

type ActionSource interface {
	ExecuteCartOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error)
	ExecuteOrderOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error)
	ExecuteFreightScenario(ctx context.Context, scenario string, requestBody any) (*model.ScenarioAction, error)
}

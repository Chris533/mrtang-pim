package api

import (
	"context"

	"mrtang-pim/internal/miniapp/model"
)

type Source interface {
	FetchDataset(ctx context.Context) (*model.Dataset, error)
}

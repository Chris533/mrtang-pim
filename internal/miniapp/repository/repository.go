package repository

import (
	"context"

	"mrtang-pim/internal/miniapp/model"
)

type HomepageSnapshotRepository interface {
	SaveHomepageSnapshot(ctx context.Context, dataset *model.Dataset) error
}

type NoopHomepageSnapshotRepository struct{}

func NewNoopHomepageSnapshotRepository() *NoopHomepageSnapshotRepository {
	return &NoopHomepageSnapshotRepository{}
}

func (r *NoopHomepageSnapshotRepository) SaveHomepageSnapshot(_ context.Context, _ *model.Dataset) error {
	return nil
}

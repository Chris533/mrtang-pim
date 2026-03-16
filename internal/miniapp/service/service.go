package service

import (
	"context"

	"mrtang-pim/internal/miniapp/api"
	"mrtang-pim/internal/miniapp/importer"
	"mrtang-pim/internal/miniapp/model"
	"mrtang-pim/internal/miniapp/repository"
)

type Service struct {
	source     api.Source
	importer   *importer.HomepageImporter
	repository repository.HomepageSnapshotRepository
}

func New(source api.Source, repo repository.HomepageSnapshotRepository) *Service {
	if repo == nil {
		repo = repository.NewNoopHomepageSnapshotRepository()
	}

	return &Service{
		source:     source,
		importer:   importer.NewHomepageImporter(),
		repository: repo,
	}
}

func (s *Service) Dataset(ctx context.Context) (*model.Dataset, error) {
	return s.source.FetchDataset(ctx)
}

func (s *Service) Contracts(ctx context.Context, localPathPrefix string) ([]model.Contract, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.Contracts(dataset, localPathPrefix), nil
}

func (s *Service) Homepage(ctx context.Context) (model.HomepageAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.HomepageAggregate{}, err
	}

	return s.importer.Homepage(dataset), nil
}

func (s *Service) CategoryPage(ctx context.Context) (model.CategoryPageAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CategoryPageAggregate{}, err
	}

	return s.importer.CategoryPage(dataset), nil
}

func (s *Service) Section(ctx context.Context, id string) (*model.HomepageSection, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.Section(dataset, id), nil
}

func (s *Service) CategorySection(ctx context.Context, id string) (*model.CategorySection, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.CategorySection(dataset, id), nil
}

func (s *Service) SaveSnapshot(ctx context.Context) error {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return err
	}

	return s.repository.SaveHomepageSnapshot(ctx, dataset)
}

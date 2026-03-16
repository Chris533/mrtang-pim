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

func (s *Service) ProductPage(ctx context.Context) (model.ProductPageAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.ProductPageAggregate{}, err
	}

	return s.importer.ProductPage(dataset), nil
}

func (s *Service) CartOrder(ctx context.Context) (model.CartOrderAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CartOrderAggregate{}, err
	}

	return s.importer.CartOrder(dataset), nil
}

func (s *Service) Cart(ctx context.Context) (model.CartAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CartAggregate{}, err
	}

	return s.importer.Cart(dataset), nil
}

func (s *Service) Order(ctx context.Context) (model.OrderAggregate, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.OrderAggregate{}, err
	}

	return s.importer.Order(dataset), nil
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

func (s *Service) Product(ctx context.Context, id string) (*model.ProductPage, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.Product(dataset, id), nil
}

func (s *Service) ProductCoverage(ctx context.Context) ([]model.ProductCoverage, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.ProductCoverage(dataset), nil
}

func (s *Service) ProductCoverageSummary(ctx context.Context) (model.ProductCoverageSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.ProductCoverageSummary{}, err
	}

	return s.importer.ProductCoverageSummary(dataset), nil
}

func (s *Service) CartOperation(ctx context.Context, id string) (*model.OperationSnapshot, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.CartOperation(dataset, id), nil
}

func (s *Service) OrderOperation(ctx context.Context, id string) (*model.OperationSnapshot, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.OrderOperation(dataset, id), nil
}

func (s *Service) FreightCost(ctx context.Context, scenario string) (*model.ScenarioAction, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return nil, err
	}

	return s.importer.FreightCost(dataset, scenario), nil
}

func (s *Service) CartDetailSummary(ctx context.Context) (model.CartDetailSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CartDetailSummary{}, err
	}

	return s.importer.CartDetailSummary(dataset), nil
}

func (s *Service) CartListSummary(ctx context.Context) (model.CartListSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CartListSummary{}, err
	}

	return s.importer.CartListSummary(dataset), nil
}

func (s *Service) OrderSubmitSummary(ctx context.Context) (model.OrderSubmitSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.OrderSubmitSummary{}, err
	}

	return s.importer.OrderSubmitSummary(dataset), nil
}

func (s *Service) FreightSummary(ctx context.Context) (model.FreightSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.FreightSummary{}, err
	}

	return s.importer.FreightSummary(dataset), nil
}

func (s *Service) DefaultDeliverySummary(ctx context.Context) (model.DefaultDeliverySummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.DefaultDeliverySummary{}, err
	}

	return s.importer.DefaultDeliverySummary(dataset), nil
}

func (s *Service) DeliveriesSummary(ctx context.Context) (model.DeliveriesSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.DeliveriesSummary{}, err
	}

	return s.importer.DeliveriesSummary(dataset), nil
}

func (s *Service) CheckoutSummary(ctx context.Context) (model.CheckoutSummary, error) {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return model.CheckoutSummary{}, err
	}

	return s.importer.CheckoutSummary(dataset), nil
}

func (s *Service) SaveSnapshot(ctx context.Context) error {
	dataset, err := s.Dataset(ctx)
	if err != nil {
		return err
	}

	return s.repository.SaveHomepageSnapshot(ctx, dataset)
}

package importer

import (
	"strings"

	"mrtang-pim/internal/miniapp/model"
)

type HomepageImporter struct{}

func NewHomepageImporter() *HomepageImporter {
	return &HomepageImporter{}
}

func (i *HomepageImporter) Homepage(dataset *model.Dataset) model.HomepageAggregate {
	if dataset == nil {
		return model.HomepageAggregate{}
	}

	return dataset.Homepage
}

func (i *HomepageImporter) CategoryPage(dataset *model.Dataset) model.CategoryPageAggregate {
	if dataset == nil {
		return model.CategoryPageAggregate{}
	}

	return dataset.CategoryPage
}

func (i *HomepageImporter) Section(dataset *model.Dataset, id string) *model.HomepageSection {
	if dataset == nil {
		return nil
	}

	for _, section := range dataset.Homepage.Sections {
		if strings.EqualFold(section.ID, id) {
			copySection := section
			return &copySection
		}
	}

	return nil
}

func (i *HomepageImporter) CategorySection(dataset *model.Dataset, id string) *model.CategorySection {
	if dataset == nil {
		return nil
	}

	for _, section := range dataset.CategoryPage.Sections {
		if strings.EqualFold(section.ID, id) {
			copySection := section
			return &copySection
		}
	}

	return nil
}

func (i *HomepageImporter) Contracts(dataset *model.Dataset, localPathPrefix string) []model.Contract {
	if dataset == nil {
		return nil
	}

	prefix := strings.TrimSpace(localPathPrefix)
	if prefix == "" {
		return append([]model.Contract(nil), dataset.Contracts...)
	}

	filtered := make([]model.Contract, 0, len(dataset.Contracts))
	for _, contract := range dataset.Contracts {
		if strings.HasPrefix(contract.LocalPath, prefix) {
			filtered = append(filtered, contract)
		}
	}

	return filtered
}

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

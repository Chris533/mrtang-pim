package importer

import (
	"sort"
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

	homepage := dataset.Homepage
	homepage.Sections = copyHomepageSections(homepage.Sections)
	return homepage
}

func (i *HomepageImporter) CategoryPage(dataset *model.Dataset) model.CategoryPageAggregate {
	if dataset == nil {
		return model.CategoryPageAggregate{}
	}

	categoryPage := dataset.CategoryPage
	categoryPage.Sections = copyCategorySections(categoryPage.Sections)
	return categoryPage
}

func (i *HomepageImporter) ProductPage(dataset *model.Dataset) model.ProductPageAggregate {
	if dataset == nil {
		return model.ProductPageAggregate{}
	}

	productPage := dataset.ProductPage
	productPage.Products = copyProductPages(productPage.Products)
	return productPage
}

func (i *HomepageImporter) Section(dataset *model.Dataset, id string) *model.HomepageSection {
	if dataset == nil {
		return nil
	}

	for _, section := range dataset.Homepage.Sections {
		if strings.EqualFold(section.ID, id) {
			copySection := section
			copySection.Products = copyHomepageProducts(copySection.Products)
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
			copySection.Products = copyHomepageProducts(copySection.Products)
			return &copySection
		}
	}

	return nil
}

func (i *HomepageImporter) Product(dataset *model.Dataset, id string) *model.ProductPage {
	if dataset == nil {
		return nil
	}

	for _, product := range dataset.ProductPage.Products {
		if strings.EqualFold(product.ID, id) {
			copyProduct := product
			copyProduct.ID = normalizedProductID(copyProduct.ID, copyProduct.SpuID, copyProduct.SkuID)
			copyProduct.Summary = normalizeHomepageProduct(copyProduct.Summary)
			return &copyProduct
		}
	}

	return nil
}

func (i *HomepageImporter) ProductCoverage(dataset *model.Dataset) []model.ProductCoverage {
	if dataset == nil {
		return nil
	}

	products := copyProductPages(dataset.ProductPage.Products)
	coverage := make([]model.ProductCoverage, 0, len(products))
	for _, product := range products {
		entry := model.ProductCoverage{
			ProductID:      product.ID,
			SpuID:          product.SpuID,
			SkuID:          product.SkuID,
			Name:           product.Summary.Name,
			SourceType:     product.SourceType,
			SourceSections: append([]string(nil), product.SourceSections...),
			UnitCount:      len(product.Summary.UnitOptions),
			HasMultiUnit:   len(product.Summary.UnitOptions) > 1,
			Priority:       productPriority(product),
		}
		coverage = append(coverage, entry)
	}

	sort.SliceStable(coverage, func(a int, b int) bool {
		left := priorityRank(coverage[a].Priority)
		right := priorityRank(coverage[b].Priority)
		if left != right {
			return left < right
		}
		return coverage[a].Name < coverage[b].Name
	})

	return coverage
}

func (i *HomepageImporter) ProductCoverageSummary(dataset *model.Dataset) model.ProductCoverageSummary {
	coverage := i.ProductCoverage(dataset)
	summary := model.ProductCoverageSummary{
		TotalProducts: len(coverage),
	}
	if len(coverage) == 0 {
		return summary
	}

	grouped := make(map[string][]model.ProductCoverage)
	priorityOrder := []string{
		"homepage_dual_unit",
		"category_dual_unit",
		"visible_single_unit",
		"done_rr_detail",
	}

	for _, item := range coverage {
		if item.HasMultiUnit {
			summary.MultiUnitTotal++
		}
		grouped[item.Priority] = append(grouped[item.Priority], item)
	}

	buckets := make([]model.ProductCoverageBucket, 0, len(priorityOrder))
	for _, priority := range priorityOrder {
		items := grouped[priority]
		if len(items) == 0 {
			continue
		}
		buckets = append(buckets, model.ProductCoverageBucket{
			Priority: priority,
			Count:    len(items),
			Items:    items,
		})
	}
	summary.ByPriority = buckets
	summary.FirstBatch = append([]model.ProductCoverage(nil), grouped["homepage_dual_unit"]...)

	return summary
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

func copyHomepageSections(sections []model.HomepageSection) []model.HomepageSection {
	if len(sections) == 0 {
		return sections
	}

	copied := make([]model.HomepageSection, len(sections))
	for idx, section := range sections {
		copied[idx] = section
		copied[idx].Products = copyHomepageProducts(section.Products)
	}

	return copied
}

func copyCategorySections(sections []model.CategorySection) []model.CategorySection {
	if len(sections) == 0 {
		return sections
	}

	copied := make([]model.CategorySection, len(sections))
	for idx, section := range sections {
		copied[idx] = section
		copied[idx].Products = copyHomepageProducts(section.Products)
	}

	return copied
}

func copyHomepageProducts(products []model.HomepageProduct) []model.HomepageProduct {
	if len(products) == 0 {
		return products
	}

	copied := make([]model.HomepageProduct, len(products))
	for idx, product := range products {
		copied[idx] = normalizeHomepageProduct(product)
	}

	return copied
}

func copyProductPages(products []model.ProductPage) []model.ProductPage {
	if len(products) == 0 {
		return products
	}

	copied := make([]model.ProductPage, len(products))
	for idx, product := range products {
		copied[idx] = product
		copied[idx].ID = normalizedProductID(product.ID, product.SpuID, product.SkuID)
		copied[idx].Summary = normalizeHomepageProduct(product.Summary)
	}

	return copied
}

func normalizeHomepageProduct(product model.HomepageProduct) model.HomepageProduct {
	product.ProductID = normalizedProductID(product.ProductID, product.SpuID, product.SkuID)
	return product
}

func normalizedProductID(current string, spuID string, skuID string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	if strings.TrimSpace(spuID) == "" || strings.TrimSpace(skuID) == "" {
		return ""
	}
	return spuID + "_" + skuID
}

func productPriority(product model.ProductPage) string {
	if strings.EqualFold(product.SourceType, "rr_detail") {
		return "done_rr_detail"
	}

	hasMultiUnit := len(product.Summary.UnitOptions) > 1
	hasHomepage := containsAny(product.SourceSections, "new", "hot")
	hasCategory := containsAny(product.SourceSections, "chicken")

	switch {
	case hasHomepage && hasMultiUnit:
		return "homepage_dual_unit"
	case hasCategory && hasMultiUnit:
		return "category_dual_unit"
	default:
		return "visible_single_unit"
	}
}

func priorityRank(priority string) int {
	switch priority {
	case "homepage_dual_unit":
		return 0
	case "category_dual_unit":
		return 1
	case "visible_single_unit":
		return 2
	case "done_rr_detail":
		return 3
	default:
		return 4
	}
}

func containsAny(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if strings.EqualFold(value, target) {
				return true
			}
		}
	}
	return false
}

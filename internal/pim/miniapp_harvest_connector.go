package pim

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mrtang-pim/internal/config"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/supplier"
)

type miniappHarvestConnector struct {
	supplierCode   string
	sourceMode     string
	assetBaseURL   string
	loadDataset    func(context.Context) (*miniappmodel.Dataset, error)
	resolveProduct func(context.Context, string, string) (*miniappmodel.ProductPage, error)
}

func newMiniappHarvestConnector(cfg config.Config) *miniappHarvestConnector {
	miniapp := miniappservice.New(newMiniappActionSource(cfg), nil)
	return &miniappHarvestConnector{
		supplierCode: strings.TrimSpace(cfg.Supplier.Code),
		sourceMode:   strings.TrimSpace(cfg.MiniApp.SourceMode),
		assetBaseURL: strings.TrimSpace(cfg.MiniApp.RawAssetBaseURL),
		loadDataset: func(ctx context.Context) (*miniappmodel.Dataset, error) {
			return miniapp.TargetSyncDataset(ctx, TargetSyncEntityProducts, "")
		},
		resolveProduct: func(ctx context.Context, spuID string, skuID string) (*miniappmodel.ProductPage, error) {
			return miniapp.ResolveProduct(ctx, spuID, skuID)
		},
	}
}

func (c *miniappHarvestConnector) Capabilities() supplier.ConnectorCapabilities {
	return supplier.ConnectorCapabilities{
		FetchProducts:       true,
		SubmitPurchaseOrder: false,
		ExportPurchaseOrder: false,
	}
}

func (c *miniappHarvestConnector) Fetch(ctx context.Context) ([]supplier.Product, error) {
	if !strings.EqualFold(strings.TrimSpace(c.sourceMode), "raw") {
		return nil, fmt.Errorf("miniapp harvest requires MINIAPP_SOURCE_MODE=raw")
	}
	if c.loadDataset == nil {
		return nil, fmt.Errorf("miniapp harvest dataset loader is not configured")
	}

	dataset, err := c.loadDataset(ctx)
	if err != nil {
		return nil, fmt.Errorf("load miniapp product dataset: %w", err)
	}

	fetchedAt := time.Now()
	products := make([]supplier.Product, 0, len(dataset.ProductPage.Products))
	indexBySKU := make(map[string]int, len(dataset.ProductPage.Products))
	for _, product := range dataset.ProductPage.Products {
		current := product
		if c.resolveProduct != nil && miniappProductNeedsResolve(current) {
			if resolved, err := c.resolveProduct(ctx, current.SpuID, current.SkuID); err == nil && resolved != nil {
				current = mergeResolvedMiniappProduct(current, *resolved)
			}
		}
		item, ok := miniappProductToSupplierProduct(c.supplierCode, c.assetBaseURL, current, fetchedAt)
		if !ok {
			continue
		}
		if index, exists := indexBySKU[item.OriginalSKU]; exists {
			products[index] = item
			continue
		}
		indexBySKU[item.OriginalSKU] = len(products)
		products = append(products, item)
	}

	return products, nil
}

func (c *miniappHarvestConnector) SubmitPurchaseOrder(ctx context.Context, order supplier.PurchaseOrder) (supplier.PurchaseOrderResult, error) {
	select {
	case <-ctx.Done():
		return supplier.PurchaseOrderResult{}, ctx.Err()
	default:
	}

	return supplier.PurchaseOrderResult{
		SupplierCode: defaultString(strings.TrimSpace(order.SupplierCode), c.supplierCode),
		ExternalRef:  strings.TrimSpace(order.ExternalRef),
		Mode:         "miniapp_cart_order",
		Accepted:     false,
		Message:      "miniapp harvest connector does not submit purchase orders directly",
	}, nil
}

func miniappProductToSupplierProduct(supplierCode string, assetBaseURL string, product miniappmodel.ProductPage, fetchedAt time.Time) (supplier.Product, bool) {
	originalSKU := strings.TrimSpace(product.SkuID)
	if originalSKU == "" {
		return supplier.Product{}, false
	}

	sourceProductID := firstNonEmptyString(strings.TrimSpace(product.ID), strings.TrimSpace(product.SpuID)+"_"+originalSKU)
	rawCategory := firstNonEmptyString(
		strings.TrimSpace(product.CategoryPath),
		firstNonEmptyString(product.ObservedCategoryPaths...),
	)
	rawImageURL := sanitizeURLWithBase(firstNonEmptyString(
		strings.TrimSpace(product.Summary.Cover),
		firstMiniappImageURL(product.Detail.Carousel),
		firstMiniappImageURL(product.Detail.DetailAssets),
	), assetBaseURL)
	defaultUnit := firstNonEmptyString(
		strings.TrimSpace(product.Pricing.DefaultUnit),
		strings.TrimSpace(product.Summary.DefaultUnit),
		strings.TrimSpace(product.Detail.DefaultUnit),
		"件",
	)
	defaultStockQty, defaultStockText := resolvedMiniappDefaultStock(product)
	galleryURLs := miniappGalleryURLs(assetBaseURL, product)

	return supplier.Product{
		SupplierCode:      strings.TrimSpace(supplierCode),
		OriginalSKU:       originalSKU,
		RawTitle:          firstNonEmptyString(strings.TrimSpace(product.Summary.Name), strings.TrimSpace(product.Detail.Name), strings.TrimSpace(product.Detail.SkuName)),
		RawDescription:    strings.TrimSpace(strings.Join(product.Detail.DetailTexts, "\n")),
		RawCategory:       rawCategory,
		RawImageURL:       rawImageURL,
		CostPrice:         0,
		BPrice:            product.Pricing.DefaultPrice,
		CPrice:            product.Pricing.DefaultPrice,
		CurrencyCode:      "CNY",
		SupplierUpdatedAt: fetchedAt,
		Payload: map[string]any{
			"source_product_id":       sourceProductID,
			"source_type":             strings.TrimSpace(product.SourceType),
			"sales_unit":              defaultUnit,
			"conversion_rate":         miniappProductConversionRate(product, defaultUnit),
			"target_audience":         "ALL",
			"release_category_key":    strings.TrimSpace(product.CategoryKey),
			"release_category_path":   rawCategory,
			"category_key":            strings.TrimSpace(product.CategoryKey),
			"category_keys":           append([]string(nil), product.CategoryKeys...),
			"observed_category_keys":  append([]string(nil), product.ObservedCategoryKeys...),
			"observed_category_paths": append([]string(nil), product.ObservedCategoryPaths...),
			"unit_options":            append([]miniappmodel.UnitOption(nil), product.Pricing.UnitOptions...),
			"order_units":             append([]miniappmodel.ProductOrderUnit(nil), product.Context.UnitOptions...),
			"default_stock_qty":       defaultStockQty,
			"default_stock_text":      defaultStockText,
			"gallery_urls":            galleryURLs,
		},
	}, true
}

func miniappProductConversionRate(product miniappmodel.ProductPage, defaultUnit string) float64 {
	defaultUnit = strings.TrimSpace(defaultUnit)
	for _, option := range product.Pricing.UnitOptions {
		if option.IsDefault && option.Rate > 0 {
			return option.Rate
		}
		if defaultUnit != "" && strings.EqualFold(strings.TrimSpace(option.UnitName), defaultUnit) && option.Rate > 0 {
			return option.Rate
		}
	}
	return 1
}

func resolvedMiniappDefaultStock(product miniappmodel.ProductPage) (float64, string) {
	if len(product.Pricing.UnitOptions) == 0 {
		return product.Pricing.DefaultStockQty, strings.TrimSpace(product.Pricing.DefaultStockText)
	}

	defaultUnit := firstNonEmptyString(
		strings.TrimSpace(product.Pricing.DefaultUnit),
		strings.TrimSpace(product.Summary.DefaultUnit),
		strings.TrimSpace(product.Detail.DefaultUnit),
	)
	for _, option := range product.Pricing.UnitOptions {
		if option.IsDefault {
			return option.StockQty, strings.TrimSpace(option.StockText)
		}
		if defaultUnit != "" && strings.EqualFold(strings.TrimSpace(option.UnitName), defaultUnit) {
			return option.StockQty, strings.TrimSpace(option.StockText)
		}
	}

	return product.Pricing.DefaultStockQty, strings.TrimSpace(product.Pricing.DefaultStockText)
}

func firstMiniappImageURL(items []miniappmodel.ProductMedia) string {
	for _, item := range items {
		if value := strings.TrimSpace(item.ImageURL); value != "" {
			return value
		}
	}
	return ""
}

func miniappProductNeedsResolve(product miniappmodel.ProductPage) bool {
	sourceType := strings.ToLower(strings.TrimSpace(product.SourceType))
	if sourceType == "raw_detail" || sourceType == "rr_detail" {
		return false
	}
	if len(product.Detail.Carousel) > 1 || len(product.Detail.DetailAssets) > 0 {
		return false
	}
	return strings.TrimSpace(product.SpuID) != "" && strings.TrimSpace(product.SkuID) != ""
}

func mergeResolvedMiniappProduct(base miniappmodel.ProductPage, resolved miniappmodel.ProductPage) miniappmodel.ProductPage {
	if strings.TrimSpace(resolved.CategoryKey) == "" {
		resolved.CategoryKey = base.CategoryKey
	}
	if strings.TrimSpace(resolved.CategoryPath) == "" {
		resolved.CategoryPath = base.CategoryPath
	}
	if len(resolved.CategoryKeys) == 0 {
		resolved.CategoryKeys = append([]string(nil), base.CategoryKeys...)
	}
	if len(resolved.SourceSections) == 0 {
		resolved.SourceSections = append([]string(nil), base.SourceSections...)
	}
	if len(resolved.ObservedCategoryKeys) == 0 {
		resolved.ObservedCategoryKeys = append([]string(nil), base.ObservedCategoryKeys...)
	}
	if len(resolved.ObservedCategoryPaths) == 0 {
		resolved.ObservedCategoryPaths = append([]string(nil), base.ObservedCategoryPaths...)
	}
	if strings.TrimSpace(resolved.Summary.Cover) == "" {
		resolved.Summary.Cover = base.Summary.Cover
	}
	if strings.TrimSpace(resolved.Summary.Name) == "" {
		resolved.Summary.Name = base.Summary.Name
	}
	if strings.TrimSpace(resolved.Summary.SkuName) == "" {
		resolved.Summary.SkuName = base.Summary.SkuName
	}
	return resolved
}

func miniappGalleryURLs(assetBaseURL string, product miniappmodel.ProductPage) []string {
	items := []string{
		sanitizeURLWithBase(strings.TrimSpace(product.Summary.Cover), assetBaseURL),
	}
	for _, media := range product.Detail.Carousel {
		items = append(items, sanitizeURLWithBase(strings.TrimSpace(media.ImageURL), assetBaseURL))
	}
	for _, media := range product.Detail.DetailAssets {
		items = append(items, sanitizeURLWithBase(strings.TrimSpace(media.ImageURL), assetBaseURL))
	}
	return uniqueTrimmed(items)
}

package pim

import (
	"context"
	"testing"
	"time"

	miniappmodel "mrtang-pim/internal/miniapp/model"
	"mrtang-pim/internal/supplier"
)

func TestMiniappProductToSupplierProduct(t *testing.T) {
	fetchedAt := time.Date(2026, 3, 24, 9, 0, 0, 0, time.UTC)
	product, ok := miniappProductToSupplierProduct("SUP_A", "", miniappmodel.ProductPage{
		ID:                   "593415780247420928_593415780754931712",
		SpuID:                "593415780247420928",
		SkuID:                "593415780754931712",
		SourceType:           "raw_detail",
		CategoryKey:          "duck-blood",
		CategoryPath:         "火锅食材/鸭血",
		CategoryKeys:         []string{"hotpot", "duck-blood"},
		ObservedCategoryKeys: []string{"hotpot", "duck-blood"},
		ObservedCategoryPaths: []string{
			"火锅食材/鸭血",
			"冷冻食材/鸭血",
		},
		Summary: miniappmodel.HomepageProduct{
			Name:        "鸭血",
			Cover:       "https://img.example.com/duck-blood-cover.jpg",
			DefaultUnit: "盒",
		},
		Detail: miniappmodel.ProductDetail{
			Name:        "鸭血",
			DetailTexts: []string{"250g/盒", "火锅专用"},
		},
		Pricing: miniappmodel.ProductPricing{
			DefaultUnit:      "盒",
			DefaultPrice:     12.8,
			DefaultStockQty:  8,
			DefaultStockText: "有货",
			UnitOptions: []miniappmodel.UnitOption{
				{UnitName: "盒", Price: 12.8, Rate: 1, IsDefault: true, StockQty: 8, StockText: "有货"},
				{UnitName: "箱", Price: 120, Rate: 10, StockQty: 2, StockText: "2箱"},
			},
		},
		Context: miniappmodel.ProductContext{
			UnitOptions: []miniappmodel.ProductOrderUnit{
				{UnitID: "u-box", UnitName: "盒", Rate: 1, IsDefault: true, AllowOrder: true},
				{UnitID: "u-case", UnitName: "箱", Rate: 10, AllowOrder: true},
			},
		},
	}, fetchedAt)
	if !ok {
		t.Fatal("expected product to map successfully")
	}

	if product.OriginalSKU != "593415780754931712" {
		t.Fatalf("unexpected sku: %q", product.OriginalSKU)
	}
	if product.RawCategory != "火锅食材/鸭血" {
		t.Fatalf("unexpected category: %q", product.RawCategory)
	}
	if product.RawImageURL != "https://img.example.com/duck-blood-cover.jpg" {
		t.Fatalf("unexpected image: %q", product.RawImageURL)
	}
	if product.BPrice != 12.8 || product.CPrice != 12.8 {
		t.Fatalf("unexpected prices: b=%.2f c=%.2f", product.BPrice, product.CPrice)
	}
	if !product.SupplierUpdatedAt.Equal(fetchedAt) {
		t.Fatalf("unexpected fetchedAt: %s", product.SupplierUpdatedAt)
	}

	if got := product.Payload["source_product_id"]; got != "593415780247420928_593415780754931712" {
		t.Fatalf("unexpected source_product_id: %#v", got)
	}
	if got := product.Payload["sales_unit"]; got != "盒" {
		t.Fatalf("unexpected sales_unit: %#v", got)
	}
	if got := product.Payload["conversion_rate"]; got != 1.0 {
		t.Fatalf("unexpected conversion_rate: %#v", got)
	}
}

func TestMiniappHarvestConnectorFetchDeduplicatesBySKU(t *testing.T) {
	connector := &miniappHarvestConnector{
		supplierCode: "SUP_A",
		sourceMode:   "raw",
		loadDataset: func(context.Context) (*miniappmodel.Dataset, error) {
			return &miniappmodel.Dataset{
				ProductPage: miniappmodel.ProductPageAggregate{
					Products: []miniappmodel.ProductPage{
						{
							ID:           "spu-1_sku-1",
							SpuID:        "spu-1",
							SkuID:        "sku-1",
							CategoryPath: "分类A",
							Summary: miniappmodel.HomepageProduct{
								Name:        "商品A",
								DefaultUnit: "件",
							},
							Pricing: miniappmodel.ProductPricing{
								DefaultUnit:  "件",
								DefaultPrice: 10,
								UnitOptions: []miniappmodel.UnitOption{
									{UnitName: "件", Price: 10, Rate: 1, IsDefault: true},
								},
							},
						},
						{
							ID:           "spu-2_sku-1",
							SpuID:        "spu-2",
							SkuID:        "sku-1",
							CategoryPath: "分类B",
							Summary: miniappmodel.HomepageProduct{
								Name:        "商品A-更新",
								DefaultUnit: "件",
							},
							Pricing: miniappmodel.ProductPricing{
								DefaultUnit:  "件",
								DefaultPrice: 11,
								UnitOptions: []miniappmodel.UnitOption{
									{UnitName: "件", Price: 11, Rate: 1, IsDefault: true},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	items, err := connector.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deduplicated item, got %d", len(items))
	}
	if items[0].RawTitle != "商品A-更新" {
		t.Fatalf("expected later duplicate to win, got %q", items[0].RawTitle)
	}
	if items[0].RawCategory != "分类B" {
		t.Fatalf("expected later duplicate category, got %q", items[0].RawCategory)
	}
}

func TestMiniappProductToSupplierProductSanitizesRelativeCoverURL(t *testing.T) {
	fetchedAt := time.Date(2026, 3, 24, 9, 30, 0, 0, time.UTC)
	product, ok := miniappProductToSupplierProduct("SUP_A", "https://static-hdsaas.handday.com", miniappmodel.ProductPage{
		ID:    "spu-1_sku-1",
		SpuID: "spu-1",
		SkuID: "sku-1",
		Summary: miniappmodel.HomepageProduct{
			Name:  "测试商品",
			Cover: "/150032890/example.jpg",
		},
		Pricing: miniappmodel.ProductPricing{
			DefaultUnit:  "件",
			DefaultPrice: 10,
			UnitOptions: []miniappmodel.UnitOption{
				{UnitName: "件", Price: 10, Rate: 1, IsDefault: true},
			},
		},
	}, fetchedAt)
	if !ok {
		t.Fatal("expected product to map successfully")
	}
	if product.RawImageURL != "https://static-hdsaas.handday.com/150032890/example.jpg" {
		t.Fatalf("unexpected sanitized image url: %q", product.RawImageURL)
	}
}

func TestConnectorCapabilitiesMergesMiniappFetchAndSubmit(t *testing.T) {
	service := &Service{
		connector: &stubConnector{
			capabilities: supplier.ConnectorCapabilities{
				FetchProducts: true,
			},
		},
		miniappCartOrder: &miniappCartOrderSubmitter{},
	}

	capabilities := service.ConnectorCapabilities()
	if !capabilities.FetchProducts {
		t.Fatal("expected fetchProducts to stay enabled")
	}
	if !capabilities.SubmitPurchaseOrder {
		t.Fatal("expected submitPurchaseOrder to be enabled")
	}
	if capabilities.ExportPurchaseOrder {
		t.Fatal("did not expect exportPurchaseOrder to be enabled")
	}
}

type stubConnector struct {
	capabilities supplier.ConnectorCapabilities
}

func (s *stubConnector) Fetch(context.Context) ([]supplier.Product, error) {
	return nil, nil
}

func (s *stubConnector) Capabilities() supplier.ConnectorCapabilities {
	return s.capabilities
}

func (s *stubConnector) SubmitPurchaseOrder(context.Context, supplier.PurchaseOrder) (supplier.PurchaseOrderResult, error) {
	return supplier.PurchaseOrderResult{}, nil
}

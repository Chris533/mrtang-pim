package pim

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/supplier"
	"mrtang-pim/internal/vendure"
)

func TestBuildProcurementSummary(t *testing.T) {
	summary := buildProcurementSummary(
		"file",
		supplier.ConnectorCapabilities{
			FetchProducts:       true,
			SubmitPurchaseOrder: false,
			ExportPurchaseOrder: true,
		},
		"PO-001",
		"攀枝花一号冻库",
		"test order",
		[]procurementCatalogItem{
			{
				SupplierCode:       "SUP_A",
				OriginalSKU:        "SKU-1",
				Title:              "谷饲肥牛卷 500g",
				NormalizedCategory: "冷冻肉类/肥牛",
				Quantity:           5,
				SalesUnit:          "盒",
				CostPrice:          36.5,
				BusinessPrice:      45,
				ConsumerPrice:      40,
				NeedColdChain:      true,
			},
			{
				SupplierCode:       "SUP_B",
				OriginalSKU:        "SKU-2",
				Title:              "手打牛肉丸",
				NormalizedCategory: "丸滑",
				Quantity:           3,
				SalesUnit:          "袋",
				CostPrice:          18,
				BusinessPrice:      24,
				ConsumerPrice:      32,
			},
		},
	)

	if summary.SupplierCount != 2 {
		t.Fatalf("expected 2 suppliers, got %d", summary.SupplierCount)
	}

	if summary.ItemCount != 2 {
		t.Fatalf("expected 2 items, got %d", summary.ItemCount)
	}

	if summary.RiskyItemCount != 1 {
		t.Fatalf("expected 1 risky item, got %d", summary.RiskyItemCount)
	}

	if summary.TotalCostAmount != 236.5 {
		t.Fatalf("unexpected total cost amount: %.2f", summary.TotalCostAmount)
	}

	if summary.Suppliers[0].Items[0].RiskLevel != "warning" {
		t.Fatalf("expected warning risk level, got %s", summary.Suppliers[0].Items[0].RiskLevel)
	}
}

func TestRenderProcurementCSV(t *testing.T) {
	summary := buildProcurementSummary(
		"file",
		supplier.ConnectorCapabilities{},
		"PO-001",
		"",
		"",
		[]procurementCatalogItem{
			{
				SupplierCode:       "SUP_A",
				OriginalSKU:        "SKU-1",
				Title:              "谷饲肥牛卷 500g",
				NormalizedCategory: "冷冻肉类/肥牛",
				Quantity:           2,
				SalesUnit:          "盒",
				CostPrice:          36.5,
				BusinessPrice:      45,
				ConsumerPrice:      59.9,
				NeedColdChain:      true,
			},
		},
	)

	content, err := renderProcurementCSV(summary)
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}

	if !strings.Contains(content, "supplier_code,external_ref,original_sku") {
		t.Fatalf("csv header missing: %s", content)
	}

	if !strings.Contains(content, "SUP_A,PO-001,SKU-1") {
		t.Fatalf("csv row missing: %s", content)
	}
}

func TestApplyProcurementStatus(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("procurement_orders"))
	record.Set("status", ProcurementStatusDraft)

	if err := applyProcurementStatus(record, ProcurementStatusReviewed, "checked"); err != nil {
		t.Fatalf("review transition failed: %v", err)
	}

	if record.GetString("status") != ProcurementStatusReviewed {
		t.Fatalf("unexpected status: %s", record.GetString("status"))
	}

	if record.GetString("reviewed_at") == "" {
		t.Fatal("expected reviewed_at to be set")
	}

	if err := applyProcurementStatus(record, ProcurementStatusReceived, "skipped"); err == nil {
		t.Fatal("expected invalid transition to fail")
	}
}

func TestNormalizeSourceReviewFilter(t *testing.T) {
	filter := normalizeSourceReviewFilter(SourceReviewFilter{
		ProductStatus: " Approved ",
		AssetStatus:   " Failed ",
		SyncState:     " Synced ",
		Query:         "  chicken  ",
	})

	if filter.ProductStatus != "approved" {
		t.Fatalf("unexpected product status: %q", filter.ProductStatus)
	}
	if filter.AssetStatus != "failed" {
		t.Fatalf("unexpected asset status: %q", filter.AssetStatus)
	}
	if filter.SyncState != "synced" {
		t.Fatalf("unexpected sync state: %q", filter.SyncState)
	}
	if filter.Query != "chicken" {
		t.Fatalf("unexpected query: %q", filter.Query)
	}
	if filter.ProductPage != 1 || filter.AssetPage != 1 {
		t.Fatalf("expected default pages to be 1, got product=%d asset=%d", filter.ProductPage, filter.AssetPage)
	}
	if filter.PageSize != 24 {
		t.Fatalf("expected default page size 24, got %d", filter.PageSize)
	}
}

func TestSortAssetFailureReasons(t *testing.T) {
	reasons := sortAssetFailureReasons(map[string]int{
		"timeout":          3,
		"decode failed":    5,
		"bad source image": 5,
		"network":          1,
		"empty":            2,
		"overflow":         4,
	})

	expected := []SourceAssetFailureReason{
		{Message: "bad source image", Count: 5},
		{Message: "decode failed", Count: 5},
		{Message: "overflow", Count: 4},
		{Message: "timeout", Count: 3},
		{Message: "empty", Count: 2},
	}

	if !reflect.DeepEqual(reasons, expected) {
		t.Fatalf("unexpected sorted reasons: %#v", reasons)
	}
}

func TestRecordPrimaryAssetURLSkipsMockFallback(t *testing.T) {
	service := &Service{
		cfg: config.Config{
			App: config.AppConfig{PublicURL: "http://127.0.0.1:26228"},
		},
	}
	record := core.NewRecord(core.NewBaseCollection("supplier_products"))
	record.Set("processed_image", "mock.svg")
	record.Set("processed_image_source", "mock")

	if got := service.recordPrimaryAssetURL(record); got != "" {
		t.Fatalf("expected empty primary asset url when only mock processed image exists, got %q", got)
	}
}

func TestRecordPrimaryAssetURLUsesProcessedWhenNotMock(t *testing.T) {
	service := &Service{
		cfg: config.Config{
			App: config.AppConfig{PublicURL: "http://127.0.0.1:26228"},
		},
	}
	record := core.NewRecord(core.NewBaseCollection("supplier_products"))
	record.Set("processed_image", "real.png")
	record.Set("processed_image_source", "webhook")

	got := service.recordPrimaryAssetURL(record)
	if got == "" || !strings.Contains(got, "/api/files/") {
		t.Fatalf("expected file url from processed image, got %q", got)
	}
}

func TestSupplierRecordVariantPayloadsExpandsMultiUnit(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("supplier_products"))
	record.Set("supplier_code", "SUP_A")
	record.Set("original_sku", "671256064473223168")
	record.Set("source_product_id", "671256063491756032_671256064473223168")
	record.Set("source_type", "list_skeleton")
	record.Set("normalized_title", "宏业五星鸡块")
	record.Set("b_price", 105)
	record.Set("c_price", 105)
	record.Set("cost_price", 50)
	record.Set("currency_code", "CNY")
	record.Set("vendure_variant_id", "665")
	record.Set("supplier_payload", `{"sales_unit":"件","conversion_rate":10,"unit_options":[{"unitName":"件","price":105,"rate":10,"isDefault":true,"stockQty":0.6},{"unitName":"袋","price":11,"rate":1,"isDefault":false,"stockQty":6}]}`)

	variants := supplierRecordVariantPayloads(record, 100)
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	defaultVariant := defaultVariantPayload(variants)
	if defaultVariant.SalesUnit != "件" {
		t.Fatalf("expected default unit 件, got %q", defaultVariant.SalesUnit)
	}
	if defaultVariant.SKU != "671256064473223168" {
		t.Fatalf("expected default sku to stay original, got %q", defaultVariant.SKU)
	}
	if defaultVariant.VendureVariant != "665" {
		t.Fatalf("expected default variant id to reuse stored id, got %q", defaultVariant.VendureVariant)
	}
	if defaultVariant.DefaultStock != 1 {
		t.Fatalf("expected fractional stock to ceil to 1, got %d", defaultVariant.DefaultStock)
	}

	secondary := variants[1]
	if secondary.SalesUnit != "袋" {
		t.Fatalf("expected secondary unit 袋, got %q", secondary.SalesUnit)
	}
	if secondary.ConversionRate != 1 {
		t.Fatalf("expected secondary rate 1, got %v", secondary.ConversionRate)
	}
	if secondary.ConsumerPrice != 1100 {
		t.Fatalf("expected secondary price 1100, got %d", secondary.ConsumerPrice)
	}
	if secondary.DefaultStock != 6 {
		t.Fatalf("expected secondary stock 6, got %d", secondary.DefaultStock)
	}
	if secondary.SKU == "671256064473223168" || !strings.Contains(secondary.SKU, "__unit_") {
		t.Fatalf("expected derived SKU for secondary unit, got %q", secondary.SKU)
	}
}

func TestSupplierRecordVariantPayloadsReusesStoredVariantMapping(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("supplier_products"))
	record.Set("supplier_code", "SUP_A")
	record.Set("original_sku", "SKU-1")
	record.Set("source_product_id", "source-product")
	record.Set("source_type", "snapshot")
	record.Set("normalized_title", "测试商品")
	record.Set("b_price", 30)
	record.Set("c_price", 30)
	record.Set("currency_code", "CNY")
	record.Set("supplier_payload", `{"sales_unit":"件","conversion_rate":10,"unit_options":[{"unitName":"件","price":30,"rate":10,"isDefault":true,"stockQty":1},{"unitName":"袋","price":4,"rate":1,"isDefault":false,"stockQty":5}]}`)

	stored := []vendureVariantState{
		{
			Key:              supplierVariantKey("source-product", "袋", 1),
			UnitName:         "袋",
			Rate:             1,
			SKU:              "SKU-1__bag",
			VendureVariantID: "777",
		},
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		t.Fatalf("marshal stored states: %v", err)
	}
	record.Set("vendure_variants_json", string(encoded))

	variants := supplierRecordVariantPayloads(record, 100)
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	var bagVariant *vendure.ProductVariantPayload
	for i := range variants {
		if variants[i].SalesUnit == "袋" {
			bagVariant = &variants[i]
			break
		}
	}
	if bagVariant == nil {
		t.Fatal("expected to find 袋 variant")
	}
	if bagVariant.VendureVariant != "777" {
		t.Fatalf("expected stored variant id 777, got %q", bagVariant.VendureVariant)
	}
	if bagVariant.SKU != "SKU-1__bag" {
		t.Fatalf("expected stored sku to be reused, got %q", bagVariant.SKU)
	}
}

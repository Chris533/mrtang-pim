package pim

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/supplier"
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

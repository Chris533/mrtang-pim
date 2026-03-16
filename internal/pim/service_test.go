package pim

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

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

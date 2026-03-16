package supplier

import (
	"context"
	"testing"
)

func TestFileConnectorCapabilities(t *testing.T) {
	connector := NewFileConnector("./fixtures.json", "SUP_A")
	capabilities := connector.Capabilities()

	if !capabilities.FetchProducts {
		t.Fatal("expected file connector to support product fetch")
	}

	if capabilities.SubmitPurchaseOrder {
		t.Fatal("expected file connector to reject live purchase order submission")
	}

	if !capabilities.ExportPurchaseOrder {
		t.Fatal("expected file connector to support purchase order export")
	}
}

func TestFileConnectorSubmitPurchaseOrder(t *testing.T) {
	connector := NewFileConnector("./fixtures.json", "SUP_A")
	result, err := connector.SubmitPurchaseOrder(context.Background(), PurchaseOrder{
		SupplierCode: "SUP_A",
		ExternalRef:  "PO-001",
		Items: []PurchaseOrderItem{
			{SupplierCode: "SUP_A", OriginalSKU: "SKU-1", Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("submit purchase order: %v", err)
	}

	if result.Mode != "manual_export" {
		t.Fatalf("expected manual_export mode, got %s", result.Mode)
	}

	if result.Accepted {
		t.Fatal("expected file connector submission to remain unaccepted")
	}
}

package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type FileConnector struct {
	path string
	code string
}

func NewFileConnector(path string, code string) *FileConnector {
	return &FileConnector{
		path: path,
		code: code,
	}
}

func (c *FileConnector) Capabilities() ConnectorCapabilities {
	return ConnectorCapabilities{
		FetchProducts:       true,
		SubmitPurchaseOrder: false,
		ExportPurchaseOrder: true,
	}
}

func (c *FileConnector) Fetch(ctx context.Context) ([]Product, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	body, err := os.ReadFile(c.path)
	if err != nil {
		return nil, fmt.Errorf("read supplier file: %w", err)
	}

	var products []Product
	if err := json.Unmarshal(body, &products); err != nil {
		return nil, fmt.Errorf("decode supplier file: %w", err)
	}

	for i := range products {
		if products[i].SupplierCode == "" {
			products[i].SupplierCode = c.code
		}

		if products[i].CurrencyCode == "" {
			products[i].CurrencyCode = "CNY"
		}
	}

	return products, nil
}

func (c *FileConnector) SubmitPurchaseOrder(ctx context.Context, order PurchaseOrder) (PurchaseOrderResult, error) {
	select {
	case <-ctx.Done():
		return PurchaseOrderResult{}, ctx.Err()
	default:
	}

	return PurchaseOrderResult{
		SupplierCode: order.SupplierCode,
		ExternalRef:  order.ExternalRef,
		Mode:         "manual_export",
		Accepted:     false,
		Message:      "file connector does not support live purchase order submission",
	}, nil
}

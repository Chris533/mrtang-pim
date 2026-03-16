package supplier

import (
	"context"
	"time"
)

type Product struct {
	SupplierCode      string                 `json:"supplier_code"`
	OriginalSKU       string                 `json:"original_sku"`
	RawTitle          string                 `json:"raw_title"`
	RawDescription    string                 `json:"raw_description"`
	RawCategory       string                 `json:"raw_category"`
	RawImageURL       string                 `json:"raw_image_url"`
	CostPrice         float64                `json:"cost_price"`
	BPrice            float64                `json:"b_price"`
	CPrice            float64                `json:"c_price"`
	CurrencyCode      string                 `json:"currency_code"`
	SupplierUpdatedAt time.Time              `json:"supplier_updated_at"`
	Payload           map[string]any         `json:"payload"`
	Attributes        map[string]interface{} `json:"attributes"`
}

type ConnectorCapabilities struct {
	FetchProducts       bool `json:"fetchProducts"`
	SubmitPurchaseOrder bool `json:"submitPurchaseOrder"`
	ExportPurchaseOrder bool `json:"exportPurchaseOrder"`
}

type PurchaseOrderItem struct {
	SupplierCode string  `json:"supplierCode"`
	OriginalSKU  string  `json:"originalSku"`
	Quantity     float64 `json:"quantity"`
	SalesUnit    string  `json:"salesUnit,omitempty"`
}

type PurchaseOrder struct {
	SupplierCode    string              `json:"supplierCode"`
	ExternalRef     string              `json:"externalRef,omitempty"`
	DeliveryAddress string              `json:"deliveryAddress,omitempty"`
	Notes           string              `json:"notes,omitempty"`
	Items           []PurchaseOrderItem `json:"items"`
}

type PurchaseOrderResult struct {
	SupplierCode string `json:"supplierCode"`
	ExternalRef  string `json:"externalRef,omitempty"`
	Mode         string `json:"mode"`
	Accepted     bool   `json:"accepted"`
	Message      string `json:"message,omitempty"`
}

type Connector interface {
	Fetch(ctx context.Context) ([]Product, error)
	Capabilities() ConnectorCapabilities
	SubmitPurchaseOrder(ctx context.Context, order PurchaseOrder) (PurchaseOrderResult, error)
}

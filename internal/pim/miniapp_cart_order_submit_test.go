package pim

import (
	"testing"

	miniappmodel "mrtang-pim/internal/miniapp/model"
)

func TestBuildMiniappAddCartBodyIncludesSkuName(t *testing.T) {
	body := buildMiniappAddCartBody([]miniappCartOrderLine{
		{
			SkuID:     "sku-1",
			SpuID:     "spu-1",
			SkuName:   "鸭血",
			UnitID:    "unit-1",
			Quantity:  2,
			UnitPrice: 18,
		},
	})

	if got := body[0]["skuName"]; got != "鸭血" {
		t.Fatalf("expected skuName to be preserved, got %v", got)
	}
}

func TestMatchCartDetailLinesUsesFreightQty(t *testing.T) {
	cartIDs, freightItems, goodsAmount, err := matchCartDetailLines(
		[]miniappCartOrderLine{
			{
				SkuID:      "sku-1",
				SalesUnit:  "件",
				Quantity:   2,
				FreightQty: 20,
				UnitPrice:  18,
			},
		},
		[]rawDetailLine{
			{
				ID:       "cart-1",
				SkuID:    "sku-1",
				UnitName: "件",
				Num:      2,
				TotPrice: 36,
			},
		},
	)
	if err != nil {
		t.Fatalf("matchCartDetailLines returned error: %v", err)
	}
	if len(cartIDs) != 1 || cartIDs[0] != "cart-1" {
		t.Fatalf("unexpected cartIDs: %#v", cartIDs)
	}
	if len(freightItems) != 1 {
		t.Fatalf("unexpected freightItems: %#v", freightItems)
	}
	if got := freightItems[0]["qty"]; got != "20" {
		t.Fatalf("expected converted freight qty, got %v", got)
	}
	if goodsAmount != 36 {
		t.Fatalf("expected goodsAmount 36, got %v", goodsAmount)
	}
}

func TestResolvedMiniappOrderUnitPrefersSalesUnit(t *testing.T) {
	product := &miniappmodel.ProductPage{
		Context: miniappmodel.ProductContext{
			UnitOptions: []miniappmodel.ProductOrderUnit{
				{UnitID: "unit-default", UnitName: "袋", Rate: 1, IsDefault: true},
				{UnitID: "unit-piece", UnitName: "件", Rate: 10},
			},
		},
	}

	unit := resolvedMiniappOrderUnit(product, "件")
	if unit.UnitID != "unit-piece" {
		t.Fatalf("expected matching unit, got %#v", unit)
	}
}

func TestBuildMiniappSubmitVerificationVerified(t *testing.T) {
	status, message, details := buildMiniappSubmitVerification(
		ProcurementSummary{ExternalRef: "ORDER-1"},
		ProcurementSupplierSummary{SupplierCode: "SUP_A", ItemCount: 1, TotalQty: 1},
		[]miniappCartOrderLine{{OriginalSKU: "sku-1", SpuID: "spu-1", SkuID: "sku-1", SalesUnit: "件", UnitID: "unit-1", Quantity: 1, FreightQty: 1, UnitPrice: 295}},
		[]string{"cart-1"},
		"delivery-1",
		295,
		0,
		map[string]any{"customerId": "customer-1"},
		map[string]any{
			"code":    "SYS_0000",
			"message": "请选择支付方式进行支付",
			"data": map[string]any{
				"billId":           "bill-1",
				"dueMoney":         295,
				"whetherOpenWxPay": true,
				"deadlineTime":     1773654185111.0,
				"customerName":     "Chris",
				"openWxPayList": []any{
					map[string]any{"name": "微信支付", "type": 13.0, "payRecommend": 0.0},
				},
			},
		},
		map[string]any{
			"code":    "SYS_0000",
			"message": "访问成功",
			"data": map[string]any{
				"id":             "bill-1",
				"billNo":         "XD-1",
				"dueMoney":       295,
				"goodsTypeCount": 1,
				"goodsCount":     1,
				"remark":         "vendure-order:ORDER-1",
				"buyerRemark":    "vendure-order:ORDER-1",
				"receiveAddressInfo": map[string]any{
					"deliveryMethodId": "delivery-1",
				},
				"orderGoods": []any{
					map[string]any{
						"spuId":         "spu-1",
						"skuId":         "sku-1",
						"skuName":       "鸭血",
						"unitId":        "unit-1",
						"unitName":      "件",
						"originalPrice": 295,
						"subTotal":      295,
						"unitRate":      1,
					},
				},
			},
		},
		nil,
	)

	if status != "verified" {
		t.Fatalf("expected verified status, got %q", status)
	}
	if message != "supplier submit response verified" {
		t.Fatalf("unexpected verification message: %q", message)
	}
	submit, ok := details["submit"].(map[string]any)
	if !ok {
		t.Fatalf("expected submit details, got %#v", details)
	}
	if submit["billId"] != "bill-1" {
		t.Fatalf("unexpected billId: %#v", submit["billId"])
	}
	if submit["paymentOptionCnt"] != 1 {
		t.Fatalf("unexpected payment option count: %#v", submit["paymentOptionCnt"])
	}
}

func TestBuildMiniappSubmitVerificationWarningOnMismatch(t *testing.T) {
	status, message, details := buildMiniappSubmitVerification(
		ProcurementSummary{ExternalRef: "ORDER-2"},
		ProcurementSupplierSummary{SupplierCode: "SUP_A", ItemCount: 1, TotalQty: 1},
		[]miniappCartOrderLine{{OriginalSKU: "sku-1", SpuID: "spu-1", SkuID: "sku-1", SalesUnit: "件", UnitID: "unit-1", Quantity: 1, FreightQty: 1, UnitPrice: 295}},
		[]string{"cart-1"},
		"delivery-1",
		295,
		0,
		map[string]any{"customerId": "customer-1"},
		map[string]any{
			"code":    "SYS_0000",
			"message": "",
			"data": map[string]any{
				"billId":           "",
				"dueMoney":         300,
				"whetherOpenWxPay": false,
				"openWxPayList":    []any{},
			},
		},
		nil,
		nil,
	)

	if status != "warning" {
		t.Fatalf("expected warning status, got %q", status)
	}
	if message == "" {
		t.Fatal("expected non-empty warning message")
	}
	verification, ok := details["verification"].(map[string]any)
	if !ok {
		t.Fatalf("expected verification details, got %#v", details)
	}
	issues, ok := verification["issues"].([]string)
	if !ok {
		t.Fatalf("expected typed issues slice, got %#v", verification["issues"])
	}
	if len(issues) < 3 {
		t.Fatalf("expected multiple issues, got %#v", issues)
	}
}

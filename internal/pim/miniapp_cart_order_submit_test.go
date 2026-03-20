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

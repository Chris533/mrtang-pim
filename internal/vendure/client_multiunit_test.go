package vendure

import (
	"testing"

	"mrtang-pim/internal/config"
)

func TestBuildVariantCustomFieldsContainsUnitRateAndBPrice(t *testing.T) {
	client := &Client{
		cfg: config.VendureConfig{
			LanguageCode:               "zh_Hans",
			VariantConversionRateField: "conversionRate",
			VariantSourceProductField:  "sourceProductId",
			VariantSourceTypeField:     "sourceType",
		},
	}

	customFields := client.buildVariantCustomFields(ProductVariantPayload{
		SalesUnit:       "bag",
		BusinessPrice:   1100,
		ConversionRate:  1,
		SourceProductID: "src-001",
		SourceType:      "list_skeleton",
	})

	if got := customFields["salesUnit"]; got != "bag" {
		t.Fatalf("expected salesUnit=bag, got %v", got)
	}
	if got := customFields["bPrice"]; got != 1100 {
		t.Fatalf("expected bPrice=1100, got %v", got)
	}
	if got := customFields["conversionRate"]; got != float64(1) {
		t.Fatalf("expected conversionRate=1, got %v", got)
	}
	if got := customFields["sourceProductId"]; got != "src-001" {
		t.Fatalf("expected sourceProductId=src-001, got %v", got)
	}
	if got := customFields["sourceType"]; got != "list_skeleton" {
		t.Fatalf("expected sourceType=list_skeleton, got %v", got)
	}
}

func TestBuildCreateVariantInputIncludesOptionIDs(t *testing.T) {
	client := &Client{
		cfg: config.VendureConfig{
			LanguageCode: "zh_Hans",
			CurrencyCode: "CNY",
			ChannelToken: "default-channel",
		},
	}

	input := client.buildCreateVariantInput(ProductVariantPayload{
		Name:          "件",
		SKU:           "sku-001",
		ConsumerPrice: 10500,
		DefaultStock:  100,
		SalesUnit:     "件",
		CurrencyCode:  "CNY",
		BusinessPrice: 10500,
		OptionIDs:     []string{"101"},
	}, "665", "900")

	optionIDs, ok := input["optionIds"].([]string)
	if !ok {
		t.Fatalf("expected optionIds to be []string, got %T", input["optionIds"])
	}
	if len(optionIDs) != 1 || optionIDs[0] != "101" {
		t.Fatalf("expected optionIds [101], got %#v", optionIDs)
	}
	if got := input["productId"]; got != "665" {
		t.Fatalf("expected productId=665, got %v", got)
	}
}

func TestBuildUpdateVariantInputIncludesOptionIDs(t *testing.T) {
	client := &Client{
		cfg: config.VendureConfig{
			LanguageCode: "zh_Hans",
		},
	}

	input := client.buildUpdateVariantInput(ProductVariantPayload{
		VendureVariant: "777",
		Name:           "袋",
		SKU:            "sku-002",
		ConsumerPrice:  1100,
		DefaultStock:   100,
		SalesUnit:      "袋",
		CurrencyCode:   "CNY",
		BusinessPrice:  1100,
		OptionIDs:      []string{"202"},
	}, "901")

	optionIDs, ok := input["optionIds"].([]string)
	if !ok {
		t.Fatalf("expected optionIds to be []string, got %T", input["optionIds"])
	}
	if len(optionIDs) != 1 || optionIDs[0] != "202" {
		t.Fatalf("expected optionIds [202], got %#v", optionIDs)
	}
	if got := input["id"]; got != "777" {
		t.Fatalf("expected id=777, got %v", got)
	}
}

func TestSalesUnitOptionCodesAreStable(t *testing.T) {
	if got := salesUnitOptionGroupCode("665"); got != "pim-sales-unit-665" {
		t.Fatalf("unexpected option group code: %s", got)
	}
	if got := salesUnitOptionCode("件"); got == "" || got == "件" {
		t.Fatalf("expected hashed ASCII option code, got %q", got)
	}
	if salesUnitOptionCode("件") != salesUnitOptionCode("件") {
		t.Fatal("expected option code to be stable")
	}
	if salesUnitOptionCode("件") == salesUnitOptionCode("袋") {
		t.Fatal("expected different units to map to different option codes")
	}
}

func TestSalesUnitNamesDeduplicatesUnits(t *testing.T) {
	units := salesUnitNames([]ProductVariantPayload{
		{SalesUnit: "件"},
		{SalesUnit: "袋"},
		{SalesUnit: "件"},
		{SalesUnit: " "},
	})

	if len(units) != 2 {
		t.Fatalf("expected 2 unique units, got %#v", units)
	}
	if units[0] != "件" || units[1] != "袋" {
		t.Fatalf("unexpected units order/content: %#v", units)
	}
}

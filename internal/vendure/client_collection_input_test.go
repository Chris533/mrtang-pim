package vendure

import (
	"testing"

	"mrtang-pim/internal/config"
)

func TestCollectionInputForCreateIncludesEmptyFilters(t *testing.T) {
	client := &Client{cfg: config.VendureConfig{LanguageCode: "zh_Hans"}}
	payload := CollectionPayload{
		Name:                "打包盒系列",
		Slug:                "da-bao-he-xi-lie",
		Description:         "desc",
		SourceCategoryKey:   "k1",
		SourceCategoryPath:  "打包盒系列",
		SourceCategoryLevel: 1,
		ParentCollectionID:  "10",
	}

	input := client.collectionInput(payload, true)

	if _, ok := input["filters"]; !ok {
		t.Fatalf("create input must include filters")
	}
	if _, ok := input["inheritFilters"]; !ok {
		t.Fatalf("create input must include inheritFilters")
	}
	if got := input["parentId"]; got != payload.ParentCollectionID {
		t.Fatalf("create input parentId mismatch: got %v want %v", got, payload.ParentCollectionID)
	}
}

func TestCollectionInputForUpdateDoesNotOverrideFilters(t *testing.T) {
	client := &Client{cfg: config.VendureConfig{LanguageCode: "zh_Hans"}}
	payload := CollectionPayload{
		Name:                "打包盒系列",
		Slug:                "da-bao-he-xi-lie",
		Description:         "desc",
		SourceCategoryKey:   "k1",
		SourceCategoryPath:  "打包盒系列",
		SourceCategoryLevel: 1,
		ParentCollectionID:  "10",
	}

	input := client.collectionInput(payload, false)

	if _, ok := input["filters"]; ok {
		t.Fatalf("update input must not include filters")
	}
	if _, ok := input["inheritFilters"]; !ok {
		t.Fatalf("update input must include inheritFilters")
	}
	if got := input["parentId"]; got != payload.ParentCollectionID {
		t.Fatalf("update input parentId mismatch: got %v want %v", got, payload.ParentCollectionID)
	}
}

func TestBuildProductCustomFieldsSkipsEmptyRelationID(t *testing.T) {
	client := &Client{cfg: config.VendureConfig{
		ProductTargetAudienceField: "targetAudience",
		ProductCEndAssetField:      "cEndFeaturedAsset",
	}}

	customFields := client.buildProductCustomFields(ProductPayload{
		TargetAudience: "ALL",
	}, "")

	if _, ok := customFields["targetAudience"]; !ok {
		t.Fatalf("expected target audience field to be set")
	}
	if _, ok := customFields["cEndFeaturedAssetId"]; ok {
		t.Fatalf("relation id field must be omitted when c-end asset id is empty")
	}
}

func TestBuildProductCustomFieldsIncludesRelationIDWhenPresent(t *testing.T) {
	client := &Client{cfg: config.VendureConfig{
		ProductCEndAssetField: "cEndFeaturedAsset",
	}}

	customFields := client.buildProductCustomFields(ProductPayload{}, "123")

	if got, ok := customFields["cEndFeaturedAssetId"]; !ok || got != "123" {
		t.Fatalf("expected relation id field cEndFeaturedAssetId=123, got=%v exists=%v", got, ok)
	}
}

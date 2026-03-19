package pim

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	miniappmodel "mrtang-pim/internal/miniapp/model"
)

type multiUnitFixture struct {
	SourceProductID string                    `json:"sourceProductId"`
	BaseSKU         string                    `json:"baseSKU"`
	DefaultUnit     string                    `json:"defaultUnit"`
	UnitOptions     []miniappmodel.UnitOption `json:"unitOptions"`
}

type plannedVariantSpec struct {
	UnitKey   string
	UnitName  string
	Rate      float64
	IsDefault bool
	SKU       string
}

func TestSourceProductConversionRatePrefersMarkedDefaultUnit(t *testing.T) {
	t.Skip("pending collection-backed integration test for PocketBase JSON field coercion")
	record := newSourceProductTestRecord()
	record.Set("default_unit", "bag")
	encoded, err := json.Marshal([]miniappmodel.UnitOption{
		{UnitName: "bag", Rate: 1, IsDefault: false},
		{UnitName: "case", Rate: 10, IsDefault: true},
	})
	if err != nil {
		t.Fatalf("marshal unit_options_json: %v", err)
	}
	record.Set("unit_options_json", string(encoded))

	got := sourceProductConversionRate(record)
	if got != 10 {
		t.Fatalf("expected default-marked rate=10, got %v", got)
	}
}

func TestSourceProductConversionRateFallsBackToDefaultUnitName(t *testing.T) {
	t.Skip("pending collection-backed integration test for PocketBase JSON field coercion")
	record := newSourceProductTestRecord()
	record.Set("default_unit", "case")
	encoded, err := json.Marshal([]miniappmodel.UnitOption{
		{UnitName: "bag", Rate: 1, IsDefault: false},
		{UnitName: "case", Rate: 10, IsDefault: false},
	})
	if err != nil {
		t.Fatalf("marshal unit_options_json: %v", err)
	}
	record.Set("unit_options_json", string(encoded))

	got := sourceProductConversionRate(record)
	if got != 10 {
		t.Fatalf("expected default unit fallback rate=10, got %v", got)
	}
}

func TestSourceProductConversionRateReturnsOneOnInvalidPayload(t *testing.T) {
	t.Skip("pending collection-backed integration test for PocketBase JSON field coercion")
	record := newSourceProductTestRecord()
	record.Set("default_unit", "case")
	record.Set("unit_options_json", `{"unexpected":"shape"}`)

	got := sourceProductConversionRate(record)
	if got != 1 {
		t.Fatalf("expected fallback rate=1 for invalid payload, got %v", got)
	}
}

func TestPlannedVariantSpecsAreStableAcrossUnitOrder(t *testing.T) {
	fixture := loadMultiUnitFixture(t)
	ordered := plannedVariantSpecsForTest(fixture.SourceProductID, fixture.BaseSKU, fixture.DefaultUnit, fixture.UnitOptions)

	reversed := append([]miniappmodel.UnitOption{}, fixture.UnitOptions...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	reordered := plannedVariantSpecsForTest(fixture.SourceProductID, fixture.BaseSKU, fixture.DefaultUnit, reversed)

	if len(ordered) != len(reordered) {
		t.Fatalf("variant count mismatch: ordered=%d reordered=%d", len(ordered), len(reordered))
	}

	orderedByKey := make(map[string]plannedVariantSpec, len(ordered))
	for _, spec := range ordered {
		orderedByKey[spec.UnitKey] = spec
	}
	for _, spec := range reordered {
		other, ok := orderedByKey[spec.UnitKey]
		if !ok {
			t.Fatalf("unexpected unit key in reordered result: %s", spec.UnitKey)
		}
		if other.SKU != spec.SKU {
			t.Fatalf("sku mismatch for unit key %s: ordered=%s reordered=%s", spec.UnitKey, other.SKU, spec.SKU)
		}
	}
}

func TestPlannedVariantSpecsKeepBaseSKUForDefaultUnit(t *testing.T) {
	fixture := loadMultiUnitFixture(t)
	specs := plannedVariantSpecsForTest(fixture.SourceProductID, fixture.BaseSKU, fixture.DefaultUnit, fixture.UnitOptions)

	var defaultSpec *plannedVariantSpec
	for i := range specs {
		if specs[i].IsDefault {
			defaultSpec = &specs[i]
			break
		}
	}
	if defaultSpec == nil {
		t.Fatal("expected default variant spec")
	}
	if defaultSpec.SKU != fixture.BaseSKU {
		t.Fatalf("default variant sku should keep base sku, got %s want %s", defaultSpec.SKU, fixture.BaseSKU)
	}
}

func TestBuildVendurePayloadMultiUnitExpansionSkeleton(t *testing.T) {
	t.Skip("pending production implementation: buildVendurePayload should expand unit_options into multiple variants")
}

func loadMultiUnitFixture(t *testing.T) multiUnitFixture {
	t.Helper()
	fixturePath := filepath.Join("testdata", "multi_unit_product_summary.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	var fixture multiUnitFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", fixturePath, err)
	}
	return fixture
}

func plannedVariantSpecsForTest(
	sourceProductID string,
	baseSKU string,
	defaultUnit string,
	unitOptions []miniappmodel.UnitOption,
) []plannedVariantSpec {
	if len(unitOptions) == 0 {
		return nil
	}

	defaultUnit = strings.TrimSpace(defaultUnit)
	specs := make([]plannedVariantSpec, 0, len(unitOptions))
	for _, option := range unitOptions {
		unitName := strings.TrimSpace(option.UnitName)
		if unitName == "" {
			continue
		}
		rate := option.Rate
		if rate <= 0 {
			rate = 1
		}
		isDefault := option.IsDefault || (defaultUnit != "" && strings.EqualFold(unitName, defaultUnit))
		unitKey := fmt.Sprintf("%s#%s#%.6f", strings.TrimSpace(sourceProductID), unitName, rate)
		sku := baseSKU
		if !isDefault {
			sku = derivedSKUForTest(baseSKU, unitKey)
		}
		specs = append(specs, plannedVariantSpec{
			UnitKey:   unitKey,
			UnitName:  unitName,
			Rate:      rate,
			IsDefault: isDefault,
			SKU:       sku,
		})
	}

	sort.SliceStable(specs, func(i, j int) bool {
		if specs[i].IsDefault != specs[j].IsDefault {
			return specs[i].IsDefault
		}
		return specs[i].UnitKey < specs[j].UnitKey
	})

	hasDefault := false
	for _, spec := range specs {
		if spec.IsDefault {
			hasDefault = true
			break
		}
	}
	if !hasDefault && len(specs) > 0 {
		specs[0].IsDefault = true
		specs[0].SKU = baseSKU
	}

	return specs
}

func derivedSKUForTest(baseSKU string, unitKey string) string {
	hash := sha1.Sum([]byte(unitKey))
	return fmt.Sprintf("%s__u_%s", strings.TrimSpace(baseSKU), hex.EncodeToString(hash[:4]))
}

func newSourceProductTestRecord() *core.Record {
	collection := core.NewBaseCollection(CollectionSourceProducts)
	collection.Fields.Add(
		&core.TextField{Name: "default_unit"},
		&core.TextField{Name: "unit_options_json"},
	)
	return core.NewRecord(collection)
}

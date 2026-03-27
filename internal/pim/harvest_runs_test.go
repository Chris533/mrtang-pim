package pim

import "testing"

func TestHarvestRunStatus(t *testing.T) {
	if status := harvestRunStatus(harvestExecutionState{}); status != HarvestRunStatusSuccess {
		t.Fatalf("expected success, got %q", status)
	}
	if status := harvestRunStatus(harvestExecutionState{Result: Result{Failed: 2}}); status != HarvestRunStatusPartial {
		t.Fatalf("expected partial, got %q", status)
	}
	if status := harvestRunStatus(harvestExecutionState{errorMessage: "boom"}); status != HarvestRunStatusFailed {
		t.Fatalf("expected failed, got %q", status)
	}
}

func TestAppendHarvestFailureRespectsLimit(t *testing.T) {
	items := make([]HarvestFailureItem, 0, harvestRunFailureItemLimit)
	for i := 0; i < harvestRunFailureItemLimit+5; i++ {
		items = appendHarvestFailure(items, HarvestFailureItem{
			SKU:   "SKU",
			Step:  "upsert",
			Error: "failed",
		})
	}
	if len(items) != harvestRunFailureItemLimit {
		t.Fatalf("expected %d items, got %d", harvestRunFailureItemLimit, len(items))
	}
}

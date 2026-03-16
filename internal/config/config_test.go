package config

import "testing"

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" pim, supplier , ,images ")
	want := []string{"pim", "supplier", "images"}

	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %d want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected value at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

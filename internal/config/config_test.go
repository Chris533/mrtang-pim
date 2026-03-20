package config

import (
	"os"
	"strings"
	"testing"
)

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

func TestValidateRuntimeRequiresRawMiniappForMiniappCartOrder(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("MRTANG_PIM_ENCRYPTION_KEY", "secret")
	cfg := Config{
		App: AppConfig{PublicURL: "https://pim.example.com"},
		Security: SecurityConfig{
			APIKey: "test-api-key",
		},
		MiniApp: MiniAppConfig{
			SourceMode:          "snapshot",
			AuthorizedAccountID: "account",
			RawOpenID:           "openid",
			RawCustomerID:       "customer",
			HomepageSnapshotFile:  os.TempDir(),
			CategorySnapshotFile:  os.TempDir(),
			ProductSnapshotFile:   os.TempDir(),
			CartOrderSnapshotFile: os.TempDir(),
		},
		Supplier: SupplierConfig{
			Connector: "miniapp_cart_order",
			Code:      "SUP_A",
		},
	}

	err := ValidateRuntime(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "MINIAPP_SOURCE_MODE=raw is required when SUPPLIER_CONNECTOR=miniapp_cart_order") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

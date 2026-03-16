package service

import (
	"context"
	"testing"

	"mrtang-pim/internal/miniapp/api"
)

func TestSnapshotServiceLoadsHomepageAndSection(t *testing.T) {
	svc := New(api.NewSnapshotSource("../../../datasets/miniapp_homepage_snapshot.json"), nil)

	dataset, err := svc.Dataset(context.Background())
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	if len(dataset.Contracts) == 0 {
		t.Fatal("expected contracts")
	}

	homepage, err := svc.Homepage(context.Background())
	if err != nil {
		t.Fatalf("load homepage: %v", err)
	}

	if len(homepage.Sections) == 0 {
		t.Fatal("expected homepage sections")
	}

	section, err := svc.Section(context.Background(), "new")
	if err != nil {
		t.Fatalf("load section: %v", err)
	}

	if section == nil || len(section.Products) == 0 {
		t.Fatal("expected section products")
	}
}

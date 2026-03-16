package service

import (
	"context"
	"testing"

	"mrtang-pim/internal/miniapp/api"
)

func TestSnapshotServiceLoadsHomepageAndSection(t *testing.T) {
	svc := New(
		api.NewSnapshotSource(
			"../../../datasets/miniapp/homepage",
			"../../../datasets/miniapp/category-page",
		),
		nil,
	)

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

	categoryPage, err := svc.CategoryPage(context.Background())
	if err != nil {
		t.Fatalf("load category page: %v", err)
	}

	if len(categoryPage.Tree) == 0 {
		t.Fatal("expected category page tree")
	}
	if len(categoryPage.Tree) != 18 {
		t.Fatalf("expected full category page tree, got %d top-level categories", len(categoryPage.Tree))
	}
	if len(categoryPage.Sections) != 18 {
		t.Fatalf("expected category page sections for all top-level categories, got %d", len(categoryPage.Sections))
	}

	categorySection, err := svc.CategorySection(context.Background(), "chicken")
	if err != nil {
		t.Fatalf("load category section: %v", err)
	}

	if categorySection == nil || len(categorySection.Products) == 0 {
		t.Fatal("expected category section products")
	}

	contracts, err := svc.Contracts(context.Background(), "/api/miniapp/category-page")
	if err != nil {
		t.Fatalf("load category contracts: %v", err)
	}

	if len(contracts) == 0 {
		t.Fatal("expected category page contracts")
	}
}

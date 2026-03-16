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
			"../../../datasets/miniapp/product-page",
			"../../../datasets/miniapp/cart-order",
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
	if section.Products[0].ProductID == "" {
		t.Fatal("expected homepage product id")
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
	if categorySection.Products[0].ProductID == "" {
		t.Fatal("expected category product id")
	}

	contracts, err := svc.Contracts(context.Background(), "/api/miniapp/category-page")
	if err != nil {
		t.Fatalf("load category contracts: %v", err)
	}

	if len(contracts) == 0 {
		t.Fatal("expected category page contracts")
	}

	productPage, err := svc.ProductPage(context.Background())
	if err != nil {
		t.Fatalf("load product page: %v", err)
	}

	if len(productPage.Products) == 0 {
		t.Fatal("expected product pages")
	}
	if len(productPage.Products) < 16 {
		t.Fatalf("expected product page coverage for visible products, got %d", len(productPage.Products))
	}

	product, err := svc.Product(context.Background(), "670168385396461568_670168388273754112")
	if err != nil {
		t.Fatalf("load product page item: %v", err)
	}

	if product == nil || product.Detail.Name == "" || len(product.Pricing.UnitOptions) == 0 {
		t.Fatal("expected product detail and pricing")
	}
	if product.ID == "" || product.Summary.ProductID == "" || product.ID != product.Summary.ProductID {
		t.Fatal("expected aligned product identifiers")
	}
	if product.SourceType != "rr_detail" {
		t.Fatalf("expected rr_detail source type, got %q", product.SourceType)
	}

	listProduct, err := svc.Product(context.Background(), section.Products[0].ProductID)
	if err != nil {
		t.Fatalf("load homepage-linked product page item: %v", err)
	}
	if listProduct == nil || listProduct.Summary.Name == "" {
		t.Fatal("expected homepage product to resolve into product page")
	}
	if listProduct.SourceType != "list_skeleton" || len(listProduct.SourceSections) == 0 {
		t.Fatal("expected list-derived product metadata")
	}

	upgradedCategoryProduct, err := svc.Product(context.Background(), "687609781015719936_687609781711974400")
	if err != nil {
		t.Fatalf("load upgraded category product page item: %v", err)
	}
	if upgradedCategoryProduct == nil || upgradedCategoryProduct.SourceType != "rr_detail" {
		t.Fatal("expected category product to be upgraded to rr_detail")
	}

	productContracts, err := svc.Contracts(context.Background(), "/api/miniapp/product-page")
	if err != nil {
		t.Fatalf("load product contracts: %v", err)
	}

	if len(productContracts) == 0 {
		t.Fatal("expected product page contracts")
	}

	coverage, err := svc.ProductCoverage(context.Background())
	if err != nil {
		t.Fatalf("load product coverage: %v", err)
	}
	if len(coverage) == 0 {
		t.Fatal("expected product coverage")
	}
	if coverage[0].Priority != "homepage_dual_unit" {
		t.Fatalf("expected homepage dual-unit products to be prioritized first, got %q", coverage[0].Priority)
	}

	coverageSummary, err := svc.ProductCoverageSummary(context.Background())
	if err != nil {
		t.Fatalf("load product coverage summary: %v", err)
	}
	if coverageSummary.TotalProducts < len(productPage.Products) {
		t.Fatal("expected coverage summary to include all product pages")
	}
	if len(coverageSummary.FirstBatch) == 0 || coverageSummary.FirstBatch[0].Priority != "homepage_dual_unit" {
		t.Fatal("expected homepage dual-unit first batch in coverage summary")
	}

	cartOrder, err := svc.CartOrder(context.Background())
	if err != nil {
		t.Fatalf("load cart order: %v", err)
	}
	if cartOrder.Cart.Add.ContractID == "" || cartOrder.Order.Submit.ContractID == "" {
		t.Fatal("expected cart-order operations")
	}
	if len(cartOrder.Order.FreightCosts) != 2 {
		t.Fatalf("expected two freight cost scenarios, got %d", len(cartOrder.Order.FreightCosts))
	}

	cartContracts, err := svc.Contracts(context.Background(), "/api/miniapp/cart-order")
	if err != nil {
		t.Fatalf("load cart-order contracts: %v", err)
	}
	if len(cartContracts) == 0 {
		t.Fatal("expected cart-order contracts")
	}

	addCart, err := svc.CartOperation(context.Background(), "add")
	if err != nil {
		t.Fatalf("load cart add operation: %v", err)
	}
	if addCart == nil || addCart.Response == nil {
		t.Fatal("expected cart add response")
	}

	submitOrder, err := svc.OrderOperation(context.Background(), "submit")
	if err != nil {
		t.Fatalf("load order submit operation: %v", err)
	}
	if submitOrder == nil || submitOrder.Response == nil {
		t.Fatal("expected order submit response")
	}

	selectedFreight, err := svc.FreightCost(context.Background(), "selected_delivery")
	if err != nil {
		t.Fatalf("load selected freight scenario: %v", err)
	}
	if selectedFreight == nil || selectedFreight.Scenario != "selected_delivery" {
		t.Fatal("expected selected_delivery freight scenario")
	}

	cartSummary, err := svc.CartDetailSummary(context.Background())
	if err != nil {
		t.Fatalf("load cart detail summary: %v", err)
	}
	if cartSummary.ItemCount == 0 || len(cartSummary.CartIDs) == 0 {
		t.Fatal("expected normalized cart detail summary")
	}

	submitSummary, err := svc.OrderSubmitSummary(context.Background())
	if err != nil {
		t.Fatalf("load order submit summary: %v", err)
	}
	if submitSummary.BillID == "" || submitSummary.AddressID == "" || len(submitSummary.CartIDs) == 0 {
		t.Fatal("expected normalized order submit summary")
	}
	if !submitSummary.RequiresPayment {
		t.Fatal("expected submit summary to indicate payment is required")
	}

	cartListSummary, err := svc.CartListSummary(context.Background())
	if err != nil {
		t.Fatalf("load cart list summary: %v", err)
	}
	if cartListSummary.ItemCount == 0 || len(cartListSummary.Items) == 0 {
		t.Fatal("expected normalized cart list summary")
	}

	freightSummary, err := svc.FreightSummary(context.Background())
	if err != nil {
		t.Fatalf("load freight summary: %v", err)
	}
	if len(freightSummary.Scenarios) != 2 {
		t.Fatalf("expected two freight scenarios in summary, got %d", len(freightSummary.Scenarios))
	}

	defaultDeliverySummary, err := svc.DefaultDeliverySummary(context.Background())
	if err != nil {
		t.Fatalf("load default delivery summary: %v", err)
	}
	if !defaultDeliverySummary.Found || defaultDeliverySummary.Address == nil {
		t.Fatal("expected default delivery summary fallback")
	}

	deliveriesSummary, err := svc.DeliveriesSummary(context.Background())
	if err != nil {
		t.Fatalf("load deliveries summary: %v", err)
	}
	if deliveriesSummary.Count == 0 || len(deliveriesSummary.Items) == 0 {
		t.Fatal("expected deliveries summary items")
	}

	checkoutSummary, err := svc.CheckoutSummary(context.Background())
	if err != nil {
		t.Fatalf("load checkout summary: %v", err)
	}
	if checkoutSummary.CartDetail.ItemCount == 0 || !checkoutSummary.DefaultDelivery.Found || checkoutSummary.Submit.BillID == "" {
		t.Fatal("expected checkout summary to aggregate cart, address and submit state")
	}
}

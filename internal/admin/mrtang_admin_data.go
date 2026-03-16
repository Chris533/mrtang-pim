package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappimporter "mrtang-pim/internal/miniapp/importer"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

type mrtangAdminPageData struct {
	Procurement      pim.ProcurementWorkbenchSummary
	ProcurementError string
	Miniapp          mrtangAdminMiniappData
	MiniappError     string
	QuickActions     []mrtangAdminLink
	GeneratedAt      string
}

type mrtangAdminMiniappData struct {
	SourceMode                  string
	SourceURL                   string
	DatasetSource               string
	ContractCount               int
	HomepageSectionCount        int
	HomepageProductCount        int
	CategoryTopLevelCount       int
	CategoryNodeCount           int
	CategorySectionCount        int
	CategorySectionWithProducts int
	CategoryProductCount        int
	ProductTotal                int
	ProductRRDetailCount        int
	ProductSkeletonCount        int
	MultiUnitTotal              int
	FirstBatch                  []miniappmodel.ProductCoverage
	CartOperationCount          int
	OrderOperationCount         int
	FreightScenarioCount        int
	Backlog                     []mrtangAdminBacklogItem
}

type mrtangAdminBacklogItem struct {
	Area        string
	Status      string
	Summary     string
	Detail      string
	ActionLabel string
	ActionPath  string
}

type mrtangAdminLink struct {
	Eyebrow string
	Title   string
	Desc    string
	Href    string
}

func RenderMrtangAdminHTML(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
) string {
	pageData := buildMrtangAdminPageData(ctx, app, cfg, pimService, miniappService)
	return renderMrtangAdminHTML(pageData)
}

func buildMrtangAdminPageData(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
) mrtangAdminPageData {
	page := mrtangAdminPageData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		QuickActions: []mrtangAdminLink{
			{Eyebrow: "Miniapp API", Title: "分类树", Desc: "查看当前标准化分类树输出。", Href: "/api/miniapp/category-page/tree"},
			{Eyebrow: "Miniapp API", Title: "商品覆盖摘要", Desc: "查看 rr_detail / skeleton 覆盖和优先批次。", Href: "/api/miniapp/product-page/coverage-summary"},
			{Eyebrow: "Miniapp API", Title: "结算摘要", Desc: "一次性查看购物车、地址、运费与提交预览。", Href: "/api/miniapp/cart-order/checkout-summary"},
			{Eyebrow: "Contracts", Title: "分类契约", Desc: "查看分类页源站 API 到本地接口映射。", Href: "/api/miniapp/contracts/category-page"},
			{Eyebrow: "Contracts", Title: "商品契约", Desc: "查看商品详情链路和本地聚合接口。", Href: "/api/miniapp/contracts/product-page"},
			{Eyebrow: "Coverage", Title: "首页双单位优先批次", Desc: "直接查看下一批优先补 rr 详情的商品。", Href: "/api/miniapp/product-page/coverage?priority=homepage_dual_unit"},
		},
	}

	if pimService != nil {
		summary, err := pimService.ProcurementWorkbenchSummary(ctx, app, 12)
		if err != nil {
			page.ProcurementError = err.Error()
		} else {
			page.Procurement = summary
		}
	}

	if miniappService != nil {
		miniappData, err := buildMrtangAdminMiniappData(ctx, cfg, miniappService)
		if err != nil {
			page.MiniappError = err.Error()
		} else {
			page.Miniapp = miniappData
		}
	}

	return page
}

func buildMrtangAdminMiniappData(ctx context.Context, cfg config.Config, service *miniappservice.Service) (mrtangAdminMiniappData, error) {
	dataset, err := service.Dataset(ctx)
	if err != nil {
		return mrtangAdminMiniappData{}, err
	}

	coverageSummary := miniappimporter.NewHomepageImporter().ProductCoverageSummary(dataset)
	data := mrtangAdminMiniappData{
		SourceMode:            strings.TrimSpace(cfg.MiniApp.SourceMode),
		SourceURL:             strings.TrimSpace(cfg.MiniApp.SourceURL),
		DatasetSource:         strings.TrimSpace(dataset.Meta.Source),
		ContractCount:         len(dataset.Contracts),
		HomepageSectionCount:  len(dataset.Homepage.Sections),
		HomepageProductCount:  countHomepageProducts(dataset.Homepage.Sections),
		CategoryTopLevelCount: len(dataset.CategoryPage.Tree),
		CategoryNodeCount:     countCategoryNodes(dataset.CategoryPage.Tree),
		CategorySectionCount:  len(dataset.CategoryPage.Sections),
		CategoryProductCount:  countCategoryProducts(dataset.CategoryPage.Sections),
		ProductTotal:          len(dataset.ProductPage.Products),
		MultiUnitTotal:        coverageSummary.MultiUnitTotal,
		FirstBatch:            append([]miniappmodel.ProductCoverage(nil), coverageSummary.FirstBatch...),
		CartOperationCount:    countCartOperations(dataset.CartOrder.Cart),
		OrderOperationCount:   countOrderOperations(dataset.CartOrder.Order),
		FreightScenarioCount:  len(dataset.CartOrder.Order.FreightCosts),
	}

	for _, section := range dataset.CategoryPage.Sections {
		if len(section.Products) > 0 {
			data.CategorySectionWithProducts++
		}
	}

	for _, product := range dataset.ProductPage.Products {
		switch strings.ToLower(strings.TrimSpace(product.SourceType)) {
		case "rr_detail":
			data.ProductRRDetailCount++
		case "list_skeleton":
			data.ProductSkeletonCount++
		}
	}

	data.Backlog = buildMrtangAdminBacklog(data)
	return data, nil
}

func buildMrtangAdminBacklog(data mrtangAdminMiniappData) []mrtangAdminBacklogItem {
	items := make([]mrtangAdminBacklogItem, 0, 6)

	sourceStatus := "partial"
	sourceSummary := "http 模式和标准化 Dataset 边界已支持，当前运行模式为 " + blankFallback(data.SourceMode, "snapshot")
	if strings.EqualFold(data.SourceMode, "http") && strings.TrimSpace(data.SourceURL) != "" {
		sourceStatus = "done"
		sourceSummary = "当前已切到 http 模式，并配置了上游标准化 Dataset 地址"
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "目标 API 接入",
		Status:      sourceStatus,
		Summary:     sourceSummary,
		Detail:      "后续真实分类、商品、购物车下单链路应优先通过目标 API 整理成标准化 Dataset，再接入 http source。",
		ActionLabel: "查看总览",
		ActionPath:  "/api/miniapp/contracts/homepage",
	})

	treeStatus := "pending"
	treeSummary := "分类树尚未固化"
	if data.CategoryTopLevelCount > 0 {
		treeStatus = "done"
		treeSummary = fmt.Sprintf("分类树已接入，当前 %d 个顶级类目，%d 个总节点", data.CategoryTopLevelCount, data.CategoryNodeCount)
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "分类树",
		Status:      treeStatus,
		Summary:     treeSummary,
		Detail:      "分类树用于分类页导航和后续分类商品采集分桶，当前应继续保持与目标 API 的节点结构一致。",
		ActionLabel: "打开分类树",
		ActionPath:  "/api/miniapp/category-page/tree",
	})

	sectionStatus := "pending"
	sectionSummary := "分类商品 section 尚未整理"
	if data.CategorySectionCount > 0 && data.CategorySectionWithProducts == data.CategorySectionCount {
		sectionStatus = "done"
		sectionSummary = fmt.Sprintf("分类商品 section 已齐全，%d/%d 带商品", data.CategorySectionWithProducts, data.CategorySectionCount)
	} else if data.CategorySectionCount > 0 {
		sectionStatus = "partial"
		sectionSummary = fmt.Sprintf("分类商品 section 已建 %d 个，其中 %d 个带真实商品", data.CategorySectionCount, data.CategorySectionWithProducts)
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "分类商品列表",
		Status:      sectionStatus,
		Summary:     sectionSummary,
		Detail:      "当前骨架和真实样本是混合状态，后续应继续从目标 API 补足空 section 的商品列表和价格库存。",
		ActionLabel: "查看分类 sections",
		ActionPath:  "/api/miniapp/category-page/sections",
	})

	productStatus := "pending"
	productSummary := "商品详情尚未整理"
	if data.ProductTotal > 0 && data.ProductRRDetailCount == data.ProductTotal {
		productStatus = "done"
		productSummary = fmt.Sprintf("商品页已全部落为 rr_detail，共 %d 条", data.ProductTotal)
	} else if data.ProductTotal > 0 {
		productStatus = "partial"
		productSummary = fmt.Sprintf("商品页共 %d 条，其中 rr_detail %d，skeleton %d", data.ProductTotal, data.ProductRRDetailCount, data.ProductSkeletonCount)
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "商品详情 / 价格库存",
		Status:      productStatus,
		Summary:     productSummary,
		Detail:      "首页和分类页已通过 productId 对齐到商品页，但还需要继续把 skeleton 升级成真实 rr_detail 或目标 API 数据。",
		ActionLabel: "查看覆盖摘要",
		ActionPath:  "/api/miniapp/product-page/coverage-summary",
	})

	multiUnitPending := 0
	for _, item := range data.FirstBatch {
		if item.Priority == "homepage_dual_unit" || item.Priority == "category_dual_unit" || item.Priority == "visible_single_unit" {
			multiUnitPending++
		}
	}
	priceStatus := "pending"
	priceSummary := "多单位价格 SKU 尚未覆盖"
	if data.MultiUnitTotal > 0 && multiUnitPending == 0 {
		priceStatus = "done"
		priceSummary = fmt.Sprintf("可见商品中的多单位价格 SKU 已全部转为 rr_detail，当前多单位商品 %d", data.MultiUnitTotal)
	} else if data.MultiUnitTotal > 0 {
		priceStatus = "partial"
		priceSummary = fmt.Sprintf("当前可见多单位商品 %d，优先批次仍有 %d 条待补", data.MultiUnitTotal, len(data.FirstBatch))
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "多单位价格 SKU",
		Status:      priceStatus,
		Summary:     priceSummary,
		Detail:      "优先补首页和分类页里已经露出的双单位商品，让单位切换、价格、库存和上下文都落成真实详情链路。",
		ActionLabel: "查看优先批次",
		ActionPath:  "/api/miniapp/product-page/coverage?priority=homepage_dual_unit",
	})

	checkoutStatus := "pending"
	checkoutSummary := "购物车与下单摘要尚未完整"
	if data.CartOperationCount >= 5 && data.OrderOperationCount >= 5 && data.FreightScenarioCount >= 2 {
		checkoutStatus = "done"
		checkoutSummary = fmt.Sprintf("购物车与下单链路已齐，cart %d 项，order %d 项，运费场景 %d 个", data.CartOperationCount, data.OrderOperationCount, data.FreightScenarioCount)
	} else if data.CartOperationCount > 0 || data.OrderOperationCount > 0 {
		checkoutStatus = "partial"
		checkoutSummary = fmt.Sprintf("购物车与下单链路已部分接入，cart %d 项，order %d 项", data.CartOperationCount, data.OrderOperationCount)
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "购物车 / 下单",
		Status:      checkoutStatus,
		Summary:     checkoutSummary,
		Detail:      "当前已能提供 checkout-summary，但后续仍应与目标 API 保持字段和场景对齐，尤其是地址、运费与提交结果。",
		ActionLabel: "打开 checkout-summary",
		ActionPath:  "/api/miniapp/cart-order/checkout-summary",
	})

	return items
}

func countHomepageProducts(sections []miniappmodel.HomepageSection) int {
	total := 0
	for _, section := range sections {
		total += len(section.Products)
	}
	return total
}

func countCategoryProducts(sections []miniappmodel.CategorySection) int {
	total := 0
	for _, section := range sections {
		total += len(section.Products)
	}
	return total
}

func countCategoryNodes(nodes []miniappmodel.CategoryNode) int {
	total := 0
	for _, node := range nodes {
		total++
		total += countCategoryNodes(node.Children)
	}
	return total
}

func countCartOperations(cart miniappmodel.CartAggregate) int {
	total := 0
	if strings.TrimSpace(cart.Add.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(cart.ChangeNum.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(cart.List.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(cart.Detail.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(cart.Settle.ContractID) != "" {
		total++
	}
	return total
}

func countOrderOperations(order miniappmodel.OrderAggregate) int {
	total := 0
	if strings.TrimSpace(order.DefaultDelivery.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(order.Deliveries.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(order.AnalyseAddress.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(order.AddDelivery.ContractID) != "" {
		total++
	}
	if strings.TrimSpace(order.Submit.ContractID) != "" {
		total++
	}
	return total
}

func blankFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

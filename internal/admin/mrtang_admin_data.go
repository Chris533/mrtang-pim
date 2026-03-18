package admin

import (
	"context"
	"encoding/json"
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
	Procurement          pim.ProcurementWorkbenchSummary
	ProcurementError     string
	Miniapp              mrtangAdminMiniappData
	MiniappError         string
	SourceCapture        mrtangAdminSourceCaptureData
	SourceError          string
	FlashMessage         string
	FlashError           string
	CanAccessSource      bool
	CanAccessProcurement bool
	QuickActions         []mrtangAdminLink
	RecentActions        []mrtangAdminRecentAction
	GeneratedAt          string
}

type mrtangAdminMiniappData struct {
	SourceMode                  string
	ConfigSourceMode            string
	SourceURL                   string
	DatasetSource               string
	RawAuthStatus               miniappmodel.RawAuthStatus
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

type mrtangAdminSourceCaptureData struct {
	CategoryCount       int
	ProductCount        int
	AssetCount          int
	ApprovedCount       int
	ImportedCount       int
	PromotedCount       int
	LinkedCount         int
	SyncedCount         int
	SyncErrorCount      int
	ProcessedAssetCount int
	FailedAssetCount    int
	RecentActions       []pim.SourceActionLog
}

type mrtangAdminRecentAction struct {
	Domain  string
	Label   string
	Target  string
	Status  string
	Message string
	Actor   string
	Note    string
	Created string
}

type DashboardMiniappAPIData struct {
	Miniapp      mrtangAdminMiniappData `json:"miniapp"`
	MiniappError string                 `json:"miniappError"`
}

func RenderMrtangAdminHTML(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
	canAccessSource bool,
	canAccessProcurement bool,
	flashMessage string,
	flashError string,
) string {
	pageData := buildMrtangAdminPageData(ctx, app, cfg, pimService, miniappService)
	pageData.CanAccessSource = canAccessSource
	pageData.CanAccessProcurement = canAccessProcurement
	pageData.FlashMessage = strings.TrimSpace(flashMessage)
	pageData.FlashError = strings.TrimSpace(flashError)
	return renderMrtangAdminHTML(pageData)
}

func RenderAuditHTML(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
	filter AuditFilter,
	flashMessage string,
	flashError string,
) string {
	pageData := buildMrtangAdminPageData(ctx, app, cfg, pimService, miniappService)
	return RenderAuditPageHTML(
		filterAuditActions(pageData.RecentActions, filter),
		strings.TrimSpace(flashMessage),
		strings.TrimSpace(flashError),
	)
}

func filterAuditActions(items []mrtangAdminRecentAction, filter AuditFilter) AuditPageData {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	filtered := make([]mrtangAdminRecentAction, 0, len(items))
	successCount := 0
	failedCount := 0
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, item := range items {
		if filter.Domain != "" && !strings.EqualFold(strings.TrimSpace(item.Domain), filter.Domain) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(strings.TrimSpace(item.Status), filter.Status) {
			continue
		}
		if query != "" {
			search := strings.ToLower(strings.Join([]string{item.Domain, item.Label, item.Target, item.Actor, item.Message, item.Note}, " "))
			if !strings.Contains(search, query) {
				continue
			}
		}
		if strings.EqualFold(item.Status, "success") {
			successCount++
		}
		if strings.EqualFold(item.Status, "failed") {
			failedCount++
		}
		filtered = append(filtered, item)
	}
	pages := 1
	if len(filtered) > 0 {
		pages = len(filtered) / filter.PageSize
		if len(filtered)%filter.PageSize != 0 {
			pages++
		}
		if pages <= 0 {
			pages = 1
		}
	}
	start := (filter.Page - 1) * filter.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + filter.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return AuditPageData{
		Items:        filtered[start:end],
		Filter:       filter,
		Total:        len(filtered),
		Page:         filter.Page,
		Pages:        pages,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}
}

func buildMrtangAdminPageData(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
) mrtangAdminPageData {
	page := buildMrtangAdminBaseData(ctx, app, cfg, pimService)
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

func buildMrtangAdminBaseData(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
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

	sourceCapture, err := buildMrtangAdminSourceCaptureData(ctx, app, pimService)
	if err != nil {
		page.SourceError = err.Error()
	} else {
		page.SourceCapture = sourceCapture
	}
	page.RecentActions = buildMrtangAdminRecentActions(app, page.SourceCapture.RecentActions, page.Procurement.RecentActions)

	return page
}

func buildMrtangAdminSourceCaptureData(ctx context.Context, app core.App, pimService *pim.Service) (mrtangAdminSourceCaptureData, error) {
	categories, err := app.FindAllRecords(pim.CollectionSourceCategories)
	if err != nil {
		return mrtangAdminSourceCaptureData{}, err
	}
	products, err := app.FindAllRecords(pim.CollectionSourceProducts)
	if err != nil {
		return mrtangAdminSourceCaptureData{}, err
	}
	assets, err := app.FindAllRecords(pim.CollectionSourceAssets)
	if err != nil {
		return mrtangAdminSourceCaptureData{}, err
	}

	data := mrtangAdminSourceCaptureData{
		CategoryCount: len(categories),
		ProductCount:  len(products),
		AssetCount:    len(assets),
	}
	for _, product := range products {
		switch strings.ToLower(strings.TrimSpace(product.GetString("review_status"))) {
		case "approved":
			data.ApprovedCount++
		case "promoted":
			data.ApprovedCount++
			data.PromotedCount++
		case "imported":
			data.ImportedCount++
		}
	}
	for _, asset := range assets {
		switch strings.ToLower(strings.TrimSpace(asset.GetString("image_processing_status"))) {
		case pim.ImageStatusProcessed:
			data.ProcessedAssetCount++
		case pim.ImageStatusFailed:
			data.FailedAssetCount++
		}
	}
	if pimService != nil {
		summary, err := pimService.SourceReviewWorkbench(ctx, app, 1, 1, pim.SourceReviewFilter{PageSize: 1})
		if err == nil {
			data.LinkedCount = summary.LinkedCount
			data.SyncedCount = summary.SyncedCount
			data.SyncErrorCount = summary.SyncErrorCount
			data.RecentActions = append([]pim.SourceActionLog(nil), summary.RecentActions...)
		}
	}
	return data, nil
}

func buildMrtangAdminMiniappData(ctx context.Context, cfg config.Config, service *miniappservice.Service) (mrtangAdminMiniappData, error) {
	loadCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	dataset, err := service.Dataset(loadCtx)
	if err != nil {
		return mrtangAdminMiniappData{}, err
	}

	coverageSummary := miniappimporter.NewHomepageImporter().ProductCoverageSummary(dataset)
	actualSourceMode := strings.TrimSpace(dataset.Meta.Source)
	if actualSourceMode == "" {
		actualSourceMode = strings.TrimSpace(cfg.MiniApp.SourceMode)
	}
	data := mrtangAdminMiniappData{
		SourceMode:            actualSourceMode,
		ConfigSourceMode:      strings.TrimSpace(cfg.MiniApp.SourceMode),
		SourceURL:             strings.TrimSpace(cfg.MiniApp.SourceURL),
		DatasetSource:         strings.TrimSpace(dataset.Meta.Source),
		RawAuthStatus:         service.RawAuthStatus(),
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

func buildDashboardMiniappAPIData(ctx context.Context, cfg config.Config, service *miniappservice.Service) DashboardMiniappAPIData {
	data := DashboardMiniappAPIData{}
	data.Miniapp.ConfigSourceMode = strings.TrimSpace(cfg.MiniApp.SourceMode)
	data.Miniapp.SourceMode = strings.TrimSpace(cfg.MiniApp.SourceMode)
	data.Miniapp.SourceURL = strings.TrimSpace(cfg.MiniApp.SourceURL)
	if service == nil {
		return data
	}
	data.Miniapp.RawAuthStatus = service.RawAuthStatus()
	miniappData, err := buildMrtangAdminMiniappData(ctx, cfg, service)
	if err != nil {
		data.MiniappError = err.Error()
		return data
	}
	data.Miniapp = miniappData
	return data
}

func buildMrtangAdminRecentActions(app core.App, source []pim.SourceActionLog, procurement []pim.ProcurementActionLog) []mrtangAdminRecentAction {
	items := make([]mrtangAdminRecentAction, 0, len(source)+len(procurement))
	for _, item := range source {
		actor := strings.TrimSpace(item.ActorName)
		if actor == "" {
			actor = strings.TrimSpace(item.ActorEmail)
		}
		if actor == "" {
			actor = "系统"
		}
		items = append(items, mrtangAdminRecentAction{
			Domain:  "源数据",
			Label:   sourceActionTypeLabel(item.ActionType),
			Target:  strings.TrimSpace(item.TargetLabel),
			Status:  strings.TrimSpace(item.Status),
			Message: strings.TrimSpace(item.Message),
			Actor:   actor,
			Note:    strings.TrimSpace(item.Note),
			Created: strings.TrimSpace(item.Created),
		})
	}
	for _, item := range procurement {
		actor := strings.TrimSpace(item.ActorName)
		if actor == "" {
			actor = strings.TrimSpace(item.ActorEmail)
		}
		if actor == "" {
			actor = "系统"
		}
		items = append(items, mrtangAdminRecentAction{
			Domain:  "采购",
			Label:   procurementActionLabel(item.ActionType),
			Target:  strings.TrimSpace(item.ExternalRef),
			Status:  strings.TrimSpace(item.Status),
			Message: strings.TrimSpace(item.Message),
			Actor:   actor,
			Note:    strings.TrimSpace(item.Note),
			Created: strings.TrimSpace(item.Created),
		})
	}
	if assetJobs, err := app.FindRecordsByFilter(pim.CollectionSourceAssetJobs, "", "-created", 6, 0, nil); err == nil {
		for _, record := range assetJobs {
			jobType := strings.TrimSpace(record.GetString("job_type"))
			mode := strings.TrimSpace(record.GetString("mode"))
			label := assetJobActionLabel(jobType, mode)
			message := strings.TrimSpace(record.GetString("error"))
			if message == "" {
				message = fmt.Sprintf("%s：成功 %d / 总数 %d，失败 %d", assetJobModeLabel(mode), record.GetInt("processed"), record.GetInt("total"), record.GetInt("failed_count"))
			}
			target := strings.TrimSpace(record.GetString("current_item"))
			if target == "" {
				target = assetJobTargetLabel(record)
			}
			items = append(items, mrtangAdminRecentAction{
				Domain:  "图片任务",
				Label:   label,
				Target:  target,
				Status:  strings.TrimSpace(record.GetString("status")),
				Message: message,
				Actor:   "系统",
				Note:    "",
				Created: strings.TrimSpace(record.GetString("created")),
			})
		}
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Created > items[i].Created {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if len(items) > 10 {
		items = items[:10]
	}
	return items
}

func procurementActionLabel(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "create_order":
		return "创建采购单"
	case "export_order":
		return "导出采购单"
	case "update_status":
		return "更新采购状态"
	default:
		return action
	}
}

func assetJobActionLabel(jobType string, mode string) string {
	switch strings.ToLower(strings.TrimSpace(jobType)) {
	case "download_original":
		if strings.Contains(strings.ToLower(strings.TrimSpace(mode)), "selected") {
			return "选中图片原图下载任务"
		}
		return "原图下载任务"
	case "process_asset":
		mode = strings.ToLower(strings.TrimSpace(mode))
		if strings.Contains(mode, "selected") && strings.Contains(mode, "failed") {
			return "选中失败图片重处理任务"
		}
		if strings.Contains(mode, "selected") {
			return "选中图片处理任务"
		}
		if strings.Contains(mode, "failed") {
			return "失败图片重处理任务"
		}
		return "图片处理任务"
	default:
		return "图片任务"
	}
}

func assetJobModeLabel(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "selected":
		return "选中项"
	case "selected_failed", "failed_only":
		return "选中失败项"
	case "failed":
		return "失败项"
	case "pending":
		return "待处理"
	default:
		return "全量"
	}
}

func assetJobTargetLabel(record *core.Record) string {
	var ids []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(record.GetString("asset_ids_json"))), &ids); err == nil {
		count := 0
		for _, item := range ids {
			if strings.TrimSpace(item) != "" {
				count++
			}
		}
		if count > 0 {
			return fmt.Sprintf("%d 张图片", count)
		}
	}
	return "图片批次"
}

func buildMrtangAdminBacklog(data mrtangAdminMiniappData) []mrtangAdminBacklogItem {
	items := make([]mrtangAdminBacklogItem, 0, 6)

	sourceStatus := "partial"
	sourceSummary := "raw 真实源站链路已接入，当前运行模式为 " + blankFallback(data.SourceMode, "snapshot")
	if strings.EqualFold(data.SourceMode, "raw") && strings.TrimSpace(data.SourceURL) != "" {
		sourceStatus = "done"
		sourceSummary = "当前已切到 raw 模式，并配置了目标站源地址"
	}
	items = append(items, mrtangAdminBacklogItem{
		Area:        "目标 API 接入",
		Status:      sourceStatus,
		Summary:     sourceSummary,
		Detail:      "后续真实分类、商品、购物车和下单链路应直接通过 raw 模式接入目标 API，并在本项目内部标准化。",
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

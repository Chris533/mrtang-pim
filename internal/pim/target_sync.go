package pim

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	miniappmodel "mrtang-pim/internal/miniapp/model"
)

const (
	CollectionTargetSyncJobs = "target_sync_jobs"
	CollectionTargetSyncRuns = "target_sync_runs"

	TargetSyncEntityCategoryTree    = "category_tree"
	TargetSyncEntityCategorySources = "category_sources"
	TargetSyncEntityCategoryRebuild = "category_rebuild"
	TargetSyncEntityProducts        = "products"
	TargetSyncEntityAssets          = "assets"

	TargetSyncScopeAll      = "all"
	TargetSyncScopeTopLevel = "top_level"

	TargetSyncStatusPending = "pending"
	TargetSyncStatusRunning = "running"
	TargetSyncStatusSuccess = "success"
	TargetSyncStatusPartial = "partial"
	TargetSyncStatusFailed  = "failed"
)

type targetSyncDatasetLoader func(context.Context, string, string) (*miniappmodel.Dataset, error)
type targetSyncSectionProductLoader func(context.Context, []miniappmodel.CategorySection, string) (*miniappmodel.Dataset, error)

type TargetSyncActor struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type TargetSyncJob struct {
	ID            string `json:"id"`
	JobKey        string `json:"jobKey"`
	Name          string `json:"name"`
	EntityType    string `json:"entityType"`
	ScopeType     string `json:"scopeType"`
	ScopeKey      string `json:"scopeKey"`
	ScopeLabel    string `json:"scopeLabel"`
	Status        string `json:"status"`
	SourceMode    string `json:"sourceMode"`
	LastRunAt     string `json:"lastRunAt"`
	LastSuccessAt string `json:"lastSuccessAt"`
	LastError     string `json:"lastError"`
}

type TargetSyncRun struct {
	ID               string                  `json:"id"`
	JobKey           string                  `json:"jobKey"`
	JobName          string                  `json:"jobName"`
	EntityType       string                  `json:"entityType"`
	ScopeType        string                  `json:"scopeType"`
	ScopeKey         string                  `json:"scopeKey"`
	ScopeLabel       string                  `json:"scopeLabel"`
	Status           string                  `json:"status"`
	SourceMode       string                  `json:"sourceMode"`
	StartedAt        string                  `json:"startedAt"`
	FinishedAt       string                  `json:"finishedAt"`
	TriggeredByEmail string                  `json:"triggeredByEmail"`
	TriggeredByName  string                  `json:"triggeredByName"`
	CreatedCount     int                     `json:"createdCount"`
	UpdatedCount     int                     `json:"updatedCount"`
	UnchangedCount   int                     `json:"unchangedCount"`
	MissingCount     int                     `json:"missingCount"`
	ScopedNodeCount  int                     `json:"scopedNodeCount"`
	ProgressTotal    int                     `json:"progressTotal"`
	ProgressDone     int                     `json:"progressDone"`
	CurrentStage     string                  `json:"currentStage"`
	CurrentItem      string                  `json:"currentItem"`
	LastProgressAt   string                  `json:"lastProgressAt"`
	ErrorMessage     string                  `json:"errorMessage"`
	Details          []TargetSyncChangeItem  `json:"details"`
	Logs             []TargetSyncProgressLog `json:"logs"`
}

type TargetSyncScopeOption struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	NodeCount    int    `json:"nodeCount"`
	ProductCount int    `json:"productCount"`
	AssetCount   int    `json:"assetCount"`
}

type TargetCategoryDiffItem struct {
	SourceKey    string `json:"sourceKey"`
	Label        string `json:"label"`
	CategoryPath string `json:"categoryPath"`
	DiffType     string `json:"diffType"`
	ScopeLabel   string `json:"scopeLabel"`
}

type TargetSyncSummary struct {
	SourceMode                 string                     `json:"sourceMode"`
	RawAuthStatus              miniappmodel.RawAuthStatus `json:"rawAuthStatus"`
	JobCount                   int                        `json:"jobCount"`
	RunCount                   int                        `json:"runCount"`
	CategoryCount              int                        `json:"categoryCount"`
	SourceCategorySectionCount int                        `json:"sourceCategorySectionCount"`
	SourceProductCount         int                        `json:"sourceProductCount"`
	SourceAssetCount           int                        `json:"sourceAssetCount"`
	ExpectedNodeCount          int                        `json:"expectedNodeCount"`
	ExpectedProductCount       int                        `json:"expectedProductCount"`
	ExpectedAssetCount         int                        `json:"expectedAssetCount"`
	ExpectedMultiUnitCount     int                        `json:"expectedMultiUnitCount"`
	TopLevelCount              int                        `json:"topLevelCount"`
	SourceImportedCount        int                        `json:"sourceImportedCount"`
	SourceApprovedCount        int                        `json:"sourceApprovedCount"`
	SourceAssetPendingCount    int                        `json:"sourceAssetPendingCount"`
	SourceAssetFailedCount     int                        `json:"sourceAssetFailedCount"`
	DiffNewCount               int                        `json:"diffNewCount"`
	DiffChangedCount           int                        `json:"diffChangedCount"`
	DiffMissingCount           int                        `json:"diffMissingCount"`
	ProductDiffNewCount        int                        `json:"productDiffNewCount"`
	ProductDiffChangedCount    int                        `json:"productDiffChangedCount"`
	AssetDiffNewCount          int                        `json:"assetDiffNewCount"`
	AssetDiffChangedCount      int                        `json:"assetDiffChangedCount"`
	Jobs                       []TargetSyncJob            `json:"jobs"`
	Runs                       []TargetSyncRun            `json:"runs"`
	ScopeOptions               []TargetSyncScopeOption    `json:"scopeOptions"`
	CategoryDiffs              []TargetCategoryDiffItem   `json:"categoryDiffs"`
	CheckoutSources            []TargetCheckoutSource     `json:"checkoutSources"`
	RecentMiniappWrites        []TargetMiniappWrite       `json:"recentMiniappWrites"`
}

type TargetSyncBaseSummary struct {
	SourceMode                 string                     `json:"sourceMode"`
	RawAuthStatus              miniappmodel.RawAuthStatus `json:"rawAuthStatus"`
	JobCount                   int                        `json:"jobCount"`
	RunCount                   int                        `json:"runCount"`
	CategoryCount              int                        `json:"categoryCount"`
	SourceCategorySectionCount int                        `json:"sourceCategorySectionCount"`
	SourceProductCount         int                        `json:"sourceProductCount"`
	SourceAssetCount           int                        `json:"sourceAssetCount"`
	SourceImportedCount        int                        `json:"sourceImportedCount"`
	SourceApprovedCount        int                        `json:"sourceApprovedCount"`
	SourceAssetPendingCount    int                        `json:"sourceAssetPendingCount"`
	SourceAssetFailedCount     int                        `json:"sourceAssetFailedCount"`
	TopLevelCount              int                        `json:"topLevelCount"`
	ScopeOptions               []TargetSyncScopeOption    `json:"scopeOptions"`
	Jobs                       []TargetSyncJob            `json:"jobs"`
	Runs                       []TargetSyncRun            `json:"runs"`
	RecentMiniappWrites        []TargetMiniappWrite       `json:"recentMiniappWrites"`
}

type TargetSyncLiveSummary struct {
	SourceMode              string                   `json:"sourceMode"`
	ExpectedNodeCount       int                      `json:"expectedNodeCount"`
	ExpectedProductCount    int                      `json:"expectedProductCount"`
	ExpectedAssetCount      int                      `json:"expectedAssetCount"`
	ExpectedMultiUnitCount  int                      `json:"expectedMultiUnitCount"`
	TopLevelCount           int                      `json:"topLevelCount"`
	DiffNewCount            int                      `json:"diffNewCount"`
	DiffChangedCount        int                      `json:"diffChangedCount"`
	DiffMissingCount        int                      `json:"diffMissingCount"`
	ProductDiffNewCount     int                      `json:"productDiffNewCount"`
	ProductDiffChangedCount int                      `json:"productDiffChangedCount"`
	AssetDiffNewCount       int                      `json:"assetDiffNewCount"`
	AssetDiffChangedCount   int                      `json:"assetDiffChangedCount"`
	ScopeOptions            []TargetSyncScopeOption  `json:"scopeOptions"`
	CategoryDiffs           []TargetCategoryDiffItem `json:"categoryDiffs"`
	CheckoutSources         []TargetCheckoutSource   `json:"checkoutSources"`
	UsedStoredData          bool                     `json:"usedStoredData"`
}

type TargetSyncCheckoutLiveSummary struct {
	CheckoutSources []TargetCheckoutSource `json:"checkoutSources"`
}

type TargetCheckoutSource struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	ContractID string `json:"contractId"`
	Note       string `json:"note"`
}

type TargetMiniappWrite struct {
	OperationID    string `json:"operationId"`
	OperationLabel string `json:"operationLabel"`
	ContractID     string `json:"contractId"`
	Status         string `json:"status"`
	Message        string `json:"message"`
	CreatedAt      string `json:"createdAt"`
}

type targetCategorySyncResult struct {
	entityType      string
	jobKey          string
	jobName         string
	scopeType       string
	scopeKey        string
	scopeLabel      string
	status          string
	sourceMode      string
	scopedNodeCount int
	createdCount    int
	updatedCount    int
	unchangedCount  int
	missingCount    int
	errorMessage    string
	details         []TargetSyncChangeItem
}

type TargetSyncChangeItem struct {
	ChangeType string `json:"changeType"`
	TargetType string `json:"targetType"`
	TargetKey  string `json:"targetKey"`
	Label      string `json:"label"`
	Path       string `json:"path"`
	Note       string `json:"note"`
}

type TargetSyncProgressLog struct {
	Time    string `json:"time"`
	Stage   string `json:"stage"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type targetSyncProgressTracker struct {
	app    core.App
	record *core.Record
	logs   []TargetSyncProgressLog
}

func (s *Service) TargetSyncSummary(_ context.Context, app core.App, dataset miniappmodel.Dataset, rawAuthStatus miniappmodel.RawAuthStatus) (TargetSyncSummary, error) {
	base, err := s.TargetSyncBaseSummary(app, dataset.Meta.Source, rawAuthStatus)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	live, err := s.TargetSyncLiveSummary(app, dataset)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	return TargetSyncSummary{
		SourceMode:                 base.SourceMode,
		RawAuthStatus:              base.RawAuthStatus,
		JobCount:                   base.JobCount,
		RunCount:                   base.RunCount,
		CategoryCount:              base.CategoryCount,
		SourceCategorySectionCount: base.SourceCategorySectionCount,
		SourceProductCount:         base.SourceProductCount,
		SourceAssetCount:           base.SourceAssetCount,
		ExpectedNodeCount:          live.ExpectedNodeCount,
		ExpectedProductCount:       live.ExpectedProductCount,
		ExpectedAssetCount:         live.ExpectedAssetCount,
		ExpectedMultiUnitCount:     live.ExpectedMultiUnitCount,
		TopLevelCount:              live.TopLevelCount,
		SourceImportedCount:        base.SourceImportedCount,
		SourceApprovedCount:        base.SourceApprovedCount,
		SourceAssetPendingCount:    base.SourceAssetPendingCount,
		SourceAssetFailedCount:     base.SourceAssetFailedCount,
		DiffNewCount:               live.DiffNewCount,
		DiffChangedCount:           live.DiffChangedCount,
		DiffMissingCount:           live.DiffMissingCount,
		ProductDiffNewCount:        live.ProductDiffNewCount,
		ProductDiffChangedCount:    live.ProductDiffChangedCount,
		AssetDiffNewCount:          live.AssetDiffNewCount,
		AssetDiffChangedCount:      live.AssetDiffChangedCount,
		Jobs:                       base.Jobs,
		Runs:                       base.Runs,
		ScopeOptions:               live.ScopeOptions,
		CategoryDiffs:              live.CategoryDiffs,
		CheckoutSources:            live.CheckoutSources,
		RecentMiniappWrites:        base.RecentMiniappWrites,
	}, nil
}

func (s *Service) TargetSyncBaseSummary(app core.App, sourceMode string, rawAuthStatus miniappmodel.RawAuthStatus) (TargetSyncBaseSummary, error) {
	if err := s.reconcileStaleTargetSyncRuns(app); err != nil {
		return TargetSyncBaseSummary{}, err
	}
	sourceCategories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	sourceCategorySections, err := app.FindAllRecords(CollectionSourceCategorySections)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	sourceProducts, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	sourceAssets, err := app.FindAllRecords(CollectionSourceAssets)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	jobs, err := s.listTargetSyncJobs(app, 20)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	runs, err := s.listTargetSyncRuns(app, 12)
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}
	storedLive, err := s.TargetSyncStoredLiveSummary(app, strings.TrimSpace(sourceMode))
	if err != nil {
		return TargetSyncBaseSummary{}, err
	}

	importedCount := 0
	approvedCount := 0
	for _, record := range sourceProducts {
		switch strings.ToLower(strings.TrimSpace(record.GetString("review_status"))) {
		case "imported":
			importedCount++
		case "approved":
			approvedCount++
		}
	}
	assetPendingCount := 0
	assetFailedCount := 0
	for _, record := range sourceAssets {
		switch strings.ToLower(strings.TrimSpace(record.GetString("image_processing_status"))) {
		case ImageStatusPending:
			assetPendingCount++
		case ImageStatusFailed:
			assetFailedCount++
		}
	}

	return TargetSyncBaseSummary{
		SourceMode:                 strings.TrimSpace(sourceMode),
		RawAuthStatus:              rawAuthStatus,
		JobCount:                   len(jobs),
		RunCount:                   len(runs),
		CategoryCount:              len(sourceCategories),
		SourceCategorySectionCount: len(sourceCategorySections),
		SourceProductCount:         len(sourceProducts),
		SourceAssetCount:           len(sourceAssets),
		SourceImportedCount:        importedCount,
		SourceApprovedCount:        approvedCount,
		SourceAssetPendingCount:    assetPendingCount,
		SourceAssetFailedCount:     assetFailedCount,
		TopLevelCount:              storedLive.TopLevelCount,
		ScopeOptions:               storedLive.ScopeOptions,
		Jobs:                       jobs,
		Runs:                       runs,
		RecentMiniappWrites:        targetMiniappWrites(app, 8),
	}, nil
}

func (s *Service) TargetSyncLiveSummary(app core.App, dataset miniappmodel.Dataset) (TargetSyncLiveSummary, error) {
	expectedNodes := flattenCategoryNodes(dataset.CategoryPage.Tree)
	sourceCategories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}
	sourceProducts, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}
	sourceAssets, err := app.FindAllRecords(CollectionSourceAssets)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}

	diffItems, diffNew, diffChanged, diffMissing := targetCategoryDiff(dataset.CategoryPage.Tree, sourceCategories)
	productDiffNew, productDiffChanged, assetDiffNew, assetDiffChanged := targetProductAssetDiffs(dataset, sourceProducts, sourceAssets)
	scopeOptions := make([]TargetSyncScopeOption, 0, len(dataset.CategoryPage.Tree)+1)
	allProducts := filteredTargetProducts(dataset, "")
	allAssets := filteredTargetAssets(dataset, "")
	scopeOptions = append(scopeOptions, TargetSyncScopeOption{
		Key:          "",
		Label:        "当前源站结果",
		NodeCount:    len(expectedNodes),
		ProductCount: len(allProducts),
		AssetCount:   len(allAssets),
	})
	for _, node := range dataset.CategoryPage.Tree {
		topProducts := filteredTargetProducts(dataset, node.Key)
		topAssets := filteredTargetAssets(dataset, node.Key)
		scopeOptions = append(scopeOptions, TargetSyncScopeOption{
			Key:          node.Key,
			Label:        node.Label,
			NodeCount:    len(flattenCategoryNodes([]miniappmodel.CategoryNode{node})),
			ProductCount: len(topProducts),
			AssetCount:   len(topAssets),
		})
	}

	return TargetSyncLiveSummary{
		SourceMode:              dataset.Meta.Source,
		ExpectedNodeCount:       len(expectedNodes),
		ExpectedProductCount:    len(allProducts),
		ExpectedAssetCount:      len(allAssets),
		ExpectedMultiUnitCount:  countTargetMultiUnitProducts(allProducts),
		TopLevelCount:           len(dataset.CategoryPage.Tree),
		DiffNewCount:            diffNew,
		DiffChangedCount:        diffChanged,
		DiffMissingCount:        diffMissing,
		ProductDiffNewCount:     productDiffNew,
		ProductDiffChangedCount: productDiffChanged,
		AssetDiffNewCount:       assetDiffNew,
		AssetDiffChangedCount:   assetDiffChanged,
		ScopeOptions:            scopeOptions,
		CategoryDiffs:           diffItems,
		CheckoutSources:         targetCheckoutSources(dataset),
		UsedStoredData:          false,
	}, nil
}

func (s *Service) TargetSyncStoredLiveSummary(app core.App, sourceMode string) (TargetSyncLiveSummary, error) {
	sourceCategories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}
	sourceProducts, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}
	sourceAssets, err := app.FindAllRecords(CollectionSourceAssets)
	if err != nil {
		return TargetSyncLiveSummary{}, err
	}

	parentByKey := make(map[string]string, len(sourceCategories))
	labelByKey := make(map[string]string, len(sourceCategories))
	topLevelKeys := make([]string, 0)
	topLevelLabels := make(map[string]string)
	nodeCountByTop := make(map[string]int)
	for _, record := range sourceCategories {
		key := strings.TrimSpace(record.GetString("source_key"))
		if key == "" {
			continue
		}
		parentByKey[key] = strings.TrimSpace(record.GetString("parent_key"))
		labelByKey[key] = strings.TrimSpace(record.GetString("label"))
	}

	for _, record := range sourceCategories {
		key := strings.TrimSpace(record.GetString("source_key"))
		if key == "" {
			continue
		}
		depth := record.GetInt("depth")
		parentKey := parentByKey[key]
		if depth == 1 || parentKey == "" {
			topLevelKeys = append(topLevelKeys, key)
			topLevelLabels[key] = labelByKey[key]
		}
	}

	findTopLevel := func(key string) string {
		current := strings.TrimSpace(key)
		visited := map[string]struct{}{}
		for current != "" {
			if _, ok := visited[current]; ok {
				break
			}
			visited[current] = struct{}{}
			parent := strings.TrimSpace(parentByKey[current])
			if parent == "" {
				return current
			}
			current = parent
		}
		return strings.TrimSpace(key)
	}

	for _, record := range sourceCategories {
		key := strings.TrimSpace(record.GetString("source_key"))
		if key == "" {
			continue
		}
		nodeCountByTop[findTopLevel(key)]++
	}

	productCountByTop := make(map[string]int)
	multiUnitCount := 0
	for _, record := range sourceProducts {
		if record.GetBool("has_multi_unit") {
			multiUnitCount++
		}
		keys := sourceProductObservedCategoryKeys(record)
		if len(keys) == 0 {
			keys = sourceProductCategoryKeys(record)
		}
		seenTop := map[string]struct{}{}
		for _, key := range keys {
			topKey := findTopLevel(key)
			if topKey == "" {
				continue
			}
			if _, ok := seenTop[topKey]; ok {
				continue
			}
			seenTop[topKey] = struct{}{}
			productCountByTop[topKey]++
		}
	}

	assetCountByTop := make(map[string]int)
	productObservedByID := make(map[string][]string, len(sourceProducts))
	for _, record := range sourceProducts {
		keys := sourceProductObservedCategoryKeys(record)
		if len(keys) == 0 {
			keys = sourceProductCategoryKeys(record)
		}
		productObservedByID[strings.TrimSpace(record.GetString("product_id"))] = keys
	}
	for _, record := range sourceAssets {
		keys := productObservedByID[strings.TrimSpace(record.GetString("product_id"))]
		seenTop := map[string]struct{}{}
		for _, key := range keys {
			topKey := findTopLevel(key)
			if topKey == "" {
				continue
			}
			if _, ok := seenTop[topKey]; ok {
				continue
			}
			seenTop[topKey] = struct{}{}
			assetCountByTop[topKey]++
		}
	}

	scopeOptions := make([]TargetSyncScopeOption, 0, len(topLevelKeys)+1)
	scopeOptions = append(scopeOptions, TargetSyncScopeOption{
		Key:          "",
		Label:        "已落库结果",
		NodeCount:    len(sourceCategories),
		ProductCount: len(sourceProducts),
		AssetCount:   len(sourceAssets),
	})
	for _, topKey := range topLevelKeys {
		scopeOptions = append(scopeOptions, TargetSyncScopeOption{
			Key:          topKey,
			Label:        targetSyncFirstNonEmpty(topLevelLabels[topKey], topKey),
			NodeCount:    nodeCountByTop[topKey],
			ProductCount: productCountByTop[topKey],
			AssetCount:   assetCountByTop[topKey],
		})
	}

	return TargetSyncLiveSummary{
		SourceMode:             targetSyncFirstNonEmpty(sourceMode, "stored"),
		ExpectedNodeCount:      len(sourceCategories),
		ExpectedProductCount:   len(sourceProducts),
		ExpectedAssetCount:     len(sourceAssets),
		ExpectedMultiUnitCount: multiUnitCount,
		TopLevelCount:          len(topLevelKeys),
		ScopeOptions:           scopeOptions,
		CategoryDiffs:          []TargetCategoryDiffItem{},
		CheckoutSources:        []TargetCheckoutSource{},
		UsedStoredData:         true,
	}, nil
}

func (s *Service) TargetSyncCheckoutLiveSummary(dataset miniappmodel.Dataset) TargetSyncCheckoutLiveSummary {
	return TargetSyncCheckoutLiveSummary{
		CheckoutSources: targetCheckoutSources(dataset),
	}
}

func targetCheckoutSources(dataset miniappmodel.Dataset) []TargetCheckoutSource {
	freightPreview := ""
	if action := findScenarioAction(dataset.CartOrder.Order.FreightCosts, "preview"); action != nil {
		freightPreview = action.ContractID
	}
	freightSelected := ""
	if action := findScenarioAction(dataset.CartOrder.Order.FreightCosts, "selected_delivery"); action != nil {
		freightSelected = action.ContractID
	}

	return []TargetCheckoutSource{
		newTargetCheckoutSource("cart_list", "购物车列表", dataset.CartOrder.Cart.List.ContractID, "读取购物车清单与合计"),
		newTargetCheckoutSource("cart_detail", "购物车详情", dataset.CartOrder.Cart.Detail.ContractID, "读取结算页商品明细"),
		newTargetCheckoutSource("cart_settle", "结算预览", dataset.CartOrder.Cart.Settle.ContractID, "读取结算前校验结果"),
		newTargetCheckoutSource("default_delivery", "默认地址", dataset.CartOrder.Order.DefaultDelivery.ContractID, "读取默认收货地址"),
		newTargetCheckoutSource("deliveries", "地址列表", dataset.CartOrder.Order.Deliveries.ContractID, "读取全部收货地址"),
		newTargetCheckoutSource("analyse_address", "地址解析", dataset.CartOrder.Order.AnalyseAddress.ContractID, "解析文本地址"),
		newTargetCheckoutSource("freight_preview", "运费预估", freightPreview, "未选择配送方式时的运费试算"),
		newTargetCheckoutSource("freight_selected_delivery", "运费确认", freightSelected, "已选择配送方式后的运费试算"),
		newTargetCheckoutSource("add_delivery", "添加地址", dataset.CartOrder.Order.AddDelivery.ContractID, "显式调用时真实写入"),
		newTargetCheckoutSource("submit", "提交订单", dataset.CartOrder.Order.Submit.ContractID, "显式调用时真实下单"),
	}
}

func newTargetCheckoutSource(key string, label string, contractID string, note string) TargetCheckoutSource {
	status := "fallback"
	trimmed := strings.TrimSpace(contractID)
	switch {
	case strings.HasPrefix(trimmed, "raw_"):
		status = "raw_live"
		if key == "default_delivery" || key == "deliveries" || key == "analyse_address" || key == "freight_preview" || key == "freight_selected_delivery" {
			status = "raw_readonly"
		}
		if key == "add_delivery" || key == "submit" {
			status = "explicit_write"
		}
	case trimmed == "":
		status = "fallback"
	}
	return TargetCheckoutSource{
		Key:        key,
		Label:      label,
		Status:     status,
		ContractID: contractID,
		Note:       note,
	}
}

func findScenarioAction(items []miniappmodel.ScenarioAction, scenario string) *miniappmodel.ScenarioAction {
	for idx := range items {
		if items[idx].Scenario == scenario {
			return &items[idx]
		}
	}
	return nil
}

func targetMiniappWrites(app core.App, limit int) []TargetMiniappWrite {
	records, err := app.FindRecordsByFilter(CollectionMiniappActionLogs, "", "-created", limit, 0, nil)
	if err != nil {
		return nil
	}
	items := make([]TargetMiniappWrite, 0, len(records))
	for _, record := range records {
		items = append(items, TargetMiniappWrite{
			OperationID:    record.GetString("operation_id"),
			OperationLabel: record.GetString("operation_label"),
			ContractID:     record.GetString("contract_id"),
			Status:         record.GetString("status"),
			Message:        record.GetString("message"),
			CreatedAt:      record.GetString("created"),
		})
	}
	return items
}

func (s *Service) EnsureTargetSyncJob(_ context.Context, app core.App, dataset miniappmodel.Dataset, entityType string, scopeKey string) (TargetSyncJob, error) {
	return s.EnsureTargetSyncJobSpec(app, dataset.Meta.Source, entityType, scopeKey, resolveTargetSyncScopeLabel(&dataset, scopeKey, ""))
}

func (s *Service) EnsureTargetSyncJobSpec(app core.App, sourceMode string, entityType string, scopeKey string, scopeLabel string) (TargetSyncJob, error) {
	entityType = normalizeTargetSyncEntity(entityType)
	scopeType := targetScopeType(scopeKey)
	scopeKey = strings.TrimSpace(scopeKey)
	scopeLabel = resolveTargetSyncScopeLabel(nil, scopeKey, scopeLabel)
	jobKey := entityType + ":all"
	if scopeType == TargetSyncScopeTopLevel {
		jobKey = entityType + ":" + scopeKey
	}
	name := targetSyncEntityActionLabel(entityType)
	if scopeType == TargetSyncScopeTopLevel {
		name = name + " / " + scopeLabel
	}

	_, err := upsertByFilter(app, CollectionTargetSyncJobs, "job_key = {:job_key}", dbx.Params{"job_key": jobKey}, func(record *core.Record, created bool) error {
		record.Set("job_key", jobKey)
		record.Set("name", name)
		record.Set("entity_type", entityType)
		record.Set("scope_type", scopeType)
		record.Set("scope_key", scopeKey)
		record.Set("scope_label", scopeLabel)
		record.Set("status", defaultTargetSyncStatus(record.GetString("status"), created))
		record.Set("source_mode", strings.TrimSpace(sourceMode))
		return setJSON(record, "config_json", map[string]any{
			"scopeKey":   scopeKey,
			"scopeLabel": scopeLabel,
		})
	})
	if err != nil {
		return TargetSyncJob{}, err
	}

	record, err := app.FindFirstRecordByFilter(CollectionTargetSyncJobs, "job_key = {:job_key}", dbx.Params{"job_key": jobKey})
	if err != nil {
		return TargetSyncJob{}, err
	}
	return targetSyncJobFromRecord(record), nil
}

func (s *Service) StartTargetSyncAsync(app core.App, sourceLoader targetSyncDatasetLoader, productLoader targetSyncSectionProductLoader, entityType string, scopeKey string, scopeLabel string, actor TargetSyncActor) (TargetSyncRun, error) {
	if sourceLoader == nil {
		return TargetSyncRun{}, fmt.Errorf("target sync source loader is nil")
	}
	if productLoader == nil {
		return TargetSyncRun{}, fmt.Errorf("target sync product loader is nil")
	}
	if err := s.reconcileStaleTargetSyncRuns(app); err != nil {
		return TargetSyncRun{}, err
	}
	job, err := s.EnsureTargetSyncJobSpec(app, strings.TrimSpace(s.cfg.MiniApp.SourceMode), entityType, scopeKey, scopeLabel)
	if err != nil {
		return TargetSyncRun{}, err
	}
	if existing, ok, err := s.currentActiveTargetSyncRun(app, job.JobKey); ok {
		return existing, fmt.Errorf("该抓取任务已在执行中")
	} else if err != nil {
		return TargetSyncRun{}, err
	}

	runRecord, err := s.createTargetSyncRun(app, job, actor)
	if err != nil {
		return TargetSyncRun{}, err
	}
	s.setActiveTargetSyncRun(job.JobKey, runRecord.Id)

	go func(job TargetSyncJob, record *core.Record) {
		defer s.clearActiveTargetSyncRun(job.JobKey, record.Id)

		runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		tracker := newTargetSyncProgressTracker(app, record)
		if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityCategoryRebuild {
			_ = tracker.setStage("categories", "重建分类商品归属", 0)
			_ = tracker.addLog("categories", "info", "开始基于已保存分类商品来源重建商品分类归属。")
		} else {
			_ = tracker.setStage("loading_dataset", "加载源站数据集", 0)
			_ = tracker.addLog("loading_dataset", "info", "开始读取当前源站数据集。")
		}

		if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityAssets && strings.TrimSpace(job.ScopeKey) == "" {
			if result, ok, fastErr := s.trySyncTargetAssetsFromSourceProducts(runCtx, app, job, tracker); fastErr != nil {
				_ = tracker.addLog("loading_dataset", "warning", "已落库商品快捷路径失败，回退到源站数据集加载："+fastErr.Error())
			} else if ok {
				_, _ = s.finalizeTargetSyncRun(app, record, result)
				_ = s.updateTargetSyncJobStatus(app, job.JobKey, result)
				return
			}
		}

		result, runErr := s.runTargetSyncJob(runCtx, app, sourceLoader, productLoader, job, record, tracker)
		if runErr != nil {
			result.status = TargetSyncStatusFailed
			result.errorMessage = runErr.Error()
		}

		_, _ = s.finalizeTargetSyncRun(app, record, result)
		_ = s.updateTargetSyncJobStatus(app, job.JobKey, result)
	}(job, runRecord)

	return targetSyncRunFromRecord(runRecord), nil
}

func (s *Service) runTargetSyncJob(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	productLoader targetSyncSectionProductLoader,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityCategoryTree {
		return s.runCategoryTreeSync(ctx, app, sourceLoader, job, record, tracker)
	}
	if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityCategorySources {
		return s.runCategorySourceSync(ctx, app, sourceLoader, job, record, tracker)
	}
	if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityCategoryRebuild {
		return s.runCategoryRebuildSync(ctx, app, job, record, tracker)
	}

	if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityProducts && strings.TrimSpace(job.ScopeKey) == "" {
		return s.runTopLevelTargetProductSync(ctx, app, sourceLoader, productLoader, job, record, tracker)
	}

	if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityProducts {
		return s.runScopedTargetProductSync(ctx, app, sourceLoader, productLoader, job, record, tracker)
	}

	datasetCtx, datasetCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(job.EntityType))
	dataset, loadErr := sourceLoader(datasetCtx, job.EntityType, job.ScopeKey)
	datasetCancel()
	if loadErr != nil {
		if normalizeTargetSyncEntity(job.EntityType) == TargetSyncEntityProducts {
			if storedDataset, ok, storedErr := s.storedTargetProductsDataset(app, job.ScopeKey); storedErr != nil {
				_ = tracker.addLog("loading_dataset", "error", "加载已保存分类商品快照失败："+storedErr.Error())
			} else if ok {
				_ = tracker.addLog("loading_dataset", "warning", "实时源站商品抓取失败，已回退到已保存分类商品快照："+loadErr.Error())
				job.SourceMode = strings.TrimSpace(storedDataset.Meta.Source)
				job.ScopeLabel = resolveTargetSyncScopeLabel(storedDataset, job.ScopeKey, job.ScopeLabel)
				record.Set("source_mode", job.SourceMode)
				record.Set("scope_label", job.ScopeLabel)
				_ = app.Save(record)
				return s.executeTargetSync(ctx, app, *storedDataset, job, tracker)
			}
		}
		_ = tracker.addLog("loading_dataset", "error", "加载源站数据集失败："+loadErr.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   job.ScopeLabel,
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载源站数据集失败：" + loadErr.Error(),
		}, loadErr
	}

	job.ScopeLabel = resolveTargetSyncScopeLabel(dataset, job.ScopeKey, job.ScopeLabel)
	job.SourceMode = strings.TrimSpace(dataset.Meta.Source)
	record.Set("source_mode", job.SourceMode)
	record.Set("scope_label", job.ScopeLabel)
	_ = app.Save(record)

	result, err := s.executeTargetSync(ctx, app, *dataset, job, tracker)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) runCategoryTreeSync(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	treeCtx, treeCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityCategoryTree))
	treeDataset, err := sourceLoader(treeCtx, TargetSyncEntityCategoryTree, job.ScopeKey)
	treeCancel()
	if err != nil {
		_ = tracker.addLog("loading_dataset", "error", "加载源站分类树失败："+err.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   job.ScopeLabel,
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载源站分类树失败：" + err.Error(),
		}, err
	}

	job.SourceMode = strings.TrimSpace(treeDataset.Meta.Source)
	job.ScopeLabel = resolveTargetSyncScopeLabel(treeDataset, job.ScopeKey, job.ScopeLabel)
	record.Set("source_mode", job.SourceMode)
	record.Set("scope_label", job.ScopeLabel)
	_ = app.Save(record)

	result, syncErr := s.syncTargetCategories(ctx, app, *treeDataset, job.ScopeKey, tracker)
	if syncErr != nil {
		return result, syncErr
	}
	if tracker != nil {
		_ = tracker.addLog("categories", "info", fmt.Sprintf("抓分类树完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) runCategorySourceSync(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	treeCtx, treeCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityCategoryTree))
	treeDataset, err := sourceLoader(treeCtx, TargetSyncEntityCategoryTree, job.ScopeKey)
	treeCancel()
	if err != nil {
		_ = tracker.addLog("loading_dataset", "error", "加载源站分类树失败："+err.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   job.ScopeLabel,
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载源站分类树失败：" + err.Error(),
		}, err
	}

	job.SourceMode = strings.TrimSpace(treeDataset.Meta.Source)
	job.ScopeLabel = resolveTargetSyncScopeLabel(treeDataset, job.ScopeKey, job.ScopeLabel)
	record.Set("source_mode", job.SourceMode)
	record.Set("scope_label", job.ScopeLabel)
	_ = app.Save(record)

	nodes := treeDataset.CategoryPage.Tree
	if key := strings.TrimSpace(job.ScopeKey); key != "" {
		node := findCategoryNode(nodes, key)
		if node == nil {
			return targetCategorySyncResult{}, fmt.Errorf("category scope not found: %s", key)
		}
		nodes = []miniappmodel.CategoryNode{*node}
	}

	requestNodes := append([]miniappmodel.CategoryNode(nil), nodes...)
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityCategorySources,
		jobKey:     job.JobKey,
		jobName:    job.Name,
		scopeType:  targetScopeType(job.ScopeKey),
		scopeKey:   strings.TrimSpace(job.ScopeKey),
		scopeLabel: job.ScopeLabel,
		status:     TargetSyncStatusSuccess,
		sourceMode: job.SourceMode,
	}
	if tracker != nil {
		_ = tracker.setStage("categories", "写入分类商品来源", len(requestNodes))
		_ = tracker.addLog("categories", "info", fmt.Sprintf("开始刷新 %d 个分类分支的商品来源。", len(requestNodes)))
	}

	failedBranches := make([]string, 0)
	for _, node := range requestNodes {
		if tracker != nil {
			_ = tracker.addLog("categories", "info", fmt.Sprintf("开始抓取分类商品来源分支 %s。", strings.TrimSpace(node.Label)))
		}
		branchCtx, branchCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityCategorySources))
		branchDataset, branchErr := sourceLoader(branchCtx, TargetSyncEntityCategorySources, node.Key)
		branchCancel()
		if branchErr != nil {
			result.status = TargetSyncStatusPartial
			result.missingCount++
			failedBranches = append(failedBranches, strings.TrimSpace(node.Label))
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
				ChangeType: "failed",
				TargetType: TargetSyncEntityCategorySources,
				TargetKey:  node.Key,
				Label:      node.Label,
				Path:       node.Label,
				Note:       "分类商品来源抓取失败：" + branchErr.Error(),
			})
			if tracker != nil {
				_ = tracker.addLog("categories", "warning", fmt.Sprintf("分支 %s 分类商品来源抓取失败：%v", strings.TrimSpace(node.Label), branchErr))
				_ = tracker.step(node.Label)
			}
			continue
		}
		created, updated, count, saveErr := s.syncTargetCategorySectionSnapshots(app, *branchDataset)
		if saveErr != nil {
			result.status = TargetSyncStatusPartial
			result.missingCount++
			failedBranches = append(failedBranches, strings.TrimSpace(node.Label))
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
				ChangeType: "failed",
				TargetType: TargetSyncEntityCategorySources,
				TargetKey:  node.Key,
				Label:      node.Label,
				Path:       node.Label,
				Note:       "分类商品来源保存失败：" + saveErr.Error(),
			})
			if tracker != nil {
				_ = tracker.addLog("categories", "warning", fmt.Sprintf("分支 %s 分类商品来源保存失败：%v", strings.TrimSpace(node.Label), saveErr))
				_ = tracker.step(node.Label)
			}
			continue
		}
		result.createdCount += created
		result.updatedCount += updated
		result.unchangedCount += max(count-created-updated, 0)
		result.scopedNodeCount += count
		if tracker != nil {
			_ = tracker.addLog("categories", "info", fmt.Sprintf("分支 %s 分类商品来源完成：保存 %d 个分类路径，新增 %d，更新 %d。", strings.TrimSpace(node.Label), count, created, updated))
			_ = tracker.step(node.Label)
		}
	}
	if len(failedBranches) > 0 && result.createdCount == 0 && result.updatedCount == 0 && result.unchangedCount == 0 {
		result.status = TargetSyncStatusFailed
		result.errorMessage = "分类商品来源全部抓取失败：" + strings.Join(failedBranches, "、")
	}
	if tracker != nil {
		_ = tracker.addLog("categories", "info", fmt.Sprintf("刷新分类商品来源完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) syncTargetCategorySectionSnapshots(app core.App, dataset miniappmodel.Dataset) (int, int, int, error) {
	createdCount := 0
	updatedCount := 0
	for _, section := range dataset.CategoryPage.Sections {
		created, changed, err := upsertSourceCategorySectionSnapshot(app, section, dataset.Meta.Source)
		if err != nil {
			return createdCount, updatedCount, 0, err
		}
		if created {
			createdCount++
		} else if changed {
			updatedCount++
		}
	}
	return createdCount, updatedCount, len(dataset.CategoryPage.Sections), nil
}

func (s *Service) syncTargetCategorySectionSnapshotDataset(app core.App, dataset miniappmodel.Dataset, scopeKey string, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityCategorySources,
		status:     TargetSyncStatusSuccess,
		sourceMode: dataset.Meta.Source,
		scopeType:  targetScopeType(scopeKey),
		scopeKey:   strings.TrimSpace(scopeKey),
		scopeLabel: targetScopeLabel(dataset, scopeKey),
	}
	if tracker != nil {
		_ = tracker.setStage("categories", "写入分类商品来源", len(dataset.CategoryPage.Sections))
	}
	created, updated, count, err := s.syncTargetCategorySectionSnapshots(app, dataset)
	if err != nil {
		return result, err
	}
	result.createdCount = created
	result.updatedCount = updated
	result.unchangedCount = max(count-created-updated, 0)
	result.scopedNodeCount = count
	if tracker != nil {
		_ = tracker.addLog("categories", "info", fmt.Sprintf("刷新分类商品来源完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) runTopLevelTargetProductSync(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	productLoader targetSyncSectionProductLoader,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	topLevelNodes, sourceMode, err := s.targetSyncTopLevelNodes(ctx, app, sourceLoader)
	if err != nil {
		_ = tracker.addLog("loading_dataset", "error", "加载分类分支失败："+err.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   "当前源站结果",
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载分类分支失败：" + err.Error(),
		}, err
	}
	job.SourceMode = sourceMode
	job.ScopeLabel = "当前源站结果"
	record.Set("source_mode", job.SourceMode)
	record.Set("scope_label", job.ScopeLabel)
	_ = app.Save(record)
	if len(topLevelNodes) == 0 {
		err := fmt.Errorf("源站分类树为空，无法按分类分支抓取商品")
		_ = tracker.addLog("loading_dataset", "error", err.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    TargetSyncScopeAll,
			scopeKey:     "",
			scopeLabel:   "当前源站结果",
			status:       TargetSyncStatusFailed,
			sourceMode:   job.SourceMode,
			errorMessage: err.Error(),
		}, err
	}

	result := targetCategorySyncResult{
		entityType: TargetSyncEntityProducts,
		jobKey:     job.JobKey,
		jobName:    job.Name,
		scopeType:  TargetSyncScopeAll,
		scopeKey:   "",
		scopeLabel: "当前源站结果",
		status:     TargetSyncStatusSuccess,
		sourceMode: job.SourceMode,
	}

	if tracker != nil {
		_ = tracker.addLog("loading_dataset", "info", fmt.Sprintf("已加载 %d 个顶级分类，开始按分支抓取商品。", len(topLevelNodes)))
		_ = tracker.setStage("products", "按分类分支抓商品规格", len(topLevelNodes))
		_ = tracker.addLog("products", "info", fmt.Sprintf("开始按 %d 个顶级分支抓取商品规格。", len(topLevelNodes)))
	}

	failedBranches := make([]string, 0)
	for _, node := range topLevelNodes {
		if tracker != nil {
			_ = tracker.addLog("products", "info", fmt.Sprintf("开始抓取分支 %s。", strings.TrimSpace(node.Label)))
		}
		sections, usedStoredSections, sectionErr := s.targetSyncCategorySectionsForScope(ctx, app, sourceLoader, node.Key)
		if sectionErr != nil {
			failedBranches = append(failedBranches, strings.TrimSpace(node.Label))
			result.status = TargetSyncStatusPartial
			result.missingCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
				ChangeType: "failed",
				TargetType: TargetSyncEntityProducts,
				TargetKey:  node.Key,
				Label:      node.Label,
				Path:       node.Label,
				Note:       "分类商品来源读取失败：" + sectionErr.Error(),
			})
			if tracker != nil {
				_ = tracker.addLog("products", "warning", fmt.Sprintf("分支 %s 分类商品来源读取失败：%v", strings.TrimSpace(node.Label), sectionErr))
				_ = tracker.step(node.Label)
			}
			continue
		}
		if tracker != nil {
			if usedStoredSections {
				_ = tracker.addLog("products", "info", fmt.Sprintf("分支 %s 优先使用已保存分类商品来源。", strings.TrimSpace(node.Label)))
			} else {
				_ = tracker.addLog("products", "info", fmt.Sprintf("分支 %s 未命中本地来源，已即时刷新分类商品来源。", strings.TrimSpace(node.Label)))
			}
		}
		branchCtx, branchCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityProducts))
		branchDataset, branchErr := productLoader(branchCtx, sections, node.Key)
		branchCancel()
		if branchErr != nil {
			failedBranches = append(failedBranches, strings.TrimSpace(node.Label))
			result.status = TargetSyncStatusPartial
			result.missingCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
				ChangeType: "failed",
				TargetType: TargetSyncEntityProducts,
				TargetKey:  node.Key,
				Label:      node.Label,
				Path:       node.Label,
				Note:       "商品规格抓取失败：" + branchErr.Error(),
			})
			if tracker != nil {
				_ = tracker.addLog("products", "warning", fmt.Sprintf("分支 %s 商品规格抓取失败：%v", strings.TrimSpace(node.Label), branchErr))
				_ = tracker.step(node.Label)
			}
			continue
		}
		branchFailures := collectRawCategoryProductFailures(branchDataset.Meta.Notes)
		if len(branchFailures) > 0 {
			result.status = TargetSyncStatusPartial
			result.missingCount += len(branchFailures)
			for _, failed := range branchFailures {
				result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
					ChangeType: "failed",
					TargetType: TargetSyncEntityProducts,
					TargetKey:  failed.Key,
					Label:      failed.Label,
					Path:       failed.Label,
					Note:       failed.Note,
				})
				if tracker != nil {
					_ = tracker.addLog("products", "warning", failed.Note)
				}
			}
		}

		branchResult, syncErr := s.syncTargetProducts(ctx, app, *branchDataset, node.Key, nil)
		if syncErr != nil {
			failedBranches = append(failedBranches, strings.TrimSpace(node.Label))
			result.status = TargetSyncStatusPartial
			result.missingCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{
				ChangeType: "failed",
				TargetType: TargetSyncEntityProducts,
				TargetKey:  node.Key,
				Label:      node.Label,
				Path:       node.Label,
				Note:       "分支入库失败：" + syncErr.Error(),
			})
			if tracker != nil {
				_ = tracker.addLog("products", "warning", fmt.Sprintf("分支 %s 入库失败：%v", strings.TrimSpace(node.Label), syncErr))
				_ = tracker.step(node.Label)
			}
			continue
		}

		result.createdCount += branchResult.createdCount
		result.updatedCount += branchResult.updatedCount
		result.unchangedCount += branchResult.unchangedCount
		result.scopedNodeCount += branchResult.scopedNodeCount
		for _, item := range branchResult.details {
			result.details = appendTargetSyncDetail(result.details, item)
		}
		if tracker != nil {
			_ = tracker.addLog("products", "info", fmt.Sprintf(
				"分支 %s 抓取完成：抓到 %d 个商品，新增 %d，更新 %d，未变化 %d。",
				strings.TrimSpace(node.Label),
				branchResult.scopedNodeCount,
				branchResult.createdCount,
				branchResult.updatedCount,
				branchResult.unchangedCount,
			))
			_ = tracker.step(node.Label)
		}
	}

	if len(failedBranches) > 0 && result.createdCount == 0 && result.updatedCount == 0 && result.unchangedCount == 0 {
		result.status = TargetSyncStatusFailed
		result.errorMessage = "所有分类分支抓取失败：" + strings.Join(failedBranches, "、")
	}
	if tracker != nil {
		if len(failedBranches) > 0 {
			_ = tracker.addLog("products", "warning", fmt.Sprintf("分支抓取完成：成功 %d 个，失败 %d 个。", len(topLevelNodes)-len(failedBranches), len(failedBranches)))
		}
		_ = tracker.addLog("products", "info", fmt.Sprintf("按当前源站结果抓商品规格完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) runScopedTargetProductSync(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	productLoader targetSyncSectionProductLoader,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	sections, usedStoredSections, err := s.targetSyncCategorySectionsForScope(ctx, app, sourceLoader, job.ScopeKey)
	if err != nil {
		_ = tracker.addLog("loading_dataset", "error", "加载分类商品来源失败："+err.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   job.ScopeLabel,
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载分类商品来源失败：" + err.Error(),
		}, err
	}
	if tracker != nil {
		if usedStoredSections {
			_ = tracker.addLog("loading_dataset", "info", "检测到已保存分类商品来源，商品规格抓取优先复用来源。")
		} else {
			_ = tracker.addLog("loading_dataset", "info", "未命中已保存分类商品来源，已即时刷新来源后继续抓取商品规格。")
		}
	}
	datasetCtx, datasetCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityProducts))
	dataset, loadErr := productLoader(datasetCtx, sections, job.ScopeKey)
	datasetCancel()
	if loadErr != nil {
		_ = tracker.addLog("loading_dataset", "error", "加载商品规格失败："+loadErr.Error())
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    job.ScopeType,
			scopeKey:     job.ScopeKey,
			scopeLabel:   job.ScopeLabel,
			status:       TargetSyncStatusFailed,
			sourceMode:   strings.TrimSpace(s.cfg.MiniApp.SourceMode),
			errorMessage: "加载商品规格失败：" + loadErr.Error(),
		}, loadErr
	}
	job.SourceMode = strings.TrimSpace(dataset.Meta.Source)
	job.ScopeLabel = resolveTargetSyncScopeLabel(dataset, job.ScopeKey, job.ScopeLabel)
	record.Set("source_mode", job.SourceMode)
	record.Set("scope_label", job.ScopeLabel)
	_ = app.Save(record)
	return s.executeTargetSync(ctx, app, *dataset, job, tracker)
}

func (s *Service) runCategoryRebuildSync(
	_ context.Context,
	app core.App,
	job TargetSyncJob,
	record *core.Record,
	tracker *targetSyncProgressTracker,
) (targetCategorySyncResult, error) {
	result, err := s.rebuildSourceProductCategoryAssignments(app, job.ScopeKey, tracker)
	if err != nil {
		return targetCategorySyncResult{
			entityType:   job.EntityType,
			jobKey:       job.JobKey,
			jobName:      job.Name,
			scopeType:    targetScopeType(job.ScopeKey),
			scopeKey:     strings.TrimSpace(job.ScopeKey),
			scopeLabel:   targetSyncFirstNonEmpty(job.ScopeLabel, "当前已保存分类来源"),
			status:       TargetSyncStatusFailed,
			sourceMode:   "source_category_sections",
			errorMessage: "重建分类商品归属失败：" + err.Error(),
		}, err
	}
	record.Set("source_mode", "source_category_sections")
	record.Set("scope_label", result.scopeLabel)
	_ = app.Save(record)
	return result, nil
}

func (s *Service) targetSyncTopLevelNodes(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
) ([]miniappmodel.CategoryNode, string, error) {
	if nodes, ok, err := s.storedTargetSyncTopLevelNodes(app); err != nil {
		return nil, "", err
	} else if ok {
		return nodes, "source_categories", nil
	}
	treeCtx, treeCancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityCategoryTree))
	treeDataset, err := sourceLoader(treeCtx, TargetSyncEntityCategoryTree, "")
	treeCancel()
	if err != nil {
		return nil, "", err
	}
	return append([]miniappmodel.CategoryNode(nil), treeDataset.CategoryPage.Tree...), strings.TrimSpace(treeDataset.Meta.Source), nil
}

func (s *Service) storedTargetSyncTopLevelNodes(app core.App) ([]miniappmodel.CategoryNode, bool, error) {
	records, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return nil, false, err
	}
	if len(records) == 0 {
		return nil, false, nil
	}
	nodes := sourceCategoryTreeFromRecords(records)
	if len(nodes) == 0 {
		return nil, false, nil
	}
	return nodes, true, nil
}

func (s *Service) targetSyncCategorySectionsForScope(
	ctx context.Context,
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	scopeKey string,
) ([]miniappmodel.CategorySection, bool, error) {
	if sections, ok, err := s.storedCategorySectionsForScope(app, scopeKey); err != nil {
		return nil, false, err
	} else if ok {
		return sections, true, nil
	}
	loadCtx, cancel := context.WithTimeout(ctx, s.targetSyncDatasetLoadTimeout(TargetSyncEntityCategorySources))
	dataset, err := sourceLoader(loadCtx, TargetSyncEntityCategorySources, scopeKey)
	cancel()
	if err != nil {
		return nil, false, err
	}
	if _, _, _, err := s.syncTargetCategorySectionSnapshots(app, *dataset); err != nil {
		return nil, false, err
	}
	sections := append([]miniappmodel.CategorySection(nil), dataset.CategoryPage.Sections...)
	if len(sections) == 0 {
		return nil, false, fmt.Errorf("分类商品来源为空")
	}
	return sections, false, nil
}

func (s *Service) storedCategorySectionsForScope(app core.App, scopeKey string) ([]miniappmodel.CategorySection, bool, error) {
	records, err := app.FindAllRecords(CollectionSourceCategorySections)
	if err != nil {
		return nil, false, err
	}
	sections := make([]miniappmodel.CategorySection, 0, len(records))
	for _, record := range records {
		if !sourceCategorySectionApplies(record, scopeKey) {
			continue
		}
		section, ok := sourceCategorySectionFromRecord(record)
		if !ok {
			continue
		}
		sections = append(sections, section)
	}
	if len(sections) == 0 {
		return nil, false, nil
	}
	return sections, true, nil
}

type rawCategoryProductFailure struct {
	Key   string
	Label string
	Note  string
}

func collectRawCategoryProductFailures(notes []string) []rawCategoryProductFailure {
	items := make([]rawCategoryProductFailure, 0)
	seen := make(map[string]struct{})
	for _, note := range notes {
		failure, ok := parseRawCategoryProductFailure(note)
		if !ok {
			continue
		}
		if _, exists := seen[failure.Key]; exists {
			continue
		}
		seen[failure.Key] = struct{}{}
		items = append(items, failure)
	}
	return items
}

func parseRawCategoryProductFailure(note string) (rawCategoryProductFailure, bool) {
	const prefix = "raw 分类商品跳过 "
	trimmed := strings.TrimSpace(note)
	if !strings.HasPrefix(trimmed, prefix) {
		return rawCategoryProductFailure{}, false
	}
	rest := strings.TrimPrefix(trimmed, prefix)
	splitIdx := strings.Index(rest, "）：")
	if splitIdx < 0 {
		return rawCategoryProductFailure{}, false
	}
	head := rest[:splitIdx]
	errNote := strings.TrimSpace(rest[splitIdx+len("）："):])
	keyStart := strings.LastIndex(head, "（")
	if keyStart < 0 {
		return rawCategoryProductFailure{}, false
	}
	label := strings.TrimSpace(head[:keyStart])
	key := strings.TrimSpace(head[keyStart+len("（"):])
	if key == "" {
		return rawCategoryProductFailure{}, false
	}
	return rawCategoryProductFailure{
		Key:   key,
		Label: targetSyncFirstNonEmpty(label, key),
		Note:  "分类路径抓取失败：" + targetSyncFirstNonEmpty(label, key) + "：" + errNote,
	}, true
}

func (s *Service) RunTargetSync(ctx context.Context, app core.App, dataset miniappmodel.Dataset, entityType string, scopeKey string, actor TargetSyncActor) (TargetSyncRun, error) {
	entityType = normalizeTargetSyncEntity(entityType)
	job, err := s.EnsureTargetSyncJob(ctx, app, dataset, entityType, scopeKey)
	if err != nil {
		return TargetSyncRun{}, err
	}
	if existing, ok, err := s.currentActiveTargetSyncRun(app, job.JobKey); ok {
		return existing, fmt.Errorf("该抓取任务已在执行中")
	} else if err != nil {
		return TargetSyncRun{}, err
	}

	runRecord, err := s.createTargetSyncRun(app, job, actor)
	if err != nil {
		return TargetSyncRun{}, err
	}
	s.setActiveTargetSyncRun(job.JobKey, runRecord.Id)
	defer s.clearActiveTargetSyncRun(job.JobKey, runRecord.Id)

	tracker := newTargetSyncProgressTracker(app, runRecord)
	_ = tracker.setStage("loading_dataset", "准备执行抓取入库", 0)
	_ = tracker.addLog("loading_dataset", "info", "已加载数据集，准备执行抓取入库。")

	result, runErr := s.executeTargetSync(ctx, app, dataset, job, tracker)
	if runErr != nil {
		result.status = TargetSyncStatusFailed
		result.errorMessage = runErr.Error()
	}

	finalRun, saveErr := s.finalizeTargetSyncRun(app, runRecord, result)
	if saveErr != nil {
		return TargetSyncRun{}, saveErr
	}
	if updateErr := s.updateTargetSyncJobStatus(app, job.JobKey, result); updateErr != nil {
		return TargetSyncRun{}, updateErr
	}
	if runErr != nil {
		return finalRun, runErr
	}
	return finalRun, nil
}

func (s *Service) executeTargetSync(ctx context.Context, app core.App, dataset miniappmodel.Dataset, job TargetSyncJob, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	var (
		result targetCategorySyncResult
		err    error
	)
	switch normalizeTargetSyncEntity(job.EntityType) {
	case TargetSyncEntityProducts:
		result, err = s.syncTargetProducts(ctx, app, dataset, job.ScopeKey, tracker)
	case TargetSyncEntityAssets:
		result, err = s.syncTargetAssets(ctx, app, dataset, job.ScopeKey, tracker)
	case TargetSyncEntityCategorySources:
		result, err = s.syncTargetCategorySectionSnapshotDataset(app, dataset, job.ScopeKey, tracker)
	default:
		result, err = s.syncTargetCategories(ctx, app, dataset, job.ScopeKey, tracker)
	}
	result.jobKey = job.JobKey
	result.jobName = job.Name
	result.scopeType = job.ScopeType
	result.scopeKey = job.ScopeKey
	result.scopeLabel = resolveTargetSyncScopeLabel(&dataset, job.ScopeKey, job.ScopeLabel)
	result.sourceMode = dataset.Meta.Source
	result.entityType = normalizeTargetSyncEntity(job.EntityType)
	return result, err
}

func (s *Service) trySyncTargetAssetsFromSourceProducts(ctx context.Context, app core.App, job TargetSyncJob, tracker *targetSyncProgressTracker) (targetCategorySyncResult, bool, error) {
	products, err := s.sourceProductsForAssetSync(app)
	if err != nil {
		return targetCategorySyncResult{}, false, err
	}
	if len(products) == 0 {
		if tracker != nil {
			_ = tracker.addLog("loading_dataset", "info", "未发现已落库商品，继续回退到源站数据集加载。")
		}
		return targetCategorySyncResult{}, false, nil
	}
	if tracker != nil {
		_ = tracker.addLog("loading_dataset", "info", fmt.Sprintf("检测到 %d 条已落库商品，图片资产抓取入库改走本地快捷路径。", len(products)))
	}
	result, err := s.syncTargetAssetsFromProducts(ctx, app, products, job, tracker)
	if err != nil {
		return targetCategorySyncResult{}, false, err
	}
	return result, true, nil
}

func (s *Service) syncTargetCategories(_ context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityCategoryTree,
		status:     TargetSyncStatusSuccess,
		sourceMode: dataset.Meta.Source,
	}
	nodes := dataset.CategoryPage.Tree
	scopeLabel := "当前源站分类树"
	scopeType := TargetSyncScopeAll
	if key := strings.TrimSpace(scopeKey); key != "" {
		node := findCategoryNode(nodes, key)
		if node == nil {
			return result, fmt.Errorf("category scope not found: %s", key)
		}
		nodes = []miniappmodel.CategoryNode{*node}
		scopeLabel = node.Label
		scopeType = TargetSyncScopeTopLevel
	}
	result.scopeType = scopeType
	result.scopeKey = strings.TrimSpace(scopeKey)
	result.scopeLabel = scopeLabel

	scopedNodes := flattenCategoryNodes(nodes)
	result.scopedNodeCount = len(scopedNodes)
	if tracker != nil {
		_ = tracker.setStage("categories", "写入分类", len(scopedNodes))
		_ = tracker.addLog("categories", "info", fmt.Sprintf("开始抓取入库 %d 个分类节点。", len(scopedNodes)))
	}
	scopeMap := make(map[string]miniappmodel.CategoryNode, len(scopedNodes))
	for _, node := range scopedNodes {
		scopeMap[node.Key] = node
	}

	categoryPathByKey, parentKeyByKey, _ := buildCategoryTreeMeta(nodes)
	for _, node := range scopedNodes {
		if tracker != nil {
			_ = tracker.step(node.Label)
		}
		expectedPath := categoryPathByKey[node.Key]
		created, changed, upsertErr := upsertTargetCategoryNode(app, node, expectedPath, parentKeyByKey[node.Key], s.cfg.MiniApp.RawAssetBaseURL)
		if upsertErr != nil {
			return result, upsertErr
		}
		if created {
			result.createdCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "created", TargetType: TargetSyncEntityCategoryTree, TargetKey: node.Key, Label: node.Label, Path: expectedPath})
			continue
		}
		if changed {
			result.updatedCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "updated", TargetType: TargetSyncEntityCategoryTree, TargetKey: node.Key, Label: node.Label, Path: expectedPath})
			continue
		}
		result.unchangedCount++
	}

	if result.updatedCount == 0 && result.createdCount == 0 && result.unchangedCount > 0 {
		result.status = TargetSyncStatusSuccess
	}

	existingRecords, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return result, err
	}
	for _, record := range existingRecords {
		key := record.GetString("source_key")
		if _, ok := scopeMap[key]; !ok {
			continue
		}
	}
	result.missingCount = countMissingScopedCategories(existingRecords, scopeMap)
	if result.missingCount > 0 && result.status == TargetSyncStatusSuccess {
		result.status = TargetSyncStatusPartial
	}
	if tracker != nil {
		_ = tracker.addLog("categories", "info", fmt.Sprintf("分类树入库完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}

	return result, nil
}

func (s *Service) syncTargetProducts(ctx context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityProducts,
		status:     TargetSyncStatusSuccess,
		sourceMode: dataset.Meta.Source,
	}
	products := filteredTargetProducts(dataset, scopeKey)
	result.scopeType = targetScopeType(scopeKey)
	result.scopeKey = strings.TrimSpace(scopeKey)
	result.scopeLabel = targetScopeLabel(dataset, scopeKey)
	result.scopedNodeCount = len(products)
	if tracker != nil {
		_ = tracker.setStage("products", "写入商品规格", len(products))
		sourceLabel := "已保存分类快照"
		if strings.TrimSpace(dataset.Meta.Source) != "stored_category_sections" {
			sourceLabel = "当前源站结果"
		}
		_ = tracker.addLog("products", "info", fmt.Sprintf("开始基于%s抓取入库 %d 个商品规格。", sourceLabel, len(products)))
	}

	snapshotCreated := 0
	snapshotUpdated := 0
	for _, section := range dataset.CategoryPage.Sections {
		created, changed, err := upsertSourceCategorySectionSnapshot(app, section, dataset.Meta.Source)
		if err != nil {
			return result, err
		}
		if created {
			snapshotCreated++
		} else if changed {
			snapshotUpdated++
		}
	}
	if tracker != nil && (snapshotCreated > 0 || snapshotUpdated > 0) {
		_ = tracker.addLog("products", "info", fmt.Sprintf("已更新 %d 个分类商品快照（新增 %d，更新 %d）。", snapshotCreated+snapshotUpdated, snapshotCreated, snapshotUpdated))
	}

	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	_, parentByKey, hasChildrenByKey := buildCategoryTreeMeta(dataset.CategoryPage.Tree)
	for _, product := range products {
		if tracker != nil {
			_ = tracker.step(product.Summary.Name)
		}
		categoryKey, categoryPath, categoryKeys, observedCategoryKeys, observedCategoryPaths := productCategoryInfo(product, sections, parentByKey, hasChildrenByKey)
		existing, _ := app.FindFirstRecordByFilter(CollectionSourceProducts, "product_id = {:product_id}", dbx.Params{"product_id": product.ID})
		before := sourceProductSignature(existing)
		created, err := s.upsertSourceProduct(ctx, app, product, categoryKey, categoryPath, categoryKeys, observedCategoryKeys, observedCategoryPaths)
		if err != nil {
			return result, err
		}
		record, err := app.FindFirstRecordByFilter(CollectionSourceProducts, "product_id = {:product_id}", dbx.Params{"product_id": product.ID})
		if err != nil {
			return result, err
		}
		after := sourceProductSignature(record)
		if created {
			result.createdCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "created", TargetType: TargetSyncEntityProducts, TargetKey: product.ID, Label: product.Summary.Name, Path: categoryPath, Note: "商品新入库"})
			continue
		}
		if before != after {
			result.updatedCount++
			result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "updated", TargetType: TargetSyncEntityProducts, TargetKey: product.ID, Label: product.Summary.Name, Path: categoryPath, Note: "商品或规格发生变更，已回到待审核"})
			continue
		}
		result.unchangedCount++
	}
	if tracker != nil {
		sourceLabel := "已保存分类快照"
		if strings.TrimSpace(dataset.Meta.Source) != "stored_category_sections" {
			sourceLabel = "当前源站结果"
		}
		_ = tracker.addLog("products", "info", fmt.Sprintf("按%s抓商品规格完成：新增 %d，更新 %d，未变化 %d。", sourceLabel, result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) syncTargetAssets(ctx context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	return s.syncTargetAssetsFromProducts(ctx, app, filteredTargetProducts(dataset, scopeKey), TargetSyncJob{
		EntityType: TargetSyncEntityAssets,
		ScopeType:  targetScopeType(scopeKey),
		ScopeKey:   strings.TrimSpace(scopeKey),
		ScopeLabel: targetScopeLabel(dataset, scopeKey),
		SourceMode: dataset.Meta.Source,
	}, tracker)
}

func (s *Service) syncTargetAssetsFromProducts(_ context.Context, app core.App, products []miniappmodel.ProductPage, job TargetSyncJob, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityAssets,
		status:     TargetSyncStatusSuccess,
		sourceMode: targetSyncFirstNonEmpty(job.SourceMode, "source_products"),
	}
	result.scopeType = targetSyncFirstNonEmpty(job.ScopeType, TargetSyncScopeAll)
	result.scopeKey = strings.TrimSpace(job.ScopeKey)
	result.scopeLabel = targetSyncFirstNonEmpty(job.ScopeLabel, "当前源站结果")
	totalAssets := 0
	for _, product := range products {
		totalAssets += len(collectProductAssets(product))
	}
	result.scopedNodeCount = totalAssets
	if tracker != nil {
		_ = tracker.setStage("assets", "写入图片资源", totalAssets)
		_ = tracker.addLog("assets", "info", fmt.Sprintf("开始抓取入库 %d 个图片资源。", totalAssets))
	}
	for _, product := range products {
		assets := collectProductAssets(product)
		for _, asset := range assets {
			if tracker != nil {
				_ = tracker.step(product.Summary.Name + " / " + asset.Role)
			}
			existing, _ := app.FindFirstRecordByFilter(CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key})
			before := sourceAssetSignature(existing)
			created, err := upsertTargetAssetItem(app, product, asset, s.cfg.MiniApp.RawAssetBaseURL)
			if err != nil {
				return result, err
			}
			record, err := app.FindFirstRecordByFilter(CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key})
			if err != nil {
				return result, err
			}
			after := sourceAssetSignature(record)
			if created {
				result.createdCount++
				result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "created", TargetType: TargetSyncEntityAssets, TargetKey: asset.Key, Label: product.Summary.Name, Path: asset.Role, Note: "图片新入库"})
				continue
			}
			if before != after {
				result.updatedCount++
				result.details = appendTargetSyncDetail(result.details, TargetSyncChangeItem{ChangeType: "updated", TargetType: TargetSyncEntityAssets, TargetKey: asset.Key, Label: product.Summary.Name, Path: asset.Role, Note: "图片发生变更，已回到待处理"})
				continue
			}
			result.unchangedCount++
		}
	}
	if tracker != nil {
		_ = tracker.addLog("assets", "info", fmt.Sprintf("按当前源站结果抓图片完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func (s *Service) sourceProductsForAssetSync(app core.App) ([]miniappmodel.ProductPage, error) {
	records, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return nil, err
	}
	products := make([]miniappmodel.ProductPage, 0, len(records))
	for _, record := range records {
		product, ok := sourceProductPageFromRecord(record)
		if !ok {
			continue
		}
		products = append(products, product)
	}
	return products, nil
}

func (s *Service) rebuildSourceProductCategoryAssignments(app core.App, scopeKey string, tracker *targetSyncProgressTracker) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityCategoryRebuild,
		status:     TargetSyncStatusSuccess,
		sourceMode: "source_category_sections",
		scopeType:  targetScopeType(scopeKey),
		scopeKey:   strings.TrimSpace(scopeKey),
		scopeLabel: targetSyncFirstNonEmpty(strings.TrimSpace(scopeKey), "当前已保存分类来源"),
	}
	categoryRecords, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return result, err
	}
	sections, ok, err := s.storedCategorySectionsForScope(app, scopeKey)
	if err != nil {
		return result, err
	}
	if !ok || len(sections) == 0 {
		return result, fmt.Errorf("当前范围没有已保存分类商品来源")
	}
	sectionLookupByProductID := buildCategorySectionsByProductID(sections)
	if len(sectionLookupByProductID) == 0 {
		return result, fmt.Errorf("当前范围没有可用于重建归属的商品来源")
	}
	tree := sourceCategoryTreeFromRecords(categoryRecords)
	_, parentByKey, hasChildrenByKey := buildCategoryTreeMeta(tree)

	productRecords, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return result, err
	}
	targetRecords := make([]*core.Record, 0, len(productRecords))
	for _, record := range productRecords {
		productID := strings.TrimSpace(record.GetString("product_id"))
		if _, exists := sectionLookupByProductID[productID]; !exists {
			continue
		}
		targetRecords = append(targetRecords, record)
	}
	result.scopedNodeCount = len(targetRecords)
	if tracker != nil {
		_ = tracker.setStage("categories", "重建分类商品归属", len(targetRecords))
		_ = tracker.addLog("categories", "info", fmt.Sprintf("开始基于已保存分类商品来源重建 %d 个商品的分类归属。", len(targetRecords)))
	}
	for _, record := range targetRecords {
		product, ok := sourceProductPageFromRecord(record)
		if !ok {
			continue
		}
		before := sourceProductSignature(record)
		matchedSections := sectionLookupByProductID[product.ID]
		product.SourceSections = nil
		product.CategoryKey = ""
		product.CategoryPath = ""
		product.CategoryKeys = nil
		product.ObservedCategoryKeys = nil
		product.ObservedCategoryPaths = nil
		sectionsLookup := make(map[string]miniappmodel.CategorySection, len(matchedSections))
		for _, section := range matchedSections {
			product = appendObservedCategorySection(product, section)
			sectionsLookup[section.CategoryKey] = section
		}
		categoryKey, categoryPath, categoryKeys, observedCategoryKeys, observedCategoryPaths := productCategoryInfo(product, sectionsLookup, parentByKey, hasChildrenByKey)
		if err := setSourceProductCategoryAssignment(record, product, categoryKey, categoryPath, categoryKeys, observedCategoryKeys, observedCategoryPaths); err != nil {
			return result, err
		}
		if err := app.Save(record); err != nil {
			return result, err
		}
		after := sourceProductSignature(record)
		if before != after {
			result.updatedCount++
		} else {
			result.unchangedCount++
		}
		if tracker != nil {
			_ = tracker.step(product.Summary.Name)
		}
	}
	if tracker != nil {
		_ = tracker.addLog("categories", "info", fmt.Sprintf("重建分类商品归属完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount))
	}
	return result, nil
}

func buildCategorySectionsByProductID(sections []miniappmodel.CategorySection) map[string][]miniappmodel.CategorySection {
	lookup := make(map[string][]miniappmodel.CategorySection)
	for _, section := range sections {
		for _, item := range section.Products {
			productID := strings.TrimSpace(item.ProductID)
			if productID == "" && strings.TrimSpace(item.SpuID) != "" && strings.TrimSpace(item.SkuID) != "" {
				productID = strings.TrimSpace(item.SpuID) + "_" + strings.TrimSpace(item.SkuID)
			}
			if productID == "" {
				continue
			}
			lookup[productID] = append(lookup[productID], section)
		}
	}
	return lookup
}

func appendObservedCategorySection(product miniappmodel.ProductPage, section miniappmodel.CategorySection) miniappmodel.ProductPage {
	product.SourceSections = appendUniqueSourceStrings(product.SourceSections, section.CategoryKey)
	product.ObservedCategoryKeys = normalizeSourceCategoryKeys(append(product.ObservedCategoryKeys, section.CategoryKey), product.CategoryKey)
	product.ObservedCategoryPaths = normalizeSourceCategoryPaths(append(product.ObservedCategoryPaths, section.CategoryPath), product.CategoryPath)
	if strings.TrimSpace(product.CategoryKey) == "" || len(section.CategoryKeys) > len(product.CategoryKeys) {
		product.CategoryKey = section.CategoryKey
		product.CategoryPath = section.CategoryPath
		product.CategoryKeys = normalizeSourceCategoryKeys(section.CategoryKeys, section.CategoryKey)
	}
	return product
}

func setSourceProductCategoryAssignment(record *core.Record, product miniappmodel.ProductPage, categoryKey string, categoryPath string, categoryKeys []string, observedCategoryKeys []string, observedCategoryPaths []string) error {
	record.Set("category_key", categoryKey)
	record.Set("category_path", categoryPath)
	record.Set("leaf_category_key", categoryKey)
	record.Set("leaf_category_path", categoryPath)
	if err := setJSON(record, "source_sections", product.SourceSections); err != nil {
		return err
	}
	if err := setJSON(record, "category_keys_json", normalizeSourceCategoryKeys(categoryKeys, categoryKey)); err != nil {
		return err
	}
	if err := setJSON(record, "observed_category_keys_json", normalizeSourceCategoryKeys(observedCategoryKeys, categoryKey)); err != nil {
		return err
	}
	if err := setJSON(record, "observed_category_paths_json", normalizeSourceCategoryPaths(observedCategoryPaths, categoryPath)); err != nil {
		return err
	}
	return nil
}

func (s *Service) storedTargetProductsDataset(app core.App, scopeKey string) (*miniappmodel.Dataset, bool, error) {
	categoryRecords, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return nil, false, err
	}
	sectionRecords, err := app.FindAllRecords(CollectionSourceCategorySections)
	if err != nil {
		return nil, false, err
	}
	filteredSectionRecords := make([]*core.Record, 0, len(sectionRecords))
	productIDs := make(map[string]struct{})
	for _, record := range sectionRecords {
		if !sourceCategorySectionApplies(record, scopeKey) {
			continue
		}
		filteredSectionRecords = append(filteredSectionRecords, record)
		for _, productID := range sourceCategorySectionProductIDs(record) {
			productIDs[strings.TrimSpace(productID)] = struct{}{}
		}
	}
	if len(filteredSectionRecords) == 0 || len(productIDs) == 0 {
		return nil, false, nil
	}

	productRecords, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return nil, false, err
	}
	products := make([]miniappmodel.ProductPage, 0, len(productIDs))
	for _, record := range productRecords {
		productID := strings.TrimSpace(record.GetString("product_id"))
		if _, ok := productIDs[productID]; !ok {
			continue
		}
		product, ok := sourceProductPageFromRecord(record)
		if !ok {
			continue
		}
		products = append(products, product)
	}
	if len(products) == 0 {
		return nil, false, nil
	}

	sections := make([]miniappmodel.CategorySection, 0, len(filteredSectionRecords))
	for _, record := range filteredSectionRecords {
		section, ok := sourceCategorySectionFromRecord(record)
		if !ok {
			continue
		}
		sections = append(sections, section)
	}
	if len(sections) == 0 {
		return nil, false, nil
	}

	return &miniappmodel.Dataset{
		Meta: miniappmodel.Meta{
			Source:      "stored_category_sections",
			Description: "Persisted category goods list snapshots",
			Notes: []string{
				"已回退到已保存分类商品快照。",
			},
		},
		CategoryPage: miniappmodel.CategoryPageAggregate{
			Tree:     sourceCategoryTreeFromRecords(categoryRecords),
			Sections: sections,
		},
		ProductPage: miniappmodel.ProductPageAggregate{
			Products: products,
		},
	}, true, nil
}

func sourceProductPageFromRecord(record *core.Record) (miniappmodel.ProductPage, bool) {
	if record == nil {
		return miniappmodel.ProductPage{}, false
	}
	product := miniappmodel.ProductPage{
		ID:           strings.TrimSpace(record.GetString("product_id")),
		SpuID:        strings.TrimSpace(record.GetString("spu_id")),
		SkuID:        strings.TrimSpace(record.GetString("sku_id")),
		SourceType:   strings.TrimSpace(record.GetString("source_type")),
		CategoryKey:  strings.TrimSpace(record.GetString("category_key")),
		CategoryPath: strings.TrimSpace(record.GetString("category_path")),
	}
	if product.ID == "" {
		return miniappmodel.ProductPage{}, false
	}
	decodeTargetSyncJSONField(record.GetString("source_sections"), &product.SourceSections)
	decodeTargetSyncJSONField(record.GetString("category_keys_json"), &product.CategoryKeys)
	decodeTargetSyncJSONField(record.GetString("observed_category_keys_json"), &product.ObservedCategoryKeys)
	decodeTargetSyncJSONField(record.GetString("observed_category_paths_json"), &product.ObservedCategoryPaths)
	decodeTargetSyncJSONField(record.GetString("summary_json"), &product.Summary)
	decodeTargetSyncJSONField(record.GetString("detail_json"), &product.Detail)
	decodeTargetSyncJSONField(record.GetString("pricing_json"), &product.Pricing)
	decodeTargetSyncJSONField(record.GetString("package_json"), &product.Package)
	decodeTargetSyncJSONField(record.GetString("context_json"), &product.Context)
	if strings.TrimSpace(product.Summary.Name) == "" {
		product.Summary.Name = strings.TrimSpace(record.GetString("name"))
	}
	if strings.TrimSpace(product.Summary.Cover) == "" {
		product.Summary.Cover = strings.TrimSpace(record.GetString("cover_url"))
	}
	if strings.TrimSpace(product.Summary.DefaultUnit) == "" {
		product.Summary.DefaultUnit = strings.TrimSpace(record.GetString("default_unit"))
	}
	return product, true
}

func decodeTargetSyncJSONField[T any](raw string, target *T) {
	raw = strings.TrimSpace(raw)
	if raw == "" || target == nil {
		return
	}
	_ = json.Unmarshal([]byte(raw), target)
}

func countMissingScopedCategories(existing []*core.Record, scopeMap map[string]miniappmodel.CategoryNode) int {
	existingMap := make(map[string]struct{}, len(existing))
	for _, record := range existing {
		existingMap[record.GetString("source_key")] = struct{}{}
	}
	missing := 0
	for key := range scopeMap {
		if _, ok := existingMap[key]; !ok {
			missing++
		}
	}
	return missing
}

func upsertTargetCategoryNode(app core.App, node miniappmodel.CategoryNode, categoryPath string, parentKey string, assetBaseURL string) (bool, bool, error) {
	var changed bool
	created, err := upsertByFilter(app, CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": node.Key}, func(record *core.Record, created bool) error {
		before := targetCategoryRecordSignature(record)
		record.Set("source_key", node.Key)
		record.Set("label", node.Label)
		record.Set("path_name", node.PathName)
		record.Set("category_path", categoryPath)
		record.Set("parent_key", parentKey)
		record.Set("image_url", sanitizeURLWithBase(node.ImageURL, assetBaseURL))
		record.Set("depth", node.Depth)
		record.Set("sort", node.Sort)
		record.Set("has_children", node.HasChildren)
		if err := setJSON(record, "source_payload", node); err != nil {
			return err
		}
		changed = before != targetCategoryRecordSignature(record)
		if created {
			changed = false
		}
		return nil
	})
	return created, changed, err
}

func targetCategoryRecordSignature(record *core.Record) string {
	return strings.Join([]string{
		record.GetString("label"),
		record.GetString("path_name"),
		record.GetString("category_path"),
		record.GetString("parent_key"),
		fmt.Sprintf("%d", record.GetInt("depth")),
		fmt.Sprintf("%d", record.GetInt("sort")),
		fmt.Sprintf("%t", record.GetBool("has_children")),
		record.GetString("image_url"),
	}, "|")
}

func (s *Service) createTargetSyncRun(app core.App, job TargetSyncJob, actor TargetSyncActor) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionTargetSyncRuns)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now().Format(time.RFC3339)
	record := core.NewRecord(collection)
	record.Set("job_key", job.JobKey)
	record.Set("job_name", job.Name)
	record.Set("entity_type", normalizeTargetSyncEntity(job.EntityType))
	record.Set("scope_type", job.ScopeType)
	record.Set("scope_key", job.ScopeKey)
	record.Set("scope_label", job.ScopeLabel)
	record.Set("status", TargetSyncStatusRunning)
	record.Set("source_mode", job.SourceMode)
	record.Set("started_at", startedAt)
	record.Set("finished_at", "")
	record.Set("triggered_by_email", strings.TrimSpace(actor.Email))
	record.Set("triggered_by_name", strings.TrimSpace(actor.Name))
	record.Set("created_count", 0)
	record.Set("updated_count", 0)
	record.Set("unchanged_count", 0)
	record.Set("missing_count", 0)
	record.Set("scoped_node_count", 0)
	record.Set("progress_total", 0)
	record.Set("progress_done", 0)
	record.Set("current_stage", "queued")
	record.Set("current_item", "")
	record.Set("last_progress_at", startedAt)
	record.Set("error_message", "")
	if err := setJSON(record, "summary_json", map[string]any{
		"createdCount":    0,
		"updatedCount":    0,
		"unchangedCount":  0,
		"missingCount":    0,
		"scopedNodeCount": 0,
	}); err != nil {
		return nil, err
	}
	if err := setJSON(record, "details_json", []TargetSyncChangeItem{}); err != nil {
		return nil, err
	}
	if err := setJSON(record, "progress_logs_json", []TargetSyncProgressLog{{
		Time:    startedAt,
		Stage:   "queued",
		Level:   "info",
		Message: "任务已创建，等待执行。",
	}}); err != nil {
		return nil, err
	}
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, s.markTargetSyncJobRunning(app, job.JobKey, job.SourceMode)
}

func (s *Service) finalizeTargetSyncRun(app core.App, record *core.Record, result targetCategorySyncResult) (TargetSyncRun, error) {
	now := time.Now().Format(time.RFC3339)
	record.Set("job_key", result.jobKey)
	record.Set("job_name", result.jobName)
	record.Set("entity_type", normalizeTargetSyncEntity(result.entityType))
	record.Set("scope_type", result.scopeType)
	record.Set("scope_key", result.scopeKey)
	record.Set("scope_label", result.scopeLabel)
	record.Set("status", result.status)
	record.Set("source_mode", result.sourceMode)
	record.Set("finished_at", now)
	record.Set("created_count", result.createdCount)
	record.Set("updated_count", result.updatedCount)
	record.Set("unchanged_count", result.unchangedCount)
	record.Set("missing_count", result.missingCount)
	record.Set("scoped_node_count", result.scopedNodeCount)
	record.Set("progress_total", targetSyncMaxInt(record.GetInt("progress_total"), result.scopedNodeCount))
	record.Set("progress_done", targetSyncMaxInt(record.GetInt("progress_done"), result.scopedNodeCount))
	record.Set("current_stage", "completed")
	record.Set("current_item", "")
	record.Set("last_progress_at", now)
	record.Set("error_message", result.errorMessage)
	if err := setJSON(record, "summary_json", map[string]any{
		"createdCount":    result.createdCount,
		"updatedCount":    result.updatedCount,
		"unchangedCount":  result.unchangedCount,
		"missingCount":    result.missingCount,
		"scopedNodeCount": result.scopedNodeCount,
	}); err != nil {
		return TargetSyncRun{}, err
	}
	if err := setJSON(record, "details_json", result.details); err != nil {
		return TargetSyncRun{}, err
	}
	tracker := newTargetSyncProgressTracker(app, record)
	if result.status == TargetSyncStatusFailed {
		_ = tracker.addLog("completed", "error", targetSyncCompletionErrorMessage(result))
	} else {
		_ = tracker.addLog("completed", "info", targetSyncCompletionSuccessMessage(result))
	}
	if err := app.Save(record); err != nil {
		return TargetSyncRun{}, err
	}
	return targetSyncRunFromRecord(record), nil
}

func (s *Service) updateTargetSyncJobStatus(app core.App, jobKey string, result targetCategorySyncResult) error {
	record, err := app.FindFirstRecordByFilter(CollectionTargetSyncJobs, "job_key = {:job_key}", dbx.Params{"job_key": jobKey})
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	record.Set("status", result.status)
	record.Set("source_mode", result.sourceMode)
	record.Set("last_run_at", now)
	record.Set("last_error", result.errorMessage)
	if result.status == TargetSyncStatusSuccess || result.status == TargetSyncStatusPartial {
		record.Set("last_success_at", now)
	}
	return app.Save(record)
}

func (s *Service) markTargetSyncJobRunning(app core.App, jobKey string, sourceMode string) error {
	record, err := app.FindFirstRecordByFilter(CollectionTargetSyncJobs, "job_key = {:job_key}", dbx.Params{"job_key": jobKey})
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	record.Set("status", TargetSyncStatusRunning)
	record.Set("source_mode", strings.TrimSpace(sourceMode))
	record.Set("last_run_at", now)
	record.Set("last_error", "")
	return app.Save(record)
}

func (s *Service) activeTargetSyncRun(jobKey string) (string, bool) {
	s.targetSyncMu.Lock()
	defer s.targetSyncMu.Unlock()
	runID, ok := s.activeTargetSyncs[jobKey]
	return runID, ok
}

func (s *Service) setActiveTargetSyncRun(jobKey string, runID string) {
	s.targetSyncMu.Lock()
	defer s.targetSyncMu.Unlock()
	s.activeTargetSyncs[jobKey] = runID
}

func (s *Service) clearActiveTargetSyncRun(jobKey string, runID string) {
	s.targetSyncMu.Lock()
	defer s.targetSyncMu.Unlock()
	if current, ok := s.activeTargetSyncs[jobKey]; ok && current == runID {
		delete(s.activeTargetSyncs, jobKey)
	}
}

func (s *Service) currentActiveTargetSyncRun(app core.App, jobKey string) (TargetSyncRun, bool, error) {
	existingID, ok := s.activeTargetSyncRun(jobKey)
	if !ok {
		return TargetSyncRun{}, false, nil
	}
	existing, err := s.GetTargetSyncRun(app, existingID)
	if err != nil {
		s.clearActiveTargetSyncRun(jobKey, existingID)
		return TargetSyncRun{}, false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(existing.Status), TargetSyncStatusRunning) {
		s.clearActiveTargetSyncRun(jobKey, existingID)
		return TargetSyncRun{}, false, nil
	}
	return existing, true, nil
}

func newTargetSyncProgressTracker(app core.App, record *core.Record) *targetSyncProgressTracker {
	return &targetSyncProgressTracker{
		app:    app,
		record: record,
		logs:   decodeTargetSyncProgressLogs(record.GetString("progress_logs_json")),
	}
}

func (t *targetSyncProgressTracker) setStage(stage string, label string, total int) error {
	now := time.Now().Format(time.RFC3339)
	t.record.Set("current_stage", strings.TrimSpace(stage))
	t.record.Set("current_item", strings.TrimSpace(label))
	t.record.Set("last_progress_at", now)
	if total >= 0 {
		t.record.Set("progress_total", total)
		if t.record.GetInt("progress_done") > total && total > 0 {
			t.record.Set("progress_done", total)
		}
	}
	return t.save()
}

func (t *targetSyncProgressTracker) step(currentItem string) error {
	now := time.Now().Format(time.RFC3339)
	done := t.record.GetInt("progress_done") + 1
	total := t.record.GetInt("progress_total")
	if total > 0 && done > total {
		done = total
	}
	t.record.Set("progress_done", done)
	t.record.Set("current_item", strings.TrimSpace(currentItem))
	t.record.Set("last_progress_at", now)
	return t.save()
}

func (t *targetSyncProgressTracker) addLog(stage string, level string, message string) error {
	entry := TargetSyncProgressLog{
		Time:    time.Now().Format(time.RFC3339),
		Stage:   strings.TrimSpace(stage),
		Level:   strings.TrimSpace(level),
		Message: strings.TrimSpace(message),
	}
	if entry.Stage == "" {
		entry.Stage = t.record.GetString("current_stage")
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	if entry.Message == "" {
		return nil
	}
	t.logs = append(t.logs, entry)
	if len(t.logs) > 80 {
		t.logs = t.logs[len(t.logs)-80:]
	}
	t.record.Set("last_progress_at", entry.Time)
	return t.save()
}

func (t *targetSyncProgressTracker) save() error {
	if err := setJSON(t.record, "progress_logs_json", t.logs); err != nil {
		return err
	}
	return t.app.Save(t.record)
}

func (s *Service) listTargetSyncJobs(app core.App, limit int) ([]TargetSyncJob, error) {
	if limit <= 0 {
		limit = 20
	}
	collection, err := app.FindCollectionByNameOrId(CollectionTargetSyncJobs)
	if err != nil {
		return []TargetSyncJob{}, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "updated")
	records, err := app.FindRecordsByFilter(CollectionTargetSyncJobs, "", sortExpr, limit, 0, nil)
	if err != nil {
		return nil, err
	}
	items := make([]TargetSyncJob, 0, len(records))
	for _, record := range records {
		items = append(items, targetSyncJobFromRecord(record))
	}
	return items, nil
}

func (s *Service) listTargetSyncRuns(app core.App, limit int) ([]TargetSyncRun, error) {
	if limit <= 0 {
		limit = 12
	}
	collection, err := app.FindCollectionByNameOrId(CollectionTargetSyncRuns)
	if err != nil {
		return []TargetSyncRun{}, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "created")
	records, err := app.FindRecordsByFilter(CollectionTargetSyncRuns, "", sortExpr, limit, 0, nil)
	if err != nil {
		return nil, err
	}
	items := make([]TargetSyncRun, 0, len(records))
	for _, record := range records {
		items = append(items, targetSyncRunFromRecord(record))
	}
	return items, nil
}

func (s *Service) GetTargetSyncRun(app core.App, id string) (TargetSyncRun, error) {
	if err := s.reconcileStaleTargetSyncRuns(app); err != nil {
		return TargetSyncRun{}, err
	}
	record, err := app.FindRecordById(CollectionTargetSyncRuns, id)
	if err != nil {
		return TargetSyncRun{}, err
	}
	return targetSyncRunFromRecord(record), nil
}

func (s *Service) RetryFailedTargetSyncBranches(
	app core.App,
	sourceLoader targetSyncDatasetLoader,
	productLoader targetSyncSectionProductLoader,
	runID string,
	actor TargetSyncActor,
) ([]TargetSyncRun, []string, error) {
	run, err := s.GetTargetSyncRun(app, runID)
	if err != nil {
		return nil, nil, err
	}
	entityType := normalizeTargetSyncEntity(run.EntityType)
	if entityType != TargetSyncEntityProducts && entityType != TargetSyncEntityCategorySources {
		return nil, nil, fmt.Errorf("当前仅支持重跑分类商品来源或商品抓取失败分支")
	}
	branches := failedTargetSyncBranchItems(run, entityType)
	if len(branches) == 0 {
		return nil, nil, fmt.Errorf("当前运行记录没有可重跑的失败分支")
	}

	started := make([]TargetSyncRun, 0, len(branches))
	warnings := make([]string, 0)
	for _, branch := range branches {
		runItem, startErr := s.StartTargetSyncAsync(app, sourceLoader, productLoader, entityType, branch.TargetKey, targetSyncFirstNonEmpty(branch.Label, branch.TargetKey), actor)
		if startErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s：%v", targetSyncFirstNonEmpty(branch.Label, branch.TargetKey), startErr))
			continue
		}
		started = append(started, runItem)
	}
	if len(started) == 0 && len(warnings) > 0 {
		return nil, warnings, fmt.Errorf("失败分支重跑未成功启动")
	}
	return started, warnings, nil
}

func failedTargetSyncBranchItems(run TargetSyncRun, entityType string) []TargetSyncChangeItem {
	items := make([]TargetSyncChangeItem, 0)
	seen := make(map[string]struct{})
	for _, item := range run.Details {
		if !strings.EqualFold(strings.TrimSpace(item.ChangeType), "failed") {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(item.TargetType), normalizeTargetSyncEntity(entityType)) {
			continue
		}
		key := strings.TrimSpace(item.TargetKey)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	return items
}

func (s *Service) reconcileStaleTargetSyncRuns(app core.App) error {
	records, err := app.FindRecordsByFilter(
		CollectionTargetSyncRuns,
		"status = {:status}",
		"",
		200,
		0,
		dbx.Params{"status": TargetSyncStatusRunning},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		jobKey := strings.TrimSpace(record.GetString("job_key"))
		if activeRunID, ok := s.activeTargetSyncRun(jobKey); ok && activeRunID == record.Id {
			continue
		}
		if err := s.markTargetSyncRunStale(app, record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) markTargetSyncRunStale(app core.App, record *core.Record) error {
	now := time.Now().Format(time.RFC3339)
	logs := decodeTargetSyncProgressLogs(record.GetString("progress_logs_json"))
	if completed, created, updated, unchanged := targetSyncCompletedFromLogs(logs, record); completed {
		message := "任务已完成，但结束状态未及时回写，系统已自动补写为完成。"
		record.Set("status", TargetSyncStatusSuccess)
		record.Set("error_message", "")
		record.Set("finished_at", now)
		record.Set("last_progress_at", now)
		record.Set("created_count", created)
		record.Set("updated_count", updated)
		record.Set("unchanged_count", unchanged)
		record.Set("current_stage", "completed")
		record.Set("current_item", "")
		logs = append(logs, TargetSyncProgressLog{
			Time:    now,
			Stage:   "completed",
			Level:   "info",
			Message: message,
		})
		if len(logs) > 80 {
			logs = logs[len(logs)-80:]
		}
		if err := setJSON(record, "progress_logs_json", logs); err != nil {
			return err
		}
		if err := app.Save(record); err != nil {
			return err
		}
		return s.updateTargetSyncJobStatus(app, record.GetString("job_key"), targetCategorySyncResult{
			status:         TargetSyncStatusSuccess,
			sourceMode:     record.GetString("source_mode"),
			createdCount:   created,
			updatedCount:   updated,
			unchangedCount: unchanged,
		})
	}

	message := "任务已失联，通常是服务重启或 raw 读取超时后未正常回写，系统已自动结束该任务。"
	record.Set("status", TargetSyncStatusFailed)
	record.Set("error_message", message)
	record.Set("finished_at", now)
	record.Set("last_progress_at", now)
	logs = append(logs, TargetSyncProgressLog{
		Time:    now,
		Stage:   targetSyncFirstNonEmpty(record.GetString("current_stage"), "stale_recovery"),
		Level:   "error",
		Message: message,
	})
	if len(logs) > 80 {
		logs = logs[len(logs)-80:]
	}
	if err := setJSON(record, "progress_logs_json", logs); err != nil {
		return err
	}
	if err := app.Save(record); err != nil {
		return err
	}
	return s.updateTargetSyncJobStatus(app, record.GetString("job_key"), targetCategorySyncResult{
		status:       TargetSyncStatusFailed,
		sourceMode:   record.GetString("source_mode"),
		errorMessage: message,
	})
}

func (s *Service) targetSyncStaleAfter() time.Duration {
	timeout := s.cfg.MiniApp.SourceTimeout * 3
	if timeout < 45*time.Second {
		timeout = 45 * time.Second
	}
	return timeout
}

func (s *Service) targetSyncDatasetLoadTimeout(entityType string) time.Duration {
	timeout := s.cfg.MiniApp.SourceTimeout + 10*time.Second
	switch normalizeTargetSyncEntity(entityType) {
	case TargetSyncEntityProducts, TargetSyncEntityAssets, TargetSyncEntityCategoryTree, TargetSyncEntityCategorySources, TargetSyncEntityCategoryRebuild:
		if timeout < 90*time.Second {
			timeout = 90 * time.Second
		}
	default:
		if timeout < 30*time.Second {
			timeout = 30 * time.Second
		}
	}
	if timeout > 4*time.Minute {
		timeout = 4 * time.Minute
	}
	return timeout
}

func targetSyncCompletedFromLogs(logs []TargetSyncProgressLog, record *core.Record) (bool, int, int, int) {
	total := record.GetInt("progress_total")
	done := record.GetInt("progress_done")
	if total <= 0 || done < total {
		return false, 0, 0, 0
	}
	for idx := len(logs) - 1; idx >= 0; idx-- {
		message := strings.TrimSpace(logs[idx].Message)
		if message == "" {
			continue
		}
		created, updated, unchanged, ok := parseTargetSyncCompletionCounts(message)
		if ok {
			return true, created, updated, unchanged
		}
	}
	return true, record.GetInt("created_count"), record.GetInt("updated_count"), record.GetInt("unchanged_count")
}

func parseTargetSyncCompletionCounts(message string) (int, int, int, bool) {
	patterns := []string{
		"抓取入库完成：新增 %d，更新 %d，未变化 %d。",
		"抓分类树完成：新增 %d，更新 %d，未变化 %d。",
		"刷新分类商品来源完成：新增 %d，更新 %d，未变化 %d。",
		"按已保存分类来源抓商品规格完成：新增 %d，更新 %d，未变化 %d。",
		"按当前源站结果抓图片完成：新增 %d，更新 %d，未变化 %d。",
		"重建分类商品归属完成：新增 %d，更新 %d，未变化 %d。",
	}
	for _, pattern := range patterns {
		var created, updated, unchanged int
		if _, err := fmt.Sscanf(message, pattern, &created, &updated, &unchanged); err == nil {
			return created, updated, unchanged, true
		}
	}
	return 0, 0, 0, false
}

func parseTargetSyncTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func targetSyncJobFromRecord(record *core.Record) TargetSyncJob {
	return TargetSyncJob{
		ID:            record.Id,
		JobKey:        record.GetString("job_key"),
		Name:          record.GetString("name"),
		EntityType:    record.GetString("entity_type"),
		ScopeType:     record.GetString("scope_type"),
		ScopeKey:      record.GetString("scope_key"),
		ScopeLabel:    record.GetString("scope_label"),
		Status:        record.GetString("status"),
		SourceMode:    record.GetString("source_mode"),
		LastRunAt:     record.GetString("last_run_at"),
		LastSuccessAt: record.GetString("last_success_at"),
		LastError:     record.GetString("last_error"),
	}
}

func targetSyncRunFromRecord(record *core.Record) TargetSyncRun {
	return TargetSyncRun{
		ID:               record.Id,
		JobKey:           record.GetString("job_key"),
		JobName:          record.GetString("job_name"),
		EntityType:       record.GetString("entity_type"),
		ScopeType:        record.GetString("scope_type"),
		ScopeKey:         record.GetString("scope_key"),
		ScopeLabel:       record.GetString("scope_label"),
		Status:           record.GetString("status"),
		SourceMode:       record.GetString("source_mode"),
		StartedAt:        record.GetString("started_at"),
		FinishedAt:       record.GetString("finished_at"),
		TriggeredByEmail: record.GetString("triggered_by_email"),
		TriggeredByName:  record.GetString("triggered_by_name"),
		CreatedCount:     record.GetInt("created_count"),
		UpdatedCount:     record.GetInt("updated_count"),
		UnchangedCount:   record.GetInt("unchanged_count"),
		MissingCount:     record.GetInt("missing_count"),
		ScopedNodeCount:  record.GetInt("scoped_node_count"),
		ProgressTotal:    record.GetInt("progress_total"),
		ProgressDone:     record.GetInt("progress_done"),
		CurrentStage:     record.GetString("current_stage"),
		CurrentItem:      record.GetString("current_item"),
		LastProgressAt:   record.GetString("last_progress_at"),
		ErrorMessage:     record.GetString("error_message"),
		Details:          decodeTargetSyncDetails(record.GetString("details_json")),
		Logs:             decodeTargetSyncProgressLogs(record.GetString("progress_logs_json")),
	}
}

func decodeTargetSyncDetails(raw string) []TargetSyncChangeItem {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []TargetSyncChangeItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func decodeTargetSyncProgressLogs(raw string) []TargetSyncProgressLog {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []TargetSyncProgressLog
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func appendTargetSyncDetail(items []TargetSyncChangeItem, item TargetSyncChangeItem) []TargetSyncChangeItem {
	if len(items) >= 40 {
		return items
	}
	return append(items, item)
}

func safeCollectionSortExpr(collection *core.Collection, preferred string) string {
	if collection.Fields.GetByName(preferred) != nil {
		return "-" + preferred
	}
	if collection.Fields.GetByName("created") != nil {
		return "-created"
	}
	return "-id"
}

func normalizeTargetSyncEntity(entityType string) string {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case TargetSyncEntityCategorySources:
		return TargetSyncEntityCategorySources
	case TargetSyncEntityCategoryRebuild:
		return TargetSyncEntityCategoryRebuild
	case TargetSyncEntityProducts:
		return TargetSyncEntityProducts
	case TargetSyncEntityAssets:
		return TargetSyncEntityAssets
	default:
		return TargetSyncEntityCategoryTree
	}
}

func targetSyncEntityLabel(entityType string) string {
	switch normalizeTargetSyncEntity(entityType) {
	case TargetSyncEntityCategorySources:
		return "分类商品来源"
	case TargetSyncEntityCategoryRebuild:
		return "分类商品归属重建"
	case TargetSyncEntityProducts:
		return "商品与规格"
	case TargetSyncEntityAssets:
		return "图片资产"
	default:
		return "分类树"
	}
}

func targetSyncEntityActionLabel(entityType string) string {
	switch normalizeTargetSyncEntity(entityType) {
	case TargetSyncEntityCategorySources:
		return "刷新分类商品来源"
	case TargetSyncEntityCategoryRebuild:
		return "重建分类商品归属"
	case TargetSyncEntityProducts:
		return "按已保存来源抓商品规格"
	case TargetSyncEntityAssets:
		return "按已保存商品抓图片"
	default:
		return "抓分类树"
	}
}

func targetSyncCompletionSuccessMessage(result targetCategorySyncResult) string {
	switch normalizeTargetSyncEntity(result.entityType) {
	case TargetSyncEntityCategorySources:
		return fmt.Sprintf("刷新分类商品来源完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount)
	case TargetSyncEntityCategoryRebuild:
		return fmt.Sprintf("重建分类商品归属完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount)
	case TargetSyncEntityProducts:
		return fmt.Sprintf("按已保存来源抓商品规格完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount)
	case TargetSyncEntityAssets:
		return fmt.Sprintf("按已保存商品抓图片完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount)
	default:
		return fmt.Sprintf("抓分类树完成：新增 %d，更新 %d，未变化 %d。", result.createdCount, result.updatedCount, result.unchangedCount)
	}
}

func targetSyncCompletionErrorMessage(result targetCategorySyncResult) string {
	if msg := strings.TrimSpace(result.errorMessage); msg != "" {
		return msg
	}
	switch normalizeTargetSyncEntity(result.entityType) {
	case TargetSyncEntityCategorySources:
		return "刷新分类商品来源失败。"
	case TargetSyncEntityCategoryRebuild:
		return "重建分类商品归属失败。"
	case TargetSyncEntityProducts:
		return "按已保存来源抓商品规格失败。"
	case TargetSyncEntityAssets:
		return "按已保存商品抓图片失败。"
	default:
		return "抓分类树失败。"
	}
}

func targetScopeType(scopeKey string) string {
	if strings.TrimSpace(scopeKey) == "" {
		return TargetSyncScopeAll
	}
	return TargetSyncScopeTopLevel
}

func targetScopeLabel(dataset miniappmodel.Dataset, scopeKey string) string {
	if strings.TrimSpace(scopeKey) == "" {
		return "当前源站结果"
	}
	if node := findCategoryNode(dataset.CategoryPage.Tree, scopeKey); node != nil {
		return node.Label
	}
	return strings.TrimSpace(scopeKey)
}

func defaultTargetSyncStatus(status string, created bool) string {
	if strings.TrimSpace(status) != "" {
		return status
	}
	if created {
		return TargetSyncStatusPending
	}
	return TargetSyncStatusSuccess
}

func resolveTargetSyncScopeLabel(dataset *miniappmodel.Dataset, scopeKey string, scopeLabel string) string {
	if strings.TrimSpace(scopeKey) == "" {
		return "当前源站结果"
	}
	if strings.TrimSpace(scopeLabel) != "" {
		return strings.TrimSpace(scopeLabel)
	}
	if dataset != nil {
		if node := findCategoryNode(dataset.CategoryPage.Tree, scopeKey); node != nil {
			return node.Label
		}
	}
	return strings.TrimSpace(scopeKey)
}

func targetSyncFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func targetSyncMaxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func flattenCategoryNodes(nodes []miniappmodel.CategoryNode) []miniappmodel.CategoryNode {
	items := make([]miniappmodel.CategoryNode, 0)
	var walk func(list []miniappmodel.CategoryNode)
	walk = func(list []miniappmodel.CategoryNode) {
		for _, node := range list {
			items = append(items, node)
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(nodes)
	return items
}

func filteredTargetProducts(dataset miniappmodel.Dataset, scopeKey string) []miniappmodel.ProductPage {
	if strings.TrimSpace(scopeKey) == "" {
		return append([]miniappmodel.ProductPage(nil), dataset.ProductPage.Products...)
	}
	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	_, parentByKey, hasChildrenByKey := buildCategoryTreeMeta(dataset.CategoryPage.Tree)
	items := make([]miniappmodel.ProductPage, 0)
	for _, product := range dataset.ProductPage.Products {
		_, _, categoryKeys, _, _ := productCategoryInfo(product, sections, parentByKey, hasChildrenByKey)
		if slices.Contains(categoryKeys, strings.TrimSpace(scopeKey)) {
			items = append(items, product)
		}
	}
	return items
}

func filteredTargetAssets(dataset miniappmodel.Dataset, scopeKey string) []sourceAssetItem {
	products := filteredTargetProducts(dataset, scopeKey)
	items := make([]sourceAssetItem, 0)
	for _, product := range products {
		items = append(items, collectProductAssets(product)...)
	}
	return items
}

func countTargetMultiUnitProducts(products []miniappmodel.ProductPage) int {
	total := 0
	for _, product := range products {
		if len(product.Pricing.UnitOptions) > 1 {
			total++
		}
	}
	return total
}

func buildCategoryTopLevelLookup(nodes []miniappmodel.CategoryNode) map[string]string {
	lookup := make(map[string]string)
	var walk func(list []miniappmodel.CategoryNode, topLevelKey string)
	walk = func(list []miniappmodel.CategoryNode, topLevelKey string) {
		for _, node := range list {
			currentTop := topLevelKey
			if currentTop == "" {
				currentTop = node.Key
			}
			lookup[node.Key] = currentTop
			if len(node.Children) > 0 {
				walk(node.Children, currentTop)
			}
		}
	}
	walk(nodes, "")
	return lookup
}

func buildCategoryTreeMeta(nodes []miniappmodel.CategoryNode) (map[string]string, map[string]string, map[string]bool) {
	paths := make(map[string]string)
	parents := make(map[string]string)
	hasChildren := make(map[string]bool)
	var walk func(list []miniappmodel.CategoryNode, parentPath string, parentKey string)
	walk = func(list []miniappmodel.CategoryNode, parentPath string, parentKey string) {
		for _, node := range list {
			path := strings.TrimSpace(node.Label)
			if parentPath != "" {
				path = parentPath + " / " + path
			}
			paths[node.Key] = path
			parents[node.Key] = parentKey
			if len(node.Children) > 0 {
				hasChildren[node.Key] = true
				walk(node.Children, path, node.Key)
			}
		}
	}
	walk(nodes, "", "")
	return paths, parents, hasChildren
}

func findCategoryNode(nodes []miniappmodel.CategoryNode, key string) *miniappmodel.CategoryNode {
	for _, node := range nodes {
		if strings.EqualFold(node.Key, key) {
			copyNode := node
			return &copyNode
		}
		if len(node.Children) > 0 {
			found := findCategoryNode(node.Children, key)
			if found != nil {
				return found
			}
		}
	}
	return nil
}

func targetCategoryDiff(tree []miniappmodel.CategoryNode, actual []*core.Record) ([]TargetCategoryDiffItem, int, int, int) {
	expected := flattenCategoryNodes(tree)
	expectedMap := make(map[string]miniappmodel.CategoryNode, len(expected))
	pathMap, _, _ := buildCategoryTreeMeta(tree)
	for _, node := range expected {
		expectedMap[node.Key] = node
	}
	actualMap := make(map[string]*core.Record, len(actual))
	for _, record := range actual {
		actualMap[record.GetString("source_key")] = record
	}

	items := make([]TargetCategoryDiffItem, 0)
	diffNew := 0
	diffChanged := 0
	diffMissing := 0
	for key, node := range expectedMap {
		record, ok := actualMap[key]
		if !ok {
			diffNew++
			items = append(items, TargetCategoryDiffItem{
				SourceKey:    key,
				Label:        node.Label,
				CategoryPath: pathMap[key],
				DiffType:     "new",
			})
			continue
		}
		if strings.TrimSpace(record.GetString("label")) != strings.TrimSpace(node.Label) ||
			strings.TrimSpace(record.GetString("path_name")) != strings.TrimSpace(node.PathName) ||
			record.GetInt("depth") != node.Depth ||
			record.GetInt("sort") != node.Sort ||
			record.GetBool("has_children") != node.HasChildren ||
			strings.TrimSpace(record.GetString("image_url")) != strings.TrimSpace(node.ImageURL) ||
			strings.TrimSpace(record.GetString("category_path")) != strings.TrimSpace(pathMap[key]) {
			diffChanged++
			items = append(items, TargetCategoryDiffItem{
				SourceKey:    key,
				Label:        node.Label,
				CategoryPath: pathMap[key],
				DiffType:     "changed",
			})
		}
	}
	for key, record := range actualMap {
		if _, ok := expectedMap[key]; ok {
			continue
		}
		diffMissing++
		items = append(items, TargetCategoryDiffItem{
			SourceKey:    key,
			Label:        record.GetString("label"),
			CategoryPath: record.GetString("category_path"),
			DiffType:     "missing",
		})
	}
	if len(items) > 12 {
		items = items[:12]
	}
	return items, diffNew, diffChanged, diffMissing
}

func targetProductAssetDiffs(dataset miniappmodel.Dataset, sourceProducts []*core.Record, sourceAssets []*core.Record) (int, int, int, int) {
	productByID := make(map[string]*core.Record, len(sourceProducts))
	for _, record := range sourceProducts {
		productByID[record.GetString("product_id")] = record
	}
	productDiffNew := 0
	productDiffChanged := 0
	for _, product := range dataset.ProductPage.Products {
		record, ok := productByID[product.ID]
		if !ok {
			productDiffNew++
			continue
		}
		if sourceProductSignature(record) != expectedProductSignature(product, dataset) {
			productDiffChanged++
		}
	}

	assetByKey := make(map[string]*core.Record, len(sourceAssets))
	for _, record := range sourceAssets {
		assetByKey[record.GetString("asset_key")] = record
	}
	assetDiffNew := 0
	assetDiffChanged := 0
	for _, asset := range filteredTargetAssets(dataset, "") {
		record, ok := assetByKey[asset.Key]
		if !ok {
			assetDiffNew++
			continue
		}
		if sourceAssetSignature(record) != expectedAssetSignature(asset) {
			assetDiffChanged++
		}
	}
	return productDiffNew, productDiffChanged, assetDiffNew, assetDiffChanged
}

func sourceProductSignature(record *core.Record) string {
	if record == nil {
		return ""
	}
	return strings.Join([]string{
		record.GetString("product_id"),
		record.GetString("name"),
		record.GetString("cover_url"),
		record.GetString("default_unit"),
		record.GetString("category_key"),
		record.GetString("category_path"),
		record.GetString("category_keys_json"),
		record.GetString("observed_category_keys_json"),
		record.GetString("observed_category_paths_json"),
		record.GetString("source_type"),
		fmt.Sprintf("%d", record.GetInt("unit_count")),
		fmt.Sprintf("%t", record.GetBool("has_multi_unit")),
		fmt.Sprintf("%.2f", record.GetFloat("default_price")),
		fmt.Sprintf("%d", record.GetInt("asset_count")),
	}, "|")
}

func expectedProductSignature(product miniappmodel.ProductPage, dataset miniappmodel.Dataset) string {
	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	_, parentByKey, hasChildrenByKey := buildCategoryTreeMeta(dataset.CategoryPage.Tree)
	categoryKey, categoryPath, categoryKeys, observedCategoryKeys, observedCategoryPaths := productCategoryInfo(product, sections, parentByKey, hasChildrenByKey)
	return strings.Join([]string{
		product.ID,
		product.Summary.Name,
		product.Summary.Cover,
		product.Summary.DefaultUnit,
		categoryKey,
		categoryPath,
		strings.Join(categoryKeys, ","),
		strings.Join(observedCategoryKeys, ","),
		strings.Join(observedCategoryPaths, "|"),
		product.SourceType,
		fmt.Sprintf("%d", len(product.Pricing.UnitOptions)),
		fmt.Sprintf("%t", len(product.Pricing.UnitOptions) > 1),
		fmt.Sprintf("%.2f", product.Pricing.DefaultPrice),
		fmt.Sprintf("%d", countProductAssets(product)),
	}, "|")
}

func sourceAssetSignature(record *core.Record) string {
	if record == nil {
		return ""
	}
	return strings.Join([]string{
		record.GetString("asset_key"),
		record.GetString("product_id"),
		record.GetString("source_url"),
		record.GetString("asset_role"),
		fmt.Sprintf("%d", record.GetInt("sort")),
	}, "|")
}

func expectedAssetSignature(asset sourceAssetItem) string {
	return strings.Join([]string{
		asset.Key,
		asset.URL,
		asset.Role,
		fmt.Sprintf("%d", asset.Sort),
	}, "|")
}

func upsertTargetAssetItem(app core.App, product miniappmodel.ProductPage, asset sourceAssetItem, assetBaseURL string) (bool, error) {
	return upsertByFilter(app, CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key}, func(record *core.Record, created bool) error {
		record.Set("asset_key", asset.Key)
		record.Set("product_id", product.ID)
		record.Set("spu_id", product.SpuID)
		record.Set("sku_id", product.SkuID)
		record.Set("name", product.Summary.Name)
		record.Set("source_url", sanitizeURLWithBase(asset.URL, assetBaseURL))
		record.Set("asset_role", asset.Role)
		record.Set("sort", asset.Sort)
		if created && strings.TrimSpace(record.GetString("original_image_status")) == "" && strings.TrimSpace(record.GetString("source_url")) != "" {
			record.Set("original_image_status", OriginalImageStatusPending)
		}
		if created && strings.TrimSpace(record.GetString("image_processing_status")) == "" {
			record.Set("image_processing_status", ImageStatusPending)
		}
		return setJSON(record, "source_payload", asset.Payload)
	})
}

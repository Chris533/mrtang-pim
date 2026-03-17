package pim

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	miniappmodel "mrtang-pim/internal/miniapp/model"
)

const (
	CollectionTargetSyncJobs = "target_sync_jobs"
	CollectionTargetSyncRuns = "target_sync_runs"

	TargetSyncEntityCategoryTree = "category_tree"
	TargetSyncEntityProducts     = "products"
	TargetSyncEntityAssets       = "assets"

	TargetSyncScopeAll      = "all"
	TargetSyncScopeTopLevel = "top_level"

	TargetSyncStatusPending = "pending"
	TargetSyncStatusRunning = "running"
	TargetSyncStatusSuccess = "success"
	TargetSyncStatusPartial = "partial"
	TargetSyncStatusFailed  = "failed"
)

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
	ID               string                 `json:"id"`
	JobKey           string                 `json:"jobKey"`
	JobName          string                 `json:"jobName"`
	EntityType       string                 `json:"entityType"`
	ScopeType        string                 `json:"scopeType"`
	ScopeKey         string                 `json:"scopeKey"`
	ScopeLabel       string                 `json:"scopeLabel"`
	Status           string                 `json:"status"`
	SourceMode       string                 `json:"sourceMode"`
	StartedAt        string                 `json:"startedAt"`
	FinishedAt       string                 `json:"finishedAt"`
	TriggeredByEmail string                 `json:"triggeredByEmail"`
	TriggeredByName  string                 `json:"triggeredByName"`
	CreatedCount     int                    `json:"createdCount"`
	UpdatedCount     int                    `json:"updatedCount"`
	UnchangedCount   int                    `json:"unchangedCount"`
	MissingCount     int                    `json:"missingCount"`
	ScopedNodeCount  int                    `json:"scopedNodeCount"`
	ErrorMessage     string                 `json:"errorMessage"`
	Details          []TargetSyncChangeItem `json:"details"`
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
	SourceMode              string                   `json:"sourceMode"`
	JobCount                int                      `json:"jobCount"`
	RunCount                int                      `json:"runCount"`
	CategoryCount           int                      `json:"categoryCount"`
	SourceProductCount      int                      `json:"sourceProductCount"`
	SourceAssetCount        int                      `json:"sourceAssetCount"`
	ExpectedNodeCount       int                      `json:"expectedNodeCount"`
	ExpectedProductCount    int                      `json:"expectedProductCount"`
	ExpectedAssetCount      int                      `json:"expectedAssetCount"`
	ExpectedMultiUnitCount  int                      `json:"expectedMultiUnitCount"`
	TopLevelCount           int                      `json:"topLevelCount"`
	SourceImportedCount     int                      `json:"sourceImportedCount"`
	SourceApprovedCount     int                      `json:"sourceApprovedCount"`
	SourceAssetPendingCount int                      `json:"sourceAssetPendingCount"`
	SourceAssetFailedCount  int                      `json:"sourceAssetFailedCount"`
	DiffNewCount            int                      `json:"diffNewCount"`
	DiffChangedCount        int                      `json:"diffChangedCount"`
	DiffMissingCount        int                      `json:"diffMissingCount"`
	ProductDiffNewCount     int                      `json:"productDiffNewCount"`
	ProductDiffChangedCount int                      `json:"productDiffChangedCount"`
	AssetDiffNewCount       int                      `json:"assetDiffNewCount"`
	AssetDiffChangedCount   int                      `json:"assetDiffChangedCount"`
	Jobs                    []TargetSyncJob          `json:"jobs"`
	Runs                    []TargetSyncRun          `json:"runs"`
	ScopeOptions            []TargetSyncScopeOption  `json:"scopeOptions"`
	CategoryDiffs           []TargetCategoryDiffItem `json:"categoryDiffs"`
	CheckoutSources         []TargetCheckoutSource   `json:"checkoutSources"`
	RecentMiniappWrites     []TargetMiniappWrite     `json:"recentMiniappWrites"`
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

func (s *Service) TargetSyncSummary(_ context.Context, app core.App, dataset miniappmodel.Dataset) (TargetSyncSummary, error) {
	expectedNodes := flattenCategoryNodes(dataset.CategoryPage.Tree)
	sourceCategories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	sourceProducts, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	sourceAssets, err := app.FindAllRecords(CollectionSourceAssets)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	jobs, err := s.listTargetSyncJobs(app, 20)
	if err != nil {
		return TargetSyncSummary{}, err
	}
	runs, err := s.listTargetSyncRuns(app, 12)
	if err != nil {
		return TargetSyncSummary{}, err
	}

	diffItems, diffNew, diffChanged, diffMissing := targetCategoryDiff(dataset.CategoryPage.Tree, sourceCategories)
	productDiffNew, productDiffChanged, assetDiffNew, assetDiffChanged := targetProductAssetDiffs(dataset, sourceProducts, sourceAssets)
	scopeOptions := make([]TargetSyncScopeOption, 0, len(dataset.CategoryPage.Tree)+1)
	allProducts := filteredTargetProducts(dataset, "")
	allAssets := filteredTargetAssets(dataset, "")
	scopeOptions = append(scopeOptions, TargetSyncScopeOption{
		Key:          "",
		Label:        "全量分类树",
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

	return TargetSyncSummary{
		SourceMode:              dataset.Meta.Source,
		JobCount:                len(jobs),
		RunCount:                len(runs),
		CategoryCount:           len(sourceCategories),
		SourceProductCount:      len(sourceProducts),
		SourceAssetCount:        len(sourceAssets),
		ExpectedNodeCount:       len(expectedNodes),
		ExpectedProductCount:    len(allProducts),
		ExpectedAssetCount:      len(allAssets),
		ExpectedMultiUnitCount:  countTargetMultiUnitProducts(allProducts),
		TopLevelCount:           len(dataset.CategoryPage.Tree),
		SourceImportedCount:     importedCount,
		SourceApprovedCount:     approvedCount,
		SourceAssetPendingCount: assetPendingCount,
		SourceAssetFailedCount:  assetFailedCount,
		DiffNewCount:            diffNew,
		DiffChangedCount:        diffChanged,
		DiffMissingCount:        diffMissing,
		ProductDiffNewCount:     productDiffNew,
		ProductDiffChangedCount: productDiffChanged,
		AssetDiffNewCount:       assetDiffNew,
		AssetDiffChangedCount:   assetDiffChanged,
		Jobs:                    jobs,
		Runs:                    runs,
		ScopeOptions:            scopeOptions,
		CategoryDiffs:           diffItems,
		CheckoutSources:         targetCheckoutSources(dataset),
		RecentMiniappWrites:     targetMiniappWrites(app, 8),
	}, nil
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
	entityType = normalizeTargetSyncEntity(entityType)
	scopeType := TargetSyncScopeAll
	scopeLabel := "全量"
	jobKey := entityType + ":all"
	if key := strings.TrimSpace(scopeKey); key != "" {
		scopeType = TargetSyncScopeTopLevel
		scopeLabel = key
		jobKey = entityType + ":" + key
		if node := findCategoryNode(dataset.CategoryPage.Tree, key); node != nil {
			scopeLabel = node.Label
		}
	}
	name := targetSyncEntityLabel(entityType) + "同步"
	if scopeType == TargetSyncScopeTopLevel {
		name = name + " / " + scopeLabel
	}

	_, err := upsertByFilter(app, CollectionTargetSyncJobs, "job_key = {:job_key}", dbx.Params{"job_key": jobKey}, func(record *core.Record, created bool) error {
		record.Set("job_key", jobKey)
		record.Set("name", name)
		record.Set("entity_type", entityType)
		record.Set("scope_type", scopeType)
		record.Set("scope_key", strings.TrimSpace(scopeKey))
		record.Set("scope_label", scopeLabel)
		record.Set("status", defaultTargetSyncStatus(record.GetString("status"), created))
		record.Set("source_mode", dataset.Meta.Source)
		return setJSON(record, "config_json", map[string]any{
			"scopeKey":   strings.TrimSpace(scopeKey),
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

func (s *Service) RunTargetSync(ctx context.Context, app core.App, dataset miniappmodel.Dataset, entityType string, scopeKey string, actor TargetSyncActor) (TargetSyncRun, error) {
	entityType = normalizeTargetSyncEntity(entityType)
	job, err := s.EnsureTargetSyncJob(ctx, app, dataset, entityType, scopeKey)
	if err != nil {
		return TargetSyncRun{}, err
	}

	var result targetCategorySyncResult
	switch entityType {
	case TargetSyncEntityProducts:
		result, err = s.syncTargetProducts(ctx, app, dataset, job.ScopeKey)
	case TargetSyncEntityAssets:
		result, err = s.syncTargetAssets(ctx, app, dataset, job.ScopeKey)
	default:
		result, err = s.syncTargetCategories(ctx, app, dataset, job.ScopeKey)
	}
	if err != nil {
		result.status = TargetSyncStatusFailed
		result.errorMessage = err.Error()
	}
	result.jobKey = job.JobKey
	result.jobName = job.Name
	result.scopeType = job.ScopeType
	result.scopeKey = job.ScopeKey
	result.scopeLabel = job.ScopeLabel
	result.sourceMode = dataset.Meta.Source
	result.entityType = entityType

	runRecord, saveErr := s.saveTargetSyncRun(app, result, actor)
	if saveErr != nil {
		return TargetSyncRun{}, saveErr
	}
	if updateErr := s.updateTargetSyncJobStatus(app, job.JobKey, result); updateErr != nil {
		return TargetSyncRun{}, updateErr
	}
	if err != nil {
		return runRecord, err
	}
	return runRecord, nil
}

func (s *Service) syncTargetCategories(_ context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityCategoryTree,
		status:     TargetSyncStatusSuccess,
		sourceMode: dataset.Meta.Source,
	}
	nodes := dataset.CategoryPage.Tree
	scopeLabel := "全量分类树"
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
	scopeMap := make(map[string]miniappmodel.CategoryNode, len(scopedNodes))
	for _, node := range scopedNodes {
		scopeMap[node.Key] = node
	}

	categoryPathByKey, parentKeyByKey := buildCategoryTreeMeta(nodes)
	for _, node := range scopedNodes {
		expectedPath := categoryPathByKey[node.Key]
		created, changed, upsertErr := upsertTargetCategoryNode(app, node, expectedPath, parentKeyByKey[node.Key])
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

	return result, nil
}

func (s *Service) syncTargetProducts(ctx context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string) (targetCategorySyncResult, error) {
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

	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	for _, product := range products {
		categoryKey := firstCategoryKey(product.SourceSections, sections)
		categoryPath := ""
		if section, ok := sections[categoryKey]; ok {
			categoryPath = strings.TrimSpace(section.CategoryPath)
		}
		existing, _ := app.FindFirstRecordByFilter(CollectionSourceProducts, "product_id = {:product_id}", dbx.Params{"product_id": product.ID})
		before := sourceProductSignature(existing)
		created, err := s.upsertSourceProduct(ctx, app, product, categoryKey, categoryPath)
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
	return result, nil
}

func (s *Service) syncTargetAssets(_ context.Context, app core.App, dataset miniappmodel.Dataset, scopeKey string) (targetCategorySyncResult, error) {
	result := targetCategorySyncResult{
		entityType: TargetSyncEntityAssets,
		status:     TargetSyncStatusSuccess,
		sourceMode: dataset.Meta.Source,
	}
	products := filteredTargetProducts(dataset, scopeKey)
	result.scopeType = targetScopeType(scopeKey)
	result.scopeKey = strings.TrimSpace(scopeKey)
	result.scopeLabel = targetScopeLabel(dataset, scopeKey)
	for _, product := range products {
		assets := collectProductAssets(product)
		result.scopedNodeCount += len(assets)
		for _, asset := range assets {
			existing, _ := app.FindFirstRecordByFilter(CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key})
			before := sourceAssetSignature(existing)
			created, err := upsertTargetAssetItem(app, product, asset)
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
	return result, nil
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

func upsertTargetCategoryNode(app core.App, node miniappmodel.CategoryNode, categoryPath string, parentKey string) (bool, bool, error) {
	var changed bool
	created, err := upsertByFilter(app, CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": node.Key}, func(record *core.Record, created bool) error {
		before := targetCategoryRecordSignature(record)
		record.Set("source_key", node.Key)
		record.Set("label", node.Label)
		record.Set("path_name", node.PathName)
		record.Set("category_path", categoryPath)
		record.Set("parent_key", parentKey)
		record.Set("image_url", node.ImageURL)
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

func (s *Service) saveTargetSyncRun(app core.App, result targetCategorySyncResult, actor TargetSyncActor) (TargetSyncRun, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionTargetSyncRuns)
	if err != nil {
		return TargetSyncRun{}, err
	}
	startedAt := time.Now().Format(time.RFC3339)
	finishedAt := startedAt
	record := core.NewRecord(collection)
	record.Set("job_key", result.jobKey)
	record.Set("job_name", result.jobName)
	record.Set("entity_type", normalizeTargetSyncEntity(result.entityType))
	record.Set("scope_type", result.scopeType)
	record.Set("scope_key", result.scopeKey)
	record.Set("scope_label", result.scopeLabel)
	record.Set("status", result.status)
	record.Set("source_mode", result.sourceMode)
	record.Set("started_at", startedAt)
	record.Set("finished_at", finishedAt)
	record.Set("triggered_by_email", strings.TrimSpace(actor.Email))
	record.Set("triggered_by_name", strings.TrimSpace(actor.Name))
	record.Set("created_count", result.createdCount)
	record.Set("updated_count", result.updatedCount)
	record.Set("unchanged_count", result.unchangedCount)
	record.Set("missing_count", result.missingCount)
	record.Set("scoped_node_count", result.scopedNodeCount)
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
	record, err := app.FindRecordById(CollectionTargetSyncRuns, id)
	if err != nil {
		return TargetSyncRun{}, err
	}
	return targetSyncRunFromRecord(record), nil
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
		ErrorMessage:     record.GetString("error_message"),
		Details:          decodeTargetSyncDetails(record.GetString("details_json")),
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
	case TargetSyncEntityProducts:
		return "商品与规格"
	case TargetSyncEntityAssets:
		return "图片资产"
	default:
		return "分类树"
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
		return "全量"
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
	scopeMap := buildCategoryTopLevelLookup(dataset.CategoryPage.Tree)
	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	items := make([]miniappmodel.ProductPage, 0)
	for _, product := range dataset.ProductPage.Products {
		categoryKey := firstCategoryKey(product.SourceSections, sections)
		if categoryKey == "" {
			continue
		}
		if strings.EqualFold(scopeMap[categoryKey], scopeKey) || strings.EqualFold(categoryKey, scopeKey) {
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

func buildCategoryTreeMeta(nodes []miniappmodel.CategoryNode) (map[string]string, map[string]string) {
	paths := make(map[string]string)
	parents := make(map[string]string)
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
				walk(node.Children, path, node.Key)
			}
		}
	}
	walk(nodes, "", "")
	return paths, parents
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
	pathMap, _ := buildCategoryTreeMeta(tree)
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
		record.GetString("source_type"),
		fmt.Sprintf("%d", record.GetInt("unit_count")),
		fmt.Sprintf("%t", record.GetBool("has_multi_unit")),
		fmt.Sprintf("%.2f", record.GetFloat("default_price")),
		fmt.Sprintf("%d", record.GetInt("asset_count")),
	}, "|")
}

func expectedProductSignature(product miniappmodel.ProductPage, dataset miniappmodel.Dataset) string {
	sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
	categoryKey := firstCategoryKey(product.SourceSections, sections)
	categoryPath := ""
	if section, ok := sections[categoryKey]; ok {
		categoryPath = strings.TrimSpace(section.CategoryPath)
	}
	return strings.Join([]string{
		product.ID,
		product.Summary.Name,
		product.Summary.Cover,
		product.Summary.DefaultUnit,
		categoryKey,
		categoryPath,
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

func upsertTargetAssetItem(app core.App, product miniappmodel.ProductPage, asset sourceAssetItem) (bool, error) {
	return upsertByFilter(app, CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key}, func(record *core.Record, created bool) error {
		record.Set("asset_key", asset.Key)
		record.Set("product_id", product.ID)
		record.Set("spu_id", product.SpuID)
		record.Set("sku_id", product.SkuID)
		record.Set("name", product.Summary.Name)
		record.Set("source_url", asset.URL)
		record.Set("asset_role", asset.Role)
		record.Set("sort", asset.Sort)
		if created && strings.TrimSpace(record.GetString("image_processing_status")) == "" {
			record.Set("image_processing_status", ImageStatusPending)
		}
		return setJSON(record, "source_payload", asset.Payload)
	})
}

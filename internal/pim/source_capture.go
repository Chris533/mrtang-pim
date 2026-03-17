package pim

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"mrtang-pim/internal/image"
	miniappmodel "mrtang-pim/internal/miniapp/model"
)

const (
	CollectionSourceCategories = "source_categories"
	CollectionSourceProducts   = "source_products"
	CollectionSourceAssets     = "source_assets"
	CollectionSourceActionLogs = "source_action_logs"
)

type SourceImportSummary struct {
	Scope             string `json:"scope"`
	CategoriesCreated int    `json:"categoriesCreated"`
	CategoriesUpdated int    `json:"categoriesUpdated"`
	ProductsCreated   int    `json:"productsCreated"`
	ProductsUpdated   int    `json:"productsUpdated"`
	AssetsCreated     int    `json:"assetsCreated"`
	AssetsUpdated     int    `json:"assetsUpdated"`
}

type SourceAssetProcessSummary struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

type SourceProductPromotionSummary struct {
	Promoted int `json:"promoted"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

type SourceBatchSummary struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

type SourceReviewProduct struct {
	ID             string             `json:"id"`
	ProductID      string             `json:"productId"`
	Name           string             `json:"name"`
	PreviewURL     string             `json:"previewUrl"`
	CategoryPath   string             `json:"categoryPath"`
	ReviewStatus   string             `json:"reviewStatus"`
	SourceType     string             `json:"sourceType"`
	UnitCount      int                `json:"unitCount"`
	HasMultiUnit   bool               `json:"hasMultiUnit"`
	DefaultPrice   float64            `json:"defaultPrice"`
	AssetCount     int                `json:"assetCount"`
	ProcessedCount int                `json:"processedCount"`
	FailedCount    int                `json:"failedCount"`
	Bridge         SourceBridgeStatus `json:"bridge"`
}

type SourceReviewAsset struct {
	ID                    string `json:"id"`
	AssetKey              string `json:"assetKey"`
	ProductID             string `json:"productId"`
	Name                  string `json:"name"`
	AssetRole             string `json:"assetRole"`
	SourceURL             string `json:"sourceUrl"`
	PreviewURL            string `json:"previewUrl"`
	ProcessedImageURL     string `json:"processedImageUrl"`
	ImageProcessingStatus string `json:"imageProcessingStatus"`
	ImageProcessingError  string `json:"imageProcessingError"`
}

type SourceAssetFailureReason struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type SourceReviewWorkbenchSummary struct {
	CategoryCount       int                        `json:"categoryCount"`
	ProductCount        int                        `json:"productCount"`
	ImportedCount       int                        `json:"importedCount"`
	ApprovedCount       int                        `json:"approvedCount"`
	PromotedCount       int                        `json:"promotedCount"`
	RejectedCount       int                        `json:"rejectedCount"`
	ReadyToReviewCount  int                        `json:"readyToReviewCount"`
	ReadyToPromoteCount int                        `json:"readyToPromoteCount"`
	ReadyToSyncCount    int                        `json:"readyToSyncCount"`
	AssetCount          int                        `json:"assetCount"`
	AssetPending        int                        `json:"assetPending"`
	AssetProcessed      int                        `json:"assetProcessed"`
	AssetFailed         int                        `json:"assetFailed"`
	LinkedCount         int                        `json:"linkedCount"`
	UnlinkedCount       int                        `json:"unlinkedCount"`
	SyncedCount         int                        `json:"syncedCount"`
	SyncErrorCount      int                        `json:"syncErrorCount"`
	FailedLinkedCount   int                        `json:"failedLinkedCount"`
	ProductPage         int                        `json:"productPage"`
	ProductPages        int                        `json:"productPages"`
	ProductLimit        int                        `json:"productLimit"`
	AssetPage           int                        `json:"assetPage"`
	AssetPages          int                        `json:"assetPages"`
	AssetLimit          int                        `json:"assetLimit"`
	AssetFailureReasons []SourceAssetFailureReason `json:"assetFailureReasons"`
	Products            []SourceReviewProduct      `json:"products"`
	Assets              []SourceReviewAsset        `json:"assets"`
	RecentActions       []SourceActionLog          `json:"recentActions"`
}

type SourceReviewFilter struct {
	ProductStatus string `json:"productStatus"`
	AssetStatus   string `json:"assetStatus"`
	SyncState     string `json:"syncState"`
	Query         string `json:"query"`
	ProductPage   int    `json:"productPage"`
	AssetPage     int    `json:"assetPage"`
	PageSize      int    `json:"pageSize"`
}

type SourceBridgeStatus struct {
	Linked           bool   `json:"linked"`
	SupplierRecordID string `json:"supplierRecordId,omitempty"`
	SyncStatus       string `json:"syncStatus,omitempty"`
	VendureProductID string `json:"vendureProductId,omitempty"`
	VendureVariantID string `json:"vendureVariantId,omitempty"`
	LastSyncError    string `json:"lastSyncError,omitempty"`
	LastSyncedAt     string `json:"lastSyncedAt,omitempty"`
}

type SourceActionActor struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type SourceProductDetail struct {
	ID             string             `json:"id"`
	ProductID      string             `json:"productId"`
	Name           string             `json:"name"`
	PreviewURL     string             `json:"previewUrl"`
	ReviewStatus   string             `json:"reviewStatus"`
	ReviewNote     string             `json:"reviewNote"`
	ReviewedByName string             `json:"reviewedByName"`
	ReviewedByMail string             `json:"reviewedByMail"`
	ReviewedAt     string             `json:"reviewedAt"`
	CategoryPath   string             `json:"categoryPath"`
	SourceType     string             `json:"sourceType"`
	SummaryJSON    string             `json:"summaryJson"`
	DetailJSON     string             `json:"detailJson"`
	PricingJSON    string             `json:"pricingJson"`
	PackageJSON    string             `json:"packageJson"`
	ContextJSON    string             `json:"contextJson"`
	UnitOptions    string             `json:"unitOptionsJson"`
	OrderUnits     string             `json:"orderUnitsJson"`
	SourceSections string             `json:"sourceSectionsJson"`
	Bridge         SourceBridgeStatus `json:"bridge"`
	RecentActions  []SourceActionLog  `json:"recentActions"`
}

type SourceAssetDetail struct {
	ID                    string            `json:"id"`
	AssetKey              string            `json:"assetKey"`
	ProductID             string            `json:"productId"`
	Name                  string            `json:"name"`
	AssetRole             string            `json:"assetRole"`
	PreviewURL            string            `json:"previewUrl"`
	SourceURL             string            `json:"sourceUrl"`
	ProcessedImageURL     string            `json:"processedImageUrl"`
	ProcessedImageSource  string            `json:"processedImageSource"`
	ImageProcessingStatus string            `json:"imageProcessingStatus"`
	ImageProcessingError  string            `json:"imageProcessingError"`
	SourcePayloadJSON     string            `json:"sourcePayloadJson"`
	RecentActions         []SourceActionLog `json:"recentActions"`
}

type SourceActionLog struct {
	ID          string `json:"id"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	TargetLabel string `json:"targetLabel"`
	ActionType  string `json:"actionType"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	ActorEmail  string `json:"actorEmail"`
	ActorName   string `json:"actorName"`
	Note        string `json:"note"`
	Created     string `json:"created"`
}

func (s *Service) ImportMiniappSource(ctx context.Context, app core.App, dataset miniappmodel.Dataset, scope string) (SourceImportSummary, error) {
	summary := SourceImportSummary{Scope: normalizedSourceScope(scope)}

	if summary.Scope == "all" || summary.Scope == "categories" {
		if err := s.importCategoryNodes(ctx, app, dataset.CategoryPage.Tree, "", "", &summary); err != nil {
			return summary, err
		}
	}

	if summary.Scope == "all" || summary.Scope == "products" || summary.Scope == "assets" {
		sections := buildCategorySectionLookup(dataset.CategoryPage.Sections)
		for _, product := range dataset.ProductPage.Products {
			categoryKey := firstCategoryKey(product.SourceSections, sections)
			categoryPath := ""
			if section, ok := sections[categoryKey]; ok {
				categoryPath = strings.TrimSpace(section.CategoryPath)
			}

			if summary.Scope == "all" || summary.Scope == "products" {
				if created, err := s.upsertSourceProduct(ctx, app, product, categoryKey, categoryPath); err != nil {
					return summary, err
				} else if created {
					summary.ProductsCreated++
				} else {
					summary.ProductsUpdated++
				}
			}

			if summary.Scope == "all" || summary.Scope == "assets" {
				created, updated, err := s.upsertSourceAssets(ctx, app, product)
				if err != nil {
					return summary, err
				}
				summary.AssetsCreated += created
				summary.AssetsUpdated += updated
			}
		}
	}

	return summary, nil
}

func (s *Service) importCategoryNodes(_ context.Context, app core.App, nodes []miniappmodel.CategoryNode, parentKey string, parentPath string, summary *SourceImportSummary) error {
	for _, node := range nodes {
		path := strings.TrimSpace(node.Label)
		if parentPath != "" {
			path = parentPath + " / " + path
		}

		created, err := upsertByFilter(app, CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": node.Key}, func(record *core.Record, created bool) error {
			record.Set("source_key", node.Key)
			record.Set("label", node.Label)
			record.Set("path_name", node.PathName)
			record.Set("category_path", path)
			record.Set("parent_key", parentKey)
			record.Set("image_url", node.ImageURL)
			record.Set("depth", node.Depth)
			record.Set("sort", node.Sort)
			record.Set("has_children", node.HasChildren)
			return setJSON(record, "source_payload", node)
		})
		if err != nil {
			return err
		}
		if created {
			summary.CategoriesCreated++
		} else {
			summary.CategoriesUpdated++
		}

		if err := s.importCategoryNodes(nil, app, node.Children, node.Key, path, summary); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) upsertSourceProduct(_ context.Context, app core.App, product miniappmodel.ProductPage, categoryKey string, categoryPath string) (bool, error) {
	return upsertByFilter(app, CollectionSourceProducts, "product_id = {:product_id}", dbx.Params{"product_id": product.ID}, func(record *core.Record, created bool) error {
		before := sourceProductSignature(record)
		record.Set("product_id", product.ID)
		record.Set("spu_id", product.SpuID)
		record.Set("sku_id", product.SkuID)
		record.Set("name", product.Summary.Name)
		record.Set("sku_name", product.Summary.SkuName)
		record.Set("cover_url", product.Summary.Cover)
		record.Set("default_unit", product.Summary.DefaultUnit)
		record.Set("default_unit_id", product.Detail.DefaultUnitID)
		record.Set("base_unit_id", product.Detail.BaseUnitID)
		record.Set("category_key", categoryKey)
		record.Set("category_path", categoryPath)
		record.Set("source_type", product.SourceType)
		if created && strings.TrimSpace(record.GetString("review_status")) == "" {
			record.Set("review_status", "imported")
		}
		record.Set("unit_count", len(product.Pricing.UnitOptions))
		record.Set("has_multi_unit", len(product.Pricing.UnitOptions) > 1)
		record.Set("default_price", product.Pricing.DefaultPrice)
		record.Set("default_stock_qty", product.Pricing.DefaultStockQty)
		record.Set("stock_text", product.Pricing.DefaultStockText)
		record.Set("asset_count", countProductAssets(product))
		if err := setJSON(record, "source_sections", product.SourceSections); err != nil {
			return err
		}
		if err := setJSON(record, "tags_json", product.Summary.Tags); err != nil {
			return err
		}
		if err := setJSON(record, "promotion_texts_json", product.Summary.PromotionTexts); err != nil {
			return err
		}
		if err := setJSON(record, "unit_options_json", product.Pricing.UnitOptions); err != nil {
			return err
		}
		if err := setJSON(record, "order_units_json", product.Context.UnitOptions); err != nil {
			return err
		}
		if err := setJSON(record, "summary_json", product.Summary); err != nil {
			return err
		}
		if err := setJSON(record, "detail_json", product.Detail); err != nil {
			return err
		}
		if err := setJSON(record, "pricing_json", product.Pricing); err != nil {
			return err
		}
		if err := setJSON(record, "package_json", product.Package); err != nil {
			return err
		}
		if err := setJSON(record, "context_json", product.Context); err != nil {
			return err
		}
		after := sourceProductSignature(record)
		if !created && before != after {
			record.Set("review_status", "imported")
			record.Set("review_note", "")
			record.Set("reviewed_by_email", "")
			record.Set("reviewed_by_name", "")
			record.Set("reviewed_at", "")
		}
		return nil
	})
}

func (s *Service) upsertSourceAssets(_ context.Context, app core.App, product miniappmodel.ProductPage) (int, int, error) {
	createdCount := 0
	updatedCount := 0
	assets := collectProductAssets(product)
	for _, asset := range assets {
		created, err := upsertByFilter(app, CollectionSourceAssets, "asset_key = {:asset_key}", dbx.Params{"asset_key": asset.Key}, func(record *core.Record, created bool) error {
			before := sourceAssetSignature(record)
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
			if err := setJSON(record, "source_payload", asset.Payload); err != nil {
				return err
			}
			after := sourceAssetSignature(record)
			if !created && before != after {
				record.Set("processed_image", nil)
				record.Set("processed_image_source", "")
				record.Set("image_processing_error", "")
				record.Set("image_processing_status", ImageStatusPending)
			}
			return nil
		})
		if err != nil {
			return createdCount, updatedCount, err
		}
		if created {
			createdCount++
		} else {
			updatedCount++
		}
	}

	return createdCount, updatedCount, nil
}

type sourceAssetItem struct {
	Key     string
	URL     string
	Role    string
	Sort    int
	Payload map[string]any
}

func collectProductAssets(product miniappmodel.ProductPage) []sourceAssetItem {
	items := make([]sourceAssetItem, 0, 1+len(product.Detail.Carousel)+len(product.Detail.DetailAssets))
	add := func(role string, sort int, url string, payload map[string]any) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		items = append(items, sourceAssetItem{
			Key:     fmt.Sprintf("%s:%s:%03d", product.ID, role, sort),
			URL:     url,
			Role:    role,
			Sort:    sort,
			Payload: payload,
		})
	}

	add("cover", 0, product.Summary.Cover, map[string]any{"role": "cover", "name": product.Summary.Name})
	for idx, media := range product.Detail.Carousel {
		add("carousel", idx, media.ImageURL, map[string]any{"type": media.Type, "videoUrl": media.VideoURL})
	}
	for idx, media := range product.Detail.DetailAssets {
		add("detail", idx, media.ImageURL, map[string]any{"type": media.Type, "videoUrl": media.VideoURL})
	}

	return items
}

func countProductAssets(product miniappmodel.ProductPage) int {
	return len(collectProductAssets(product))
}

func buildCategorySectionLookup(sections []miniappmodel.CategorySection) map[string]miniappmodel.CategorySection {
	lookup := make(map[string]miniappmodel.CategorySection, len(sections))
	for _, section := range sections {
		lookup[section.CategoryKey] = section
	}
	return lookup
}

func firstCategoryKey(sourceSections []string, lookup map[string]miniappmodel.CategorySection) string {
	for _, sectionID := range sourceSections {
		if _, ok := lookup[sectionID]; ok {
			return sectionID
		}
	}
	return ""
}

func normalizedSourceScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "categories", "products", "assets":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "all"
	}
}

func setJSON(record *core.Record, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	record.Set(key, string(encoded))
	return nil
}

func upsertByFilter(app core.App, collectionName string, filter string, params dbx.Params, mutate func(record *core.Record, created bool) error) (bool, error) {
	record, err := app.FindFirstRecordByFilter(collectionName, filter, params)
	created := false
	if err != nil {
		collection, findErr := app.FindCollectionByNameOrId(collectionName)
		if findErr != nil {
			return false, findErr
		}
		record = core.NewRecord(collection)
		created = true
	}

	if err := mutate(record, created); err != nil {
		return false, err
	}
	if err := app.Save(record); err != nil {
		return false, err
	}

	return created, nil
}

func (s *Service) ProcessSourceAsset(ctx context.Context, app core.App, assetID string) error {
	return s.ProcessSourceAssetWithAudit(ctx, app, assetID, SourceActionActor{}, "")
}

func (s *Service) ProcessSourceAssetWithAudit(ctx context.Context, app core.App, assetID string, actor SourceActionActor, note string) error {
	record, err := app.FindRecordById(CollectionSourceAssets, assetID)
	if err != nil {
		return err
	}

	record.Set("image_processing_status", ImageStatusProcessing)
	record.Set("image_processing_error", "")
	if err := app.Save(record); err != nil {
		return err
	}

	result, err := s.processor.Process(ctx, imageRequestForSourceAsset(record, s.cfg.Supplier.Code))
	if err != nil {
		record.Set("image_processing_status", ImageStatusFailed)
		record.Set("image_processing_error", err.Error())
		if saveErr := app.Save(record); saveErr != nil {
			return saveErr
		}
		s.logSourceAction(app, "asset", record.Id, record.GetString("asset_key"), "process_asset", "failed", err.Error(), actor, note, map[string]any{
			"productId": record.GetString("product_id"),
		})
		return nil
	}

	record.Set("processed_image", result.File)
	record.Set("processed_image_source", result.Source)
	record.Set("image_processing_status", ImageStatusProcessed)
	record.Set("image_processing_error", "")
	if err := app.Save(record); err != nil {
		return err
	}
	s.logSourceAction(app, "asset", record.Id, record.GetString("asset_key"), "process_asset", "success", "processed source asset", actor, note, map[string]any{
		"productId": record.GetString("product_id"),
	})
	return nil
}

func (s *Service) ProcessPendingSourceAssets(ctx context.Context, app core.App, limit int) (SourceAssetProcessSummary, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSourceAssets,
		"(image_processing_status = {:pending} || image_processing_status = {:failed}) && source_url != ''",
		"sort",
		limit,
		0,
		dbx.Params{
			"pending": ImageStatusPending,
			"failed":  ImageStatusFailed,
		},
	)
	if err != nil {
		return SourceAssetProcessSummary{}, err
	}

	summary := SourceAssetProcessSummary{}
	for _, record := range records {
		if err := s.ProcessSourceAsset(ctx, app, record.Id); err != nil {
			summary.Failed++
			app.Logger().Error("process source asset failed", "assetId", record.Id, "error", err)
			continue
		}
		summary.Processed++
	}

	return summary, nil
}

func (s *Service) ProcessFailedSourceAssets(ctx context.Context, app core.App, limit int) (SourceAssetProcessSummary, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSourceAssets,
		"image_processing_status = {:failed} && source_url != ''",
		"sort",
		limit,
		0,
		dbx.Params{
			"failed": ImageStatusFailed,
		},
	)
	if err != nil {
		return SourceAssetProcessSummary{}, err
	}

	summary := SourceAssetProcessSummary{}
	for _, record := range records {
		if err := s.ProcessSourceAsset(ctx, app, record.Id); err != nil {
			summary.Failed++
			app.Logger().Error("reprocess failed source asset failed", "assetId", record.Id, "error", err)
			continue
		}
		summary.Processed++
	}

	return summary, nil
}

func (s *Service) PromoteApprovedSourceProducts(ctx context.Context, app core.App, limit int) (SourceProductPromotionSummary, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSourceProducts,
		"review_status = {:status}",
		"updated",
		limit,
		0,
		dbx.Params{"status": "approved"},
	)
	if err != nil {
		return SourceProductPromotionSummary{}, err
	}

	summary := SourceProductPromotionSummary{}
	for _, record := range records {
		promoted, err := s.promoteSourceProductRecord(ctx, app, record)
		if err != nil {
			summary.Failed++
			app.Logger().Error("promote source product failed", "productId", record.GetString("product_id"), "error", err)
			continue
		}
		if promoted {
			summary.Promoted++
		} else {
			summary.Skipped++
		}
	}

	return summary, nil
}

func (s *Service) promoteSourceProductRecord(ctx context.Context, app core.App, sourceRecord *core.Record) (bool, error) {
	sku := strings.TrimSpace(sourceRecord.GetString("sku_id"))
	if sku == "" {
		return false, fmt.Errorf("missing sku_id")
	}

	productID := strings.TrimSpace(sourceRecord.GetString("product_id"))
	title := strings.TrimSpace(sourceRecord.GetString("name"))
	categoryPath := strings.TrimSpace(sourceRecord.GetString("category_path"))
	rawImageURL := strings.TrimSpace(sourceRecord.GetString("cover_url"))
	processedFile, processedSource, err := s.bestProcessedSourceAssetFile(app, productID)
	if err != nil {
		return false, err
	}
	bestAssetURL := rawImageURL
	if url, _ := s.bestSourceAssetURL(app, productID); strings.TrimSpace(url) != "" {
		bestAssetURL = url
	}

	_, err = upsertByFilter(app, CollectionSupplierProducts, "supplier_code = {:supplier} && original_sku = {:sku}", dbx.Params{
		"supplier": s.cfg.Supplier.Code,
		"sku":      sku,
	}, func(record *core.Record, created bool) error {
		record.Set("supplier_code", s.cfg.Supplier.Code)
		record.Set("original_sku", sku)
		record.Set("raw_title", title)
		record.Set("normalized_title", title)
		record.Set("raw_description", sourceDescription(sourceRecord))
		record.Set("marketing_description", sourceDescription(sourceRecord))
		record.Set("raw_category", categoryPath)
		record.Set("normalized_category", categoryPath)
		record.Set("raw_image_url", bestAssetURL)
		record.Set("cost_price", 0)
		record.Set("b_price", sourceRecord.GetFloat("default_price"))
		record.Set("c_price", sourceRecord.GetFloat("default_price"))
		record.Set("currency_code", "CNY")
		record.Set("supplier_updated_at", time.Now().Format(time.RFC3339))
		if err := setJSON(record, "supplier_payload", map[string]any{
			"source_product_id": productID,
			"sales_unit":        defaultString(sourceRecord.GetString("default_unit"), "件"),
			"category_key":      sourceRecord.GetString("category_key"),
			"unit_options":      decodeRawJSON(sourceRecord.GetString("unit_options_json")),
			"order_units":       decodeRawJSON(sourceRecord.GetString("order_units_json")),
		}); err != nil {
			return err
		}
		record.Set("last_sync_error", "")
		record.Set("image_processing_error", "")
		if processedFile != nil {
			record.Set("processed_image", processedFile)
			record.Set("processed_image_source", processedSource)
			record.Set("image_processing_status", ImageStatusProcessed)
			record.Set("sync_status", StatusApproved)
		} else {
			record.Set("image_processing_status", ImageStatusPending)
			record.Set("sync_status", StatusPending)
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	sourceRecord.Set("review_status", "promoted")
	if err := app.Save(sourceRecord); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) bestSourceAssetURL(app core.App, productID string) (string, error) {
	assets, err := app.FindRecordsByFilter(CollectionSourceAssets, "product_id = {:product_id}", "sort", 20, 0, dbx.Params{"product_id": productID})
	if err != nil {
		return "", err
	}

	var coverURL string
	for _, asset := range assets {
		if strings.EqualFold(asset.GetString("asset_role"), "cover") && strings.TrimSpace(asset.GetString("source_url")) != "" {
			return asset.GetString("source_url"), nil
		}
		if coverURL == "" && strings.TrimSpace(asset.GetString("source_url")) != "" {
			coverURL = asset.GetString("source_url")
		}
	}
	return coverURL, nil
}

func (s *Service) bestProcessedSourceAssetFile(app core.App, productID string) (*filesystem.File, string, error) {
	assets, err := app.FindRecordsByFilter(CollectionSourceAssets, "product_id = {:product_id}", "sort", 50, 0, dbx.Params{"product_id": productID})
	if err != nil {
		return nil, "", err
	}

	var fallback *core.Record
	for _, asset := range assets {
		if !strings.EqualFold(asset.GetString("image_processing_status"), ImageStatusProcessed) || strings.TrimSpace(asset.GetString("processed_image")) == "" {
			continue
		}
		if strings.EqualFold(asset.GetString("asset_role"), "cover") {
			return cloneRecordFile(app, asset, "processed_image")
		}
		if fallback == nil {
			fallback = asset
		}
	}
	if fallback != nil {
		return cloneRecordFile(app, fallback, "processed_image")
	}
	return nil, "", nil
}

func cloneRecordFile(app core.App, record *core.Record, fieldName string) (*filesystem.File, string, error) {
	filename := strings.TrimSpace(record.GetString(fieldName))
	if filename == "" {
		return nil, "", nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, "", err
	}
	defer fsys.Close()

	file, err := fsys.GetReuploadableFile(record.BaseFilesPath()+"/"+filename, true)
	if err != nil {
		return nil, "", err
	}

	return file, record.GetString("processed_image_source"), nil
}

func imageRequestForSourceAsset(record *core.Record, supplierCode string) image.Request {
	return image.Request{
		SupplierCode: supplierCode,
		SKU:          record.GetString("sku_id"),
		Title:        defaultString(record.GetString("name"), record.GetString("product_id")),
		SourceURL:    record.GetString("source_url"),
	}
}

func sourceDescription(record *core.Record) string {
	detail := decodeRawJSON(record.GetString("detail_json"))
	if m, ok := detail.(map[string]any); ok {
		if texts, ok := m["detailTexts"].([]any); ok && len(texts) > 0 {
			parts := make([]string, 0, len(texts))
			for _, item := range texts {
				value := strings.TrimSpace(fmt.Sprintf("%v", item))
				if value != "" {
					parts = append(parts, value)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}
	return strings.TrimSpace(record.GetString("name"))
}

func decodeRawJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	return decoded
}

func prettyJSONString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return raw
	}
	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return raw
	}
	return string(pretty)
}

func normalizeSourceReviewFilter(filter SourceReviewFilter) SourceReviewFilter {
	filter.ProductStatus = strings.ToLower(strings.TrimSpace(filter.ProductStatus))
	filter.AssetStatus = strings.ToLower(strings.TrimSpace(filter.AssetStatus))
	filter.SyncState = strings.ToLower(strings.TrimSpace(filter.SyncState))
	filter.Query = strings.TrimSpace(filter.Query)
	if filter.ProductPage <= 0 {
		filter.ProductPage = 1
	}
	if filter.AssetPage <= 0 {
		filter.AssetPage = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 24
	}
	return filter
}

func buildSourceReviewParams(filter SourceReviewFilter) dbx.Params {
	params := dbx.Params{}
	if filter.ProductStatus != "" {
		params["product_status"] = filter.ProductStatus
	}
	if filter.AssetStatus != "" {
		params["asset_status"] = filter.AssetStatus
	}
	if filter.Query != "" {
		params["query"] = "%" + filter.Query + "%"
	}
	return params
}

func buildSourceProductFilterExpr(filter SourceReviewFilter) string {
	parts := make([]string, 0, 2)
	if filter.ProductStatus != "" {
		parts = append(parts, "review_status = {:product_status}")
	}
	if filter.Query != "" {
		parts = append(parts, "(name ~ {:query} || product_id ~ {:query} || category_path ~ {:query})")
	}
	return strings.Join(parts, " && ")
}

func buildSourceAssetFilterExpr(filter SourceReviewFilter) string {
	parts := make([]string, 0, 2)
	if filter.AssetStatus != "" {
		parts = append(parts, "image_processing_status = {:asset_status}")
	}
	if filter.Query != "" {
		parts = append(parts, "(name ~ {:query} || asset_key ~ {:query} || product_id ~ {:query})")
	}
	return strings.Join(parts, " && ")
}

func buildSupplierProductLookup(supplierCode string, records []*core.Record) map[string]*core.Record {
	lookup := make(map[string]*core.Record, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.GetString("supplier_code")) != strings.TrimSpace(supplierCode) {
			continue
		}
		sku := strings.TrimSpace(record.GetString("original_sku"))
		if sku == "" {
			continue
		}
		lookup[sku] = record
	}
	return lookup
}

func supplierBridgeStatusForRecord(record *core.Record) SourceBridgeStatus {
	if record == nil {
		return SourceBridgeStatus{}
	}
	return SourceBridgeStatus{
		Linked:           true,
		SupplierRecordID: record.Id,
		SyncStatus:       record.GetString("sync_status"),
		VendureProductID: record.GetString("vendure_product_id"),
		VendureVariantID: record.GetString("vendure_variant_id"),
		LastSyncError:    record.GetString("last_sync_error"),
		LastSyncedAt:     record.GetString("last_synced_at"),
	}
}

func matchesSourceSyncState(state string, bridge SourceBridgeStatus) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "":
		return true
	case "unlinked":
		return !bridge.Linked
	case "error":
		return bridge.Linked && strings.EqualFold(bridge.SyncStatus, StatusError)
	case "synced":
		return bridge.Linked && strings.EqualFold(bridge.SyncStatus, StatusSynced)
	case "linked":
		return bridge.Linked
	default:
		return true
	}
}

func matchesSourceProductFilter(filter SourceReviewFilter, record *core.Record) bool {
	if filter.ProductStatus != "" && !strings.EqualFold(strings.TrimSpace(record.GetString("review_status")), filter.ProductStatus) {
		return false
	}
	if filter.Query != "" {
		query := strings.ToLower(filter.Query)
		search := strings.ToLower(strings.Join([]string{
			record.GetString("name"),
			record.GetString("product_id"),
			record.GetString("category_path"),
		}, " "))
		if !strings.Contains(search, query) {
			return false
		}
	}
	return true
}

func matchesSourceAssetFilter(filter SourceReviewFilter, record *core.Record) bool {
	if filter.AssetStatus != "" && !strings.EqualFold(strings.TrimSpace(record.GetString("image_processing_status")), filter.AssetStatus) {
		return false
	}
	if filter.Query != "" {
		query := strings.ToLower(filter.Query)
		search := strings.ToLower(strings.Join([]string{
			record.GetString("name"),
			record.GetString("asset_key"),
			record.GetString("product_id"),
		}, " "))
		if !strings.Contains(search, query) {
			return false
		}
	}
	return true
}

func totalPages(total int, pageSize int) int {
	if total == 0 {
		return 1
	}
	if pageSize <= 0 {
		pageSize = 24
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	if pages <= 0 {
		pages = 1
	}
	return pages
}

func paginateProducts(items []SourceReviewProduct, page int, pageSize int) []SourceReviewProduct {
	start, end := paginateBounds(len(items), page, pageSize)
	return items[start:end]
}

func paginateAssets(items []SourceReviewAsset, page int, pageSize int) []SourceReviewAsset {
	start, end := paginateBounds(len(items), page, pageSize)
	return items[start:end]
}

func paginateBounds(total int, page int, pageSize int) (int, int) {
	if pageSize <= 0 {
		pageSize = 24
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func (s *Service) SourceReviewWorkbench(ctx context.Context, app core.App, productLimit int, assetLimit int, filter SourceReviewFilter) (SourceReviewWorkbenchSummary, error) {
	summary := SourceReviewWorkbenchSummary{}
	filter = normalizeSourceReviewFilter(filter)

	categories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return summary, err
	}
	allProducts, err := app.FindAllRecords(CollectionSourceProducts)
	if err != nil {
		return summary, err
	}
	supplierRecords, err := app.FindAllRecords(CollectionSupplierProducts)
	if err != nil {
		return summary, err
	}
	allAssets, err := app.FindAllRecords(CollectionSourceAssets)
	if err != nil {
		return summary, err
	}

	summary.CategoryCount = len(categories)
	summary.ProductCount = len(allProducts)
	summary.AssetCount = len(allAssets)
	summary.ProductLimit = filter.PageSize
	summary.AssetLimit = filter.PageSize
	supplierBySKU := buildSupplierProductLookup(s.cfg.Supplier.Code, supplierRecords)
	failureCounts := map[string]int{}

	assetStats := make(map[string]struct{ processed, failed int }, len(allAssets))
	for _, asset := range allAssets {
		switch strings.ToLower(strings.TrimSpace(asset.GetString("image_processing_status"))) {
		case ImageStatusProcessed:
			summary.AssetProcessed++
			stat := assetStats[asset.GetString("product_id")]
			stat.processed++
			assetStats[asset.GetString("product_id")] = stat
		case ImageStatusFailed:
			summary.AssetFailed++
			stat := assetStats[asset.GetString("product_id")]
			stat.failed++
			assetStats[asset.GetString("product_id")] = stat
			reason := strings.TrimSpace(asset.GetString("image_processing_error"))
			if reason == "" {
				reason = "unknown failure"
			}
			failureCounts[reason]++
		default:
			summary.AssetPending++
		}
	}
	summary.AssetFailureReasons = sortAssetFailureReasons(failureCounts)

	for _, record := range allProducts {
		status := strings.ToLower(strings.TrimSpace(record.GetString("review_status")))
		switch status {
		case "approved":
			summary.ApprovedCount++
			summary.ReadyToPromoteCount++
		case "promoted":
			summary.PromotedCount++
		case "rejected":
			summary.RejectedCount++
		default:
			summary.ImportedCount++
			summary.ReadyToReviewCount++
		}

		bridge := supplierBridgeStatusForRecord(supplierBySKU[record.GetString("sku_id")])
		if bridge.Linked {
			summary.LinkedCount++
		} else {
			summary.UnlinkedCount++
		}
		switch strings.ToLower(strings.TrimSpace(bridge.SyncStatus)) {
		case StatusSynced:
			summary.SyncedCount++
		case StatusError:
			summary.SyncErrorCount++
			if bridge.Linked {
				summary.FailedLinkedCount++
			}
		case StatusApproved, StatusReady:
			if bridge.Linked {
				summary.ReadyToSyncCount++
			}
		}
	}

	filteredProducts := make([]SourceReviewProduct, 0, len(allProducts))
	for _, record := range allProducts {
		stat := assetStats[record.GetString("product_id")]
		bridge := supplierBridgeStatusForRecord(supplierBySKU[record.GetString("sku_id")])
		if !matchesSourceProductFilter(filter, record) {
			continue
		}
		if !matchesSourceSyncState(filter.SyncState, bridge) {
			continue
		}
		filteredProducts = append(filteredProducts, SourceReviewProduct{
			ID:             record.Id,
			ProductID:      record.GetString("product_id"),
			Name:           record.GetString("name"),
			PreviewURL:     s.sourceProductPreviewURL(app, record),
			CategoryPath:   record.GetString("category_path"),
			ReviewStatus:   record.GetString("review_status"),
			SourceType:     record.GetString("source_type"),
			UnitCount:      record.GetInt("unit_count"),
			HasMultiUnit:   record.GetBool("has_multi_unit"),
			DefaultPrice:   record.GetFloat("default_price"),
			AssetCount:     record.GetInt("asset_count"),
			ProcessedCount: stat.processed,
			FailedCount:    stat.failed,
			Bridge:         bridge,
		})
	}
	summary.ProductPage = filter.ProductPage
	summary.ProductPages = totalPages(len(filteredProducts), filter.PageSize)
	summary.Products = paginateProducts(filteredProducts, filter.ProductPage, filter.PageSize)

	filteredAssets := make([]SourceReviewAsset, 0, len(allAssets))
	for _, record := range allAssets {
		if !matchesSourceAssetFilter(filter, record) {
			continue
		}
		filteredAssets = append(filteredAssets, SourceReviewAsset{
			ID:                    record.Id,
			AssetKey:              record.GetString("asset_key"),
			ProductID:             record.GetString("product_id"),
			Name:                  record.GetString("name"),
			AssetRole:             record.GetString("asset_role"),
			SourceURL:             record.GetString("source_url"),
			PreviewURL:            s.sourceAssetPreviewURL(app, record),
			ProcessedImageURL:     s.recordFileURL(record, "processed_image"),
			ImageProcessingStatus: record.GetString("image_processing_status"),
			ImageProcessingError:  record.GetString("image_processing_error"),
		})
	}
	summary.AssetPage = filter.AssetPage
	summary.AssetPages = totalPages(len(filteredAssets), filter.PageSize)
	summary.Assets = paginateAssets(filteredAssets, filter.AssetPage, filter.PageSize)
	summary.RecentActions, _ = s.listRecentSourceActions(app, 8)

	return summary, nil
}

func (s *Service) UpdateSourceProductReviewStatus(ctx context.Context, app core.App, recordID string, status string) error {
	return s.UpdateSourceProductReviewStatusWithAudit(ctx, app, recordID, status, "", SourceActionActor{})
}

func (s *Service) UpdateSourceProductReviewStatusWithAudit(ctx context.Context, app core.App, recordID string, status string, note string, actor SourceActionActor) error {
	record, err := app.FindRecordById(CollectionSourceProducts, recordID)
	if err != nil {
		return err
	}

	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "imported", "approved", "rejected", "promoted":
	default:
		return fmt.Errorf("invalid source product review status %q", status)
	}

	record.Set("review_status", status)
	record.Set("review_note", strings.TrimSpace(note))
	record.Set("reviewed_by_email", strings.TrimSpace(actor.Email))
	record.Set("reviewed_by_name", strings.TrimSpace(actor.Name))
	record.Set("reviewed_at", time.Now().Format(time.RFC3339))
	if err := app.Save(record); err != nil {
		return err
	}
	s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "update_review_status", "success", "updated source product review status", actor, note, map[string]any{
		"reviewStatus": status,
	})
	return nil
}

func (s *Service) PromoteSourceProduct(ctx context.Context, app core.App, recordID string) error {
	return s.PromoteSourceProductWithAudit(ctx, app, recordID, SourceActionActor{}, "")
}

func (s *Service) PromoteSourceProductWithAudit(ctx context.Context, app core.App, recordID string, actor SourceActionActor, note string) error {
	record, err := app.FindRecordById(CollectionSourceProducts, recordID)
	if err != nil {
		return err
	}
	_, err = s.promoteSourceProductRecord(ctx, app, record)
	if err != nil {
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_product", "failed", err.Error(), actor, note, nil)
		return err
	}
	s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_product", "success", "promoted source product", actor, note, nil)
	return nil
}

func (s *Service) BatchUpdateSourceProductReviewStatus(ctx context.Context, app core.App, ids []string, status string) (SourceBatchSummary, error) {
	return s.BatchUpdateSourceProductReviewStatusWithAudit(ctx, app, ids, status, "", SourceActionActor{})
}

func (s *Service) BatchUpdateSourceProductReviewStatusWithAudit(ctx context.Context, app core.App, ids []string, status string, note string, actor SourceActionActor) (SourceBatchSummary, error) {
	summary := SourceBatchSummary{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.UpdateSourceProductReviewStatusWithAudit(ctx, app, id, status, note, actor); err != nil {
			summary.Failed++
			continue
		}
		summary.Processed++
	}
	return summary, nil
}

func (s *Service) BatchPromoteSourceProducts(ctx context.Context, app core.App, ids []string, syncNow bool) (SourceBatchSummary, error) {
	return s.BatchPromoteSourceProductsWithAudit(ctx, app, ids, syncNow, SourceActionActor{}, "")
}

func (s *Service) BatchPromoteSourceProductsWithAudit(ctx context.Context, app core.App, ids []string, syncNow bool, actor SourceActionActor, note string) (SourceBatchSummary, error) {
	summary := SourceBatchSummary{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		var err error
		if syncNow {
			err = s.PromoteAndSyncSourceProductWithAudit(ctx, app, id, actor, note)
		} else {
			err = s.PromoteSourceProductWithAudit(ctx, app, id, actor, note)
		}
		if err != nil {
			summary.Failed++
			continue
		}
		summary.Processed++
	}
	return summary, nil
}

func (s *Service) BatchProcessSourceAssets(ctx context.Context, app core.App, ids []string) (SourceBatchSummary, error) {
	return s.BatchProcessSourceAssetsWithAudit(ctx, app, ids, SourceActionActor{}, "")
}

func (s *Service) BatchProcessSourceAssetsWithAudit(ctx context.Context, app core.App, ids []string, actor SourceActionActor, note string) (SourceBatchSummary, error) {
	summary := SourceBatchSummary{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.ProcessSourceAssetWithAudit(ctx, app, id, actor, note); err != nil {
			summary.Failed++
			continue
		}
		summary.Processed++
	}
	return summary, nil
}

func (s *Service) PromoteAndSyncSourceProduct(ctx context.Context, app core.App, recordID string) error {
	return s.PromoteAndSyncSourceProductWithAudit(ctx, app, recordID, SourceActionActor{}, "")
}

func (s *Service) PromoteAndSyncSourceProductWithAudit(ctx context.Context, app core.App, recordID string, actor SourceActionActor, note string) error {
	record, err := app.FindRecordById(CollectionSourceProducts, recordID)
	if err != nil {
		return err
	}
	if _, err := s.promoteSourceProductRecord(ctx, app, record); err != nil {
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_and_sync", "failed", err.Error(), actor, note, nil)
		return err
	}

	sku := strings.TrimSpace(record.GetString("sku_id"))
	if sku == "" {
		return fmt.Errorf("missing sku_id")
	}
	supplierRecord, err := app.FindFirstRecordByFilter(
		CollectionSupplierProducts,
		"supplier_code = {:supplier} && original_sku = {:sku}",
		dbx.Params{"supplier": s.cfg.Supplier.Code, "sku": sku},
	)
	if err != nil {
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_and_sync", "failed", err.Error(), actor, note, nil)
		return err
	}
	if err := s.syncRecord(ctx, app, supplierRecord); err != nil {
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_and_sync", "failed", err.Error(), actor, note, nil)
		return err
	}
	s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "promote_and_sync", "success", "promoted and synced source product", actor, note, map[string]any{
		"supplierRecordId": supplierRecord.Id,
	})
	return nil
}

func (s *Service) RetrySourceProductSync(ctx context.Context, app core.App, recordID string) error {
	return s.RetrySourceProductSyncWithAudit(ctx, app, recordID, SourceActionActor{}, "")
}

func (s *Service) RetrySourceProductSyncWithAudit(ctx context.Context, app core.App, recordID string, actor SourceActionActor, note string) error {
	record, err := app.FindRecordById(CollectionSourceProducts, recordID)
	if err != nil {
		return err
	}
	supplierRecord, err := app.FindFirstRecordByFilter(
		CollectionSupplierProducts,
		"supplier_code = {:supplier} && original_sku = {:sku}",
		dbx.Params{"supplier": s.cfg.Supplier.Code, "sku": record.GetString("sku_id")},
	)
	if err != nil {
		err = fmt.Errorf("linked supplier product not found")
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "retry_sync", "failed", err.Error(), actor, note, nil)
		return err
	}
	if err := s.syncRecord(ctx, app, supplierRecord); err != nil {
		s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "retry_sync", "failed", err.Error(), actor, note, map[string]any{
			"supplierRecordId": supplierRecord.Id,
		})
		return err
	}
	s.logSourceAction(app, "product", record.Id, record.GetString("product_id"), "retry_sync", "success", "retried supplier sync", actor, note, map[string]any{
		"supplierRecordId": supplierRecord.Id,
	})
	return nil
}

func (s *Service) BatchRetrySourceProductSync(ctx context.Context, app core.App, ids []string) (SourceBatchSummary, error) {
	return s.BatchRetrySourceProductSyncWithAudit(ctx, app, ids, SourceActionActor{}, "")
}

func (s *Service) BatchRetrySourceProductSyncWithAudit(ctx context.Context, app core.App, ids []string, actor SourceActionActor, note string) (SourceBatchSummary, error) {
	summary := SourceBatchSummary{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.RetrySourceProductSyncWithAudit(ctx, app, id, actor, note); err != nil {
			summary.Failed++
			continue
		}
		summary.Processed++
	}
	return summary, nil
}

func (s *Service) SourceProductDetail(ctx context.Context, app core.App, recordID string) (SourceProductDetail, error) {
	record, err := app.FindRecordById(CollectionSourceProducts, recordID)
	if err != nil {
		return SourceProductDetail{}, err
	}
	supplierRecord, _ := app.FindFirstRecordByFilter(
		CollectionSupplierProducts,
		"supplier_code = {:supplier} && original_sku = {:sku}",
		dbx.Params{"supplier": s.cfg.Supplier.Code, "sku": record.GetString("sku_id")},
	)
	recentActions, _ := s.listRecentSourceActionsForTarget(app, "product", record.Id, 8)

	return SourceProductDetail{
		ID:             record.Id,
		ProductID:      record.GetString("product_id"),
		Name:           record.GetString("name"),
		PreviewURL:     s.sourceProductPreviewURL(app, record),
		ReviewStatus:   record.GetString("review_status"),
		ReviewNote:     record.GetString("review_note"),
		ReviewedByName: record.GetString("reviewed_by_name"),
		ReviewedByMail: record.GetString("reviewed_by_email"),
		ReviewedAt:     record.GetString("reviewed_at"),
		CategoryPath:   record.GetString("category_path"),
		SourceType:     record.GetString("source_type"),
		SummaryJSON:    prettyJSONString(record.GetString("summary_json")),
		DetailJSON:     prettyJSONString(record.GetString("detail_json")),
		PricingJSON:    prettyJSONString(record.GetString("pricing_json")),
		PackageJSON:    prettyJSONString(record.GetString("package_json")),
		ContextJSON:    prettyJSONString(record.GetString("context_json")),
		UnitOptions:    prettyJSONString(record.GetString("unit_options_json")),
		OrderUnits:     prettyJSONString(record.GetString("order_units_json")),
		SourceSections: prettyJSONString(record.GetString("source_sections")),
		Bridge:         supplierBridgeStatusForRecord(supplierRecord),
		RecentActions:  recentActions,
	}, nil
}

func (s *Service) SourceAssetDetail(ctx context.Context, app core.App, recordID string) (SourceAssetDetail, error) {
	record, err := app.FindRecordById(CollectionSourceAssets, recordID)
	if err != nil {
		return SourceAssetDetail{}, err
	}
	recentActions, _ := s.listRecentSourceActionsForTarget(app, "asset", record.Id, 8)

	return SourceAssetDetail{
		ID:                    record.Id,
		AssetKey:              record.GetString("asset_key"),
		ProductID:             record.GetString("product_id"),
		Name:                  record.GetString("name"),
		AssetRole:             record.GetString("asset_role"),
		PreviewURL:            s.sourceAssetPreviewURL(app, record),
		SourceURL:             record.GetString("source_url"),
		ProcessedImageURL:     s.recordFileURL(record, "processed_image"),
		ProcessedImageSource:  record.GetString("processed_image_source"),
		ImageProcessingStatus: record.GetString("image_processing_status"),
		ImageProcessingError:  record.GetString("image_processing_error"),
		SourcePayloadJSON:     prettyJSONString(record.GetString("source_payload")),
		RecentActions:         recentActions,
	}, nil
}

func (s *Service) sourceProductPreviewURL(app core.App, record *core.Record) string {
	productID := strings.TrimSpace(record.GetString("product_id"))
	if productID == "" {
		return strings.TrimSpace(record.GetString("cover_url"))
	}
	if processed, err := s.bestProcessedSourceAssetURL(app, productID); err == nil && strings.TrimSpace(processed) != "" {
		return processed
	}
	if source, err := s.bestSourceAssetURL(app, productID); err == nil && strings.TrimSpace(source) != "" {
		return source
	}
	return strings.TrimSpace(record.GetString("cover_url"))
}

func (s *Service) sourceAssetPreviewURL(app core.App, record *core.Record) string {
	if processed := s.recordFileURL(record, "processed_image"); strings.TrimSpace(processed) != "" {
		return processed
	}
	return strings.TrimSpace(record.GetString("source_url"))
}

func (s *Service) bestProcessedSourceAssetURL(app core.App, productID string) (string, error) {
	assets, err := app.FindRecordsByFilter(CollectionSourceAssets, "product_id = {:product_id}", "sort", 50, 0, dbx.Params{"product_id": productID})
	if err != nil {
		return "", err
	}
	var fallback string
	for _, asset := range assets {
		if !strings.EqualFold(asset.GetString("image_processing_status"), ImageStatusProcessed) || strings.TrimSpace(asset.GetString("processed_image")) == "" {
			continue
		}
		url := s.recordFileURL(asset, "processed_image")
		if strings.EqualFold(asset.GetString("asset_role"), "cover") && url != "" {
			return url, nil
		}
		if fallback == "" {
			fallback = url
		}
	}
	return fallback, nil
}

func (s *Service) recordFileURL(record *core.Record, fieldName string) string {
	filename := strings.TrimSpace(record.GetString(fieldName))
	if filename == "" {
		return ""
	}
	base := strings.TrimRight(s.cfg.App.PublicURL, "/")
	return fmt.Sprintf("%s/api/files/%s/%s/%s", base, record.Collection().Id, record.Id, filename)
}

func sortAssetFailureReasons(counts map[string]int) []SourceAssetFailureReason {
	items := make([]SourceAssetFailureReason, 0, len(counts))
	for message, count := range counts {
		items = append(items, SourceAssetFailureReason{
			Message: message,
			Count:   count,
		})
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Count > items[i].Count || (items[j].Count == items[i].Count && items[j].Message < items[i].Message) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func (s *Service) logSourceAction(app core.App, targetType string, targetID string, targetLabel string, actionType string, status string, message string, actor SourceActionActor, note string, details any) {
	collection, err := app.FindCollectionByNameOrId(CollectionSourceActionLogs)
	if err != nil {
		return
	}
	record := core.NewRecord(collection)
	record.Set("target_type", strings.TrimSpace(targetType))
	record.Set("target_id", strings.TrimSpace(targetID))
	record.Set("target_label", strings.TrimSpace(targetLabel))
	record.Set("action_type", strings.TrimSpace(actionType))
	record.Set("status", strings.TrimSpace(status))
	record.Set("message", strings.TrimSpace(message))
	record.Set("actor_email", strings.TrimSpace(actor.Email))
	record.Set("actor_name", strings.TrimSpace(actor.Name))
	record.Set("note", strings.TrimSpace(note))
	if details != nil {
		if err := setJSON(record, "details_json", details); err != nil {
			return
		}
	}
	_ = app.Save(record)
}

func (s *Service) listRecentSourceActions(app core.App, limit int) ([]SourceActionLog, error) {
	if limit <= 0 {
		limit = 8
	}
	records, err := app.FindRecordsByFilter(CollectionSourceActionLogs, "", "-created", limit, 0, nil)
	if err != nil {
		return nil, err
	}
	items := make([]SourceActionLog, 0, len(records))
	for _, record := range records {
		items = append(items, SourceActionLog{
			ID:          record.Id,
			TargetType:  record.GetString("target_type"),
			TargetID:    record.GetString("target_id"),
			TargetLabel: record.GetString("target_label"),
			ActionType:  record.GetString("action_type"),
			Status:      record.GetString("status"),
			Message:     record.GetString("message"),
			ActorEmail:  record.GetString("actor_email"),
			ActorName:   record.GetString("actor_name"),
			Note:        record.GetString("note"),
			Created:     record.GetString("created"),
		})
	}
	return items, nil
}

func (s *Service) listRecentSourceActionsForTarget(app core.App, targetType string, targetID string, limit int) ([]SourceActionLog, error) {
	if limit <= 0 {
		limit = 8
	}
	records, err := app.FindRecordsByFilter(
		CollectionSourceActionLogs,
		"target_type = {:target_type} && target_id = {:target_id}",
		"-created",
		limit,
		0,
		dbx.Params{
			"target_type": strings.TrimSpace(targetType),
			"target_id":   strings.TrimSpace(targetID),
		},
	)
	if err != nil {
		return nil, err
	}
	items := make([]SourceActionLog, 0, len(records))
	for _, record := range records {
		items = append(items, SourceActionLog{
			ID:          record.Id,
			TargetType:  record.GetString("target_type"),
			TargetID:    record.GetString("target_id"),
			TargetLabel: record.GetString("target_label"),
			ActionType:  record.GetString("action_type"),
			Status:      record.GetString("status"),
			Message:     record.GetString("message"),
			ActorEmail:  record.GetString("actor_email"),
			ActorName:   record.GetString("actor_name"),
			Note:        record.GetString("note"),
			Created:     record.GetString("created"),
		})
	}
	return items, nil
}

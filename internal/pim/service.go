package pim

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/image"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/supplier"
	"mrtang-pim/internal/vendure"
)

const (
	CollectionSupplierProducts        = "supplier_products"
	CollectionCategoryMappings        = "category_mappings"
	CollectionBackendCategoryMappings = "backend_category_mappings"
	CollectionProcurementOrders       = "procurement_orders"
	CollectionProcurementActionLogs   = "procurement_action_logs"
	CollectionMiniappActionLogs       = "miniapp_action_logs"
	CollectionSourceAssetJobs         = "source_asset_jobs"
	CollectionSourceProductJobs       = "source_product_jobs"

	StatusPending      = "pending"
	StatusAIProcessing = "ai_processing"
	StatusReady        = "ready"
	StatusApproved     = "approved"
	StatusSynced       = "synced"
	StatusOffline      = "offline"
	StatusError        = "error"

	SupplierStatusActive       = "active"
	SupplierStatusOutOfStock   = "out_of_stock"
	SupplierStatusOffline      = "offline"
	SupplierStatusPriceChanged = "price_changed"
	SupplierStatusSpecChanged  = "spec_changed"

	ImageStatusPending    = "pending"
	ImageStatusProcessing = "processing"
	ImageStatusProcessed  = "processed"
	ImageStatusFailed     = "failed"

	OriginalImageStatusPending     = "pending"
	OriginalImageStatusDownloading = "downloading"
	OriginalImageStatusDownloaded  = "downloaded"
	OriginalImageStatusFailed      = "failed"

	ProcurementStatusDraft    = "draft"
	ProcurementStatusReviewed = "reviewed"
	ProcurementStatusExported = "exported"
	ProcurementStatusOrdered  = "ordered"
	ProcurementStatusReceived = "received"
	ProcurementStatusCanceled = "canceled"
)

type Result struct {
	Action    string `json:"action"`
	Processed int    `json:"processed"`
	Created   int    `json:"created"`
	Updated   int    `json:"updated"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	Offline   int    `json:"offline"`
}

type SupplierSingleImageCandidate struct {
	ID               string `json:"id"`
	SupplierCode     string `json:"supplierCode"`
	OriginalSKU      string `json:"originalSku"`
	Title            string `json:"title"`
	SourceType       string `json:"sourceType"`
	SyncStatus       string `json:"syncStatus"`
	VendureProductID string `json:"vendureProductId"`
	GalleryCount     int    `json:"galleryCount"`
	UpdatedAt        string `json:"updatedAt"`
}

type SupplierDuplicateCleanupResult struct {
	ProductCount    int      `json:"productCount"`
	LinkedCount     int      `json:"linkedCount"`
	DuplicateGroups int      `json:"duplicateGroups"`
	CandidateCount  int      `json:"candidateCount"`
	DeletedCount    int      `json:"deletedCount"`
	FailedCount     int      `json:"failedCount"`
	DeletedIDs      []string `json:"deletedIds,omitempty"`
	FailedIDs       []string `json:"failedIds,omitempty"`
}

type harvestOfflineSummary struct {
	OfflineCount int
	FailureItems []HarvestFailureItem
}

type ProcurementItemRequest struct {
	SupplierCode  string  `json:"supplierCode"`
	OriginalSKU   string  `json:"originalSku"`
	SalesUnit     string  `json:"salesUnit,omitempty"`
	Quantity      float64 `json:"quantity"`
	ExpectedPrice float64 `json:"expectedPrice,omitempty"`
}

type ProcurementRequest struct {
	ExternalRef     string                   `json:"externalRef"`
	DeliveryAddress string                   `json:"deliveryAddress"`
	Notes           string                   `json:"notes"`
	Items           []ProcurementItemRequest `json:"items"`
}

type ProcurementSummaryItem struct {
	SupplierCode       string  `json:"supplierCode"`
	OriginalSKU        string  `json:"originalSku"`
	Title              string  `json:"title"`
	NormalizedCategory string  `json:"normalizedCategory"`
	Quantity           float64 `json:"quantity"`
	SalesUnit          string  `json:"salesUnit"`
	CostPrice          float64 `json:"costPrice"`
	CostAmount         float64 `json:"costAmount"`
	BusinessPrice      float64 `json:"businessPrice"`
	BusinessAmount     float64 `json:"businessAmount"`
	ConsumerPrice      float64 `json:"consumerPrice"`
	ConsumerAmount     float64 `json:"consumerAmount"`
	MarginRatio        float64 `json:"marginRatio"`
	RiskLevel          string  `json:"riskLevel"`
	NeedColdChain      bool    `json:"needColdChain"`
}

type ProcurementSupplierSummary struct {
	SupplierCode        string                   `json:"supplierCode"`
	ItemCount           int                      `json:"itemCount"`
	TotalQty            float64                  `json:"totalQty"`
	TotalCostAmount     float64                  `json:"totalCostAmount"`
	TotalBusinessAmount float64                  `json:"totalBusinessAmount"`
	TotalConsumerAmount float64                  `json:"totalConsumerAmount"`
	RiskyItemCount      int                      `json:"riskyItemCount"`
	Items               []ProcurementSummaryItem `json:"items"`
}

type ProcurementSummary struct {
	Connector           string                         `json:"connector"`
	Capabilities        supplier.ConnectorCapabilities `json:"capabilities"`
	ExternalRef         string                         `json:"externalRef"`
	DeliveryAddress     string                         `json:"deliveryAddress,omitempty"`
	Notes               string                         `json:"notes,omitempty"`
	SupplierCount       int                            `json:"supplierCount"`
	ItemCount           int                            `json:"itemCount"`
	TotalQty            float64                        `json:"totalQty"`
	TotalCostAmount     float64                        `json:"totalCostAmount"`
	TotalBusinessAmount float64                        `json:"totalBusinessAmount"`
	TotalConsumerAmount float64                        `json:"totalConsumerAmount"`
	RiskyItemCount      int                            `json:"riskyItemCount"`
	Suppliers           []ProcurementSupplierSummary   `json:"suppliers"`
}

type ProcurementExport struct {
	FileName    string             `json:"fileName"`
	ContentType string             `json:"contentType"`
	RowCount    int                `json:"rowCount"`
	CSV         string             `json:"csv"`
	Summary     ProcurementSummary `json:"summary"`
}

func (s *Service) MiniappCollectionsTree(ctx context.Context) ([]vendure.MiniappCollectionNode, error) {
	return s.vendure.MiniappCollectionsTree(ctx)
}

func (s *Service) MiniappCollectionProducts(ctx context.Context, slug string, audience string, skip int, take int) (vendure.MiniappProductList, error) {
	return s.vendure.MiniappCollectionProducts(ctx, slug, audience, skip, take)
}

func (s *Service) MiniappProductDetail(ctx context.Context, slug string, audience string) (*vendure.MiniappProductDetail, error) {
	return s.vendure.MiniappProductDetail(ctx, slug, audience)
}

func (s *Service) CleanupBackendAssets(ctx context.Context) (vendure.AssetCleanupResult, error) {
	return s.vendure.CleanupOrphanedPIMAssets(ctx)
}

func (s *Service) CleanupDuplicateOrphanProducts(ctx context.Context, app core.App) (SupplierDuplicateCleanupResult, error) {
	linkedIDs, err := s.linkedVendureProductIDs(app)
	if err != nil {
		return SupplierDuplicateCleanupResult{}, err
	}

	products, err := s.vendure.ListProducts(ctx)
	if err != nil {
		return SupplierDuplicateCleanupResult{}, err
	}

	grouped := make(map[string][]vendure.ProductBasic)
	for _, item := range products {
		key := normalizeDuplicateNameKey(item.Name)
		if key == "" {
			continue
		}
		grouped[key] = append(grouped[key], item)
	}

	candidates := make([]string, 0, 128)
	duplicateGroups := 0
	for _, items := range grouped {
		if len(items) < 2 {
			continue
		}
		hasLinked := false
		for _, item := range items {
			if _, ok := linkedIDs[strings.TrimSpace(item.ID)]; ok {
				hasLinked = true
				break
			}
		}
		if !hasLinked {
			continue
		}
		duplicateGroups++
		for _, item := range items {
			productID := strings.TrimSpace(item.ID)
			if productID == "" {
				continue
			}
			if _, ok := linkedIDs[productID]; ok {
				continue
			}
			candidates = append(candidates, productID)
		}
	}

	deleteResult, err := s.vendure.DeleteProducts(ctx, candidates)
	if err != nil {
		return SupplierDuplicateCleanupResult{}, err
	}

	return SupplierDuplicateCleanupResult{
		ProductCount:    len(products),
		LinkedCount:     len(linkedIDs),
		DuplicateGroups: duplicateGroups,
		CandidateCount:  len(uniqueTrimmed(candidates)),
		DeletedCount:    deleteResult.Deleted,
		FailedCount:     deleteResult.Failed,
		DeletedIDs:      deleteResult.DeletedIDs,
		FailedIDs:       deleteResult.FailedIDs,
	}, nil
}

type ProcurementSubmitResponse struct {
	Summary ProcurementSummary             `json:"summary"`
	Results []supplier.PurchaseOrderResult `json:"results"`
}

type ProcurementPrecheckItem struct {
	SupplierCode  string  `json:"supplierCode"`
	OriginalSKU   string  `json:"originalSku"`
	SalesUnit     string  `json:"salesUnit,omitempty"`
	Quantity      float64 `json:"quantity,omitempty"`
	Status        string  `json:"status"`
	Message       string  `json:"message,omitempty"`
	AvailableQty  float64 `json:"availableQty,omitempty"`
	OldPrice      float64 `json:"oldPrice,omitempty"`
	NewPrice      float64 `json:"newPrice,omitempty"`
	BusinessPrice float64 `json:"businessPrice,omitempty"`
	ConsumerPrice float64 `json:"consumerPrice,omitempty"`
}

type ProcurementPrecheckResult struct {
	Ok        bool                      `json:"ok"`
	CheckedAt string                    `json:"checkedAt,omitempty"`
	Items     []ProcurementPrecheckItem `json:"items"`
}

type ProcurementStatusUpdateRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type ProcurementOrder struct {
	ID              string                         `json:"id"`
	ExternalRef     string                         `json:"externalRef"`
	Status          string                         `json:"status"`
	Connector       string                         `json:"connector"`
	Capabilities    supplier.ConnectorCapabilities `json:"capabilities"`
	DeliveryAddress string                         `json:"deliveryAddress,omitempty"`
	Notes           string                         `json:"notes,omitempty"`
	LastActionNote  string                         `json:"lastActionNote,omitempty"`
	SupplierCount   int                            `json:"supplierCount"`
	ItemCount       int                            `json:"itemCount"`
	TotalQty        float64                        `json:"totalQty"`
	TotalCostAmount float64                        `json:"totalCostAmount"`
	RiskyItemCount  int                            `json:"riskyItemCount"`
	Summary         ProcurementSummary             `json:"summary"`
	Results         []supplier.PurchaseOrderResult `json:"results,omitempty"`
	ExportCSV       string                         `json:"exportCsv,omitempty"`
	Created         string                         `json:"created,omitempty"`
	Updated         string                         `json:"updated,omitempty"`
	ReviewedAt      string                         `json:"reviewedAt,omitempty"`
	ExportedAt      string                         `json:"exportedAt,omitempty"`
	OrderedAt       string                         `json:"orderedAt,omitempty"`
	ReceivedAt      string                         `json:"receivedAt,omitempty"`
	CanceledAt      string                         `json:"canceledAt,omitempty"`
}

type ProcurementActionActor struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type ProcurementProgressLogger func(actionType string, status string, message string, details any)

type ProcurementActionLog struct {
	ID          string `json:"id"`
	OrderID     string `json:"orderId"`
	ExternalRef string `json:"externalRef"`
	ActionType  string `json:"actionType"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	ActorEmail  string `json:"actorEmail"`
	ActorName   string `json:"actorName"`
	Note        string `json:"note"`
	Created     string `json:"created"`
}

type ProcurementWorkbenchSummary struct {
	TotalOrders     int                    `json:"totalOrders"`
	DraftOrders     int                    `json:"draftOrders"`
	ReviewedOrders  int                    `json:"reviewedOrders"`
	ExportedOrders  int                    `json:"exportedOrders"`
	OrderedOrders   int                    `json:"orderedOrders"`
	ReceivedOrders  int                    `json:"receivedOrders"`
	CanceledOrders  int                    `json:"canceledOrders"`
	OpenRiskyOrders int                    `json:"openRiskyOrders"`
	OpenOrderCount  int                    `json:"openOrderCount"`
	RecentOrders    []ProcurementOrder     `json:"recentOrders"`
	RecentActions   []ProcurementActionLog `json:"recentActions"`
	Page            int                    `json:"page"`
	Pages           int                    `json:"pages"`
	PageSize        int                    `json:"pageSize"`
	FilterStatus    string                 `json:"filterStatus"`
	FilterRisk      string                 `json:"filterRisk"`
	Query           string                 `json:"query"`
}

type procurementCatalogItem struct {
	SupplierCode       string
	OriginalSKU        string
	Title              string
	NormalizedCategory string
	Quantity           float64
	SalesUnit          string
	CostPrice          float64
	BusinessPrice      float64
	ConsumerPrice      float64
	NeedColdChain      bool
}

type supplierProductSnapshot struct {
	Title            string                      `json:"title"`
	Category         string                      `json:"category"`
	ImageURL         string                      `json:"imageUrl"`
	GalleryURLs      []string                    `json:"galleryUrls,omitempty"`
	CostPrice        float64                     `json:"costPrice"`
	BPrice           float64                     `json:"bPrice"`
	CPrice           float64                     `json:"cPrice"`
	CurrencyCode     string                      `json:"currencyCode"`
	SourceProductID  string                      `json:"sourceProductId"`
	SourceType       string                      `json:"sourceType"`
	SalesUnit        string                      `json:"salesUnit"`
	ConversionRate   float64                     `json:"conversionRate"`
	DefaultStockQty  float64                     `json:"defaultStockQty"`
	DefaultStockText string                      `json:"defaultStockText"`
	UnitOptions      []supplierPayloadUnitOption `json:"unitOptions,omitempty"`
}

type supplierProductDiff struct {
	ContentChanged bool
	PriceChanged   bool
	StockChanged   bool
	SpecChanged    bool
}

type SourceAssetDownloadProgressLog struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type SourceAssetFailedItem struct {
	AssetID   string `json:"assetId"`
	AssetKey  string `json:"assetKey"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	AssetRole string `json:"assetRole"`
	Error     string `json:"error"`
}

type SourceAssetDownloadProgress struct {
	ID          string                           `json:"id"`
	Status      string                           `json:"status"`
	Mode        string                           `json:"mode"`
	Total       int                              `json:"total"`
	Processed   int                              `json:"processed"`
	Failed      int                              `json:"failed"`
	CurrentItem string                           `json:"currentItem"`
	StartedAt   string                           `json:"startedAt"`
	FinishedAt  string                           `json:"finishedAt"`
	Error       string                           `json:"error"`
	Logs        []SourceAssetDownloadProgressLog `json:"logs"`
	FailedItems []SourceAssetFailedItem          `json:"failedItems"`
	AssetIDs    []string                         `json:"assetIds"`
}

type SourceAssetProcessProgressLog struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type SourceAssetProcessProgress struct {
	ID          string                          `json:"id"`
	Status      string                          `json:"status"`
	Mode        string                          `json:"mode"`
	Total       int                             `json:"total"`
	Processed   int                             `json:"processed"`
	Failed      int                             `json:"failed"`
	CurrentItem string                          `json:"currentItem"`
	StartedAt   string                          `json:"startedAt"`
	FinishedAt  string                          `json:"finishedAt"`
	Error       string                          `json:"error"`
	Logs        []SourceAssetProcessProgressLog `json:"logs"`
	FailedItems []SourceAssetFailedItem         `json:"failedItems"`
	AssetIDs    []string                        `json:"assetIds"`
}

type SourceProductSyncProgressLog struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type SourceProductFailedItem struct {
	RecordID   string `json:"recordId"`
	ProductID  string `json:"productId"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	SyncStatus string `json:"syncStatus"`
	Error      string `json:"error"`
}

type SourceProductSyncProgress struct {
	ID          string                         `json:"id"`
	JobType     string                         `json:"jobType"`
	Mode        string                         `json:"mode"`
	Status      string                         `json:"status"`
	Total       int                            `json:"total"`
	Processed   int                            `json:"processed"`
	Failed      int                            `json:"failed"`
	CurrentItem string                         `json:"currentItem"`
	StartedAt   string                         `json:"startedAt"`
	FinishedAt  string                         `json:"finishedAt"`
	Error       string                         `json:"error"`
	Logs        []SourceProductSyncProgressLog `json:"logs"`
	FailedItems []SourceProductFailedItem      `json:"failedItems"`
	ProductIDs  []string                       `json:"productIds"`
}

type SupplierSyncProgress struct {
	ID          string                         `json:"id"`
	Status      string                         `json:"status"`
	Total       int                            `json:"total"`
	Processed   int                            `json:"processed"`
	Failed      int                            `json:"failed"`
	CurrentItem string                         `json:"currentItem"`
	StartedAt   string                         `json:"startedAt"`
	FinishedAt  string                         `json:"finishedAt"`
	Error       string                         `json:"error"`
	Logs        []SourceProductSyncProgressLog `json:"logs"`
}

type BackendCategoryMappingItem struct {
	ID                  string `json:"id"`
	SourceKey           string `json:"sourceKey"`
	Label               string `json:"label"`
	SourcePath          string `json:"sourcePath"`
	BackendCollection   string `json:"backendCollection"`
	BackendCollectionID string `json:"backendCollectionId"`
	BackendPath         string `json:"backendPath"`
	PublishStatus       string `json:"publishStatus"`
	LastError           string `json:"lastError"`
	Note                string `json:"note"`
	PublishedAt         string `json:"publishedAt"`
}

type BackendReleaseProductItem struct {
	ID                 string  `json:"id"`
	SupplierCode       string  `json:"supplierCode"`
	SKU                string  `json:"sku"`
	Title              string  `json:"title"`
	NormalizedCategory string  `json:"normalizedCategory"`
	TargetAudience     string  `json:"targetAudience"`
	ConversionRate     float64 `json:"conversionRate"`
	SyncStatus         string  `json:"syncStatus"`
	SupplierStatus     string  `json:"supplierStatus"`
	VendureProductID   string  `json:"vendureProductId"`
	VendureVariantID   string  `json:"vendureVariantId"`
	CreatedAt          string  `json:"createdAt"`
	SupplierUpdatedAt  string  `json:"supplierUpdatedAt"`
	UpdatedAt          string  `json:"updatedAt"`
	LastSyncedAt       string  `json:"lastSyncedAt"`
	LastSeenAt         string  `json:"lastSeenAt"`
	OfflineAt          string  `json:"offlineAt"`
	LastSyncError      string  `json:"lastSyncError"`
	Reason             string  `json:"reason"`
	HasProcessedImage  bool    `json:"hasProcessedImage"`
	HasConsumerImage   bool    `json:"hasConsumerImage"`
	ReadyForPreview    bool    `json:"readyForPreview"`
}

type BackendReleaseFilter struct {
	SyncStatus string `json:"syncStatus"`
	Query      string `json:"query"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	SortBy     string `json:"sortBy"`
	SortOrder  string `json:"sortOrder"`
}

type BackendCategoryBranchSummary struct {
	RootKey        string `json:"rootKey"`
	Label          string `json:"label"`
	TotalCount     int    `json:"totalCount"`
	PublishedCount int    `json:"publishedCount"`
	PendingCount   int    `json:"pendingCount"`
	ErrorCount     int    `json:"errorCount"`
}

type BackendCategoryMappingSuggestion struct {
	SourceKey            string `json:"sourceKey"`
	Label                string `json:"label"`
	SourcePath           string `json:"sourcePath"`
	SourceLevel          int    `json:"sourceLevel"`
	SuggestedCollection  string `json:"suggestedCollection"`
	SuggestedBackendPath string `json:"suggestedBackendPath"`
	Reason               string `json:"reason"`
}

type BackendReleaseSummary struct {
	CategoryCount        int                                `json:"categoryCount"`
	MappedCategoryCount  int                                `json:"mappedCategoryCount"`
	PublishedCount       int                                `json:"publishedCount"`
	PendingCategoryCount int                                `json:"pendingCategoryCount"`
	ErrorCategoryCount   int                                `json:"errorCategoryCount"`
	ProductCount         int                                `json:"productCount"`
	ReadyProductCount    int                                `json:"readyProductCount"`
	SyncedProductCount   int                                `json:"syncedProductCount"`
	ErrorProductCount    int                                `json:"errorProductCount"`
	OfflineProductCount  int                                `json:"offlineProductCount"`
	PublishedRootCount   int                                `json:"publishedRootCount"`
	FilteredProductCount int                                `json:"filteredProductCount"`
	ProductPage          int                                `json:"productPage"`
	ProductPages         int                                `json:"productPages"`
	ProductPageSize      int                                `json:"productPageSize"`
	Categories           []BackendCategoryMappingItem       `json:"categories"`
	Branches             []BackendCategoryBranchSummary     `json:"branches"`
	Products             []BackendReleaseProductItem        `json:"products"`
	SuggestedCategories  []BackendCategoryMappingSuggestion `json:"suggestedCategories"`
	RecommendedProducts  []BackendReleaseProductItem        `json:"recommendedProducts"`
}

type BackendReleasePayloadPreview struct {
	RecordID string         `json:"recordId"`
	Payload  map[string]any `json:"payload"`
}

type BackendCategoryPublishBatchResult struct {
	Requested    int                          `json:"requested"`
	Published    int                          `json:"published"`
	Failed       int                          `json:"failed"`
	RequestedIDs []string                     `json:"requestedIds"`
	Items        []BackendCategoryMappingItem `json:"items"`
	Errors       []string                     `json:"errors"`
}

type Service struct {
	cfg                config.Config
	connector          supplier.Connector
	miniappCartOrder   *miniappCartOrderSubmitter
	processor          image.Processor
	vendure            *vendure.Client
	lock               sync.Mutex
	targetSyncMu       sync.Mutex
	activeTargetSyncs  map[string]string
	sourceAssetMu      sync.Mutex
	activeAssetLoads   map[string]*SourceAssetDownloadProgress
	activeAssetProcs   map[string]*SourceAssetProcessProgress
	sourceProductMu    sync.Mutex
	activeProductJobs  map[string]*SourceProductSyncProgress
	supplierSyncMu     sync.Mutex
	activeSupplierSync *SupplierSyncProgress
	lastSupplierSync   *SupplierSyncProgress
}

func NewService(cfg config.Config) *Service {
	var connector supplier.Connector
	switch strings.ToLower(cfg.Supplier.Connector) {
	case "file":
		connector = supplier.NewFileConnector(cfg.Supplier.FilePath, cfg.Supplier.Code)
	case "http":
		connector = supplier.NewHTTPConnector(supplier.HTTPConnectorConfig{
			BaseURL:       cfg.Supplier.HTTPBaseURL,
			SubmitPath:    cfg.Supplier.HTTPSubmitPath,
			FetchPath:     cfg.Supplier.HTTPFetchPath,
			Token:         cfg.Supplier.HTTPToken,
			APIKey:        cfg.Supplier.HTTPAPIKey,
			SupplierCode:  cfg.Supplier.Code,
			Timeout:       cfg.Supplier.HTTPTimeout,
			SkipTLSVerify: cfg.Supplier.HTTPSkipTLSVerify,
		})
	case "miniapp_cart_order":
		connector = newMiniappHarvestConnector(cfg)
	default:
		connector = supplier.NewFileConnector(cfg.Supplier.FilePath, cfg.Supplier.Code)
	}

	return &Service{
		cfg:               cfg,
		connector:         connector,
		miniappCartOrder:  newMiniappCartOrderSubmitter(cfg),
		processor:         image.NewProcessor(cfg.Image),
		vendure:           vendure.NewClient(cfg.Vendure),
		activeTargetSyncs: make(map[string]string),
		activeAssetLoads:  make(map[string]*SourceAssetDownloadProgress),
		activeAssetProcs:  make(map[string]*SourceAssetProcessProgress),
		activeProductJobs: make(map[string]*SourceProductSyncProgress),
	}
}

func (s *Service) ConnectorCapabilities() supplier.ConnectorCapabilities {
	capabilities := s.connector.Capabilities()
	if s.miniappCartOrder == nil {
		return capabilities
	}

	submitCapabilities := s.miniappCartOrder.Capabilities()
	capabilities.FetchProducts = capabilities.FetchProducts || submitCapabilities.FetchProducts
	capabilities.SubmitPurchaseOrder = capabilities.SubmitPurchaseOrder || submitCapabilities.SubmitPurchaseOrder
	capabilities.ExportPurchaseOrder = capabilities.ExportPurchaseOrder || submitCapabilities.ExportPurchaseOrder
	return capabilities
}

func (s *Service) ProcurementSummary(ctx context.Context, app core.App, req ProcurementRequest) (ProcurementSummary, error) {
	items, err := s.resolveProcurementItems(ctx, app, req.Items)
	if err != nil {
		return ProcurementSummary{}, err
	}

	summary := buildProcurementSummary(
		s.cfg.Supplier.Connector,
		s.ConnectorCapabilities(),
		defaultProcurementExternalRef(req.ExternalRef),
		strings.TrimSpace(req.DeliveryAddress),
		strings.TrimSpace(req.Notes),
		items,
	)

	return summary, nil
}

func (s *Service) PrecheckProcurementItems(_ context.Context, app core.App, req ProcurementRequest) (ProcurementPrecheckResult, error) {
	if len(req.Items) == 0 {
		return ProcurementPrecheckResult{}, fmt.Errorf("procurement items are required")
	}

	items := make([]ProcurementPrecheckItem, 0, len(req.Items))
	allOK := true
	for _, requestItem := range req.Items {
		quantity := requestItem.Quantity
		if quantity <= 0 {
			return ProcurementPrecheckResult{}, fmt.Errorf("procurement quantity must be positive for sku %s", strings.TrimSpace(requestItem.OriginalSKU))
		}

		supplierCode := strings.TrimSpace(requestItem.SupplierCode)
		if supplierCode == "" {
			supplierCode = s.cfg.Supplier.Code
		}

		sku := strings.TrimSpace(requestItem.OriginalSKU)
		if sku == "" {
			return ProcurementPrecheckResult{}, fmt.Errorf("procurement sku is required")
		}

		itemResult := ProcurementPrecheckItem{
			SupplierCode: supplierCode,
			OriginalSKU:  sku,
			SalesUnit:    strings.TrimSpace(requestItem.SalesUnit),
			Quantity:     quantity,
			Status:       "ok",
		}

		record, err := app.FindFirstRecordByFilter(
			CollectionSupplierProducts,
			"supplier_code = {:supplier} && original_sku = {:sku}",
			dbx.Params{
				"supplier": supplierCode,
				"sku":      sku,
			},
		)
		if err != nil {
			itemResult.Status = "offline"
			itemResult.Message = "supplier item not found"
			items = append(items, itemResult)
			allOK = false
			continue
		}

		if record.GetString("sync_status") == StatusOffline || record.GetString("supplier_status") == SupplierStatusOffline {
			itemResult.Status = "offline"
			itemResult.Message = defaultString(strings.TrimSpace(record.GetString("offline_reason")), "supplier item offline")
			items = append(items, itemResult)
			allOK = false
			continue
		}

		requestedSalesUnit := strings.TrimSpace(requestItem.SalesUnit)
		unitOption, hasUnitOption := supplierRecordUnitOptionExact(record, requestedSalesUnit)
		defaultUnitOption := supplierRecordUnitOption(record, "")
		if requestedSalesUnit != "" && !hasUnitOption && supplierRecordHasUnitOptions(record) {
			itemResult.Status = "spec_mismatch"
			itemResult.Message = "requested sales unit is no longer available"
			items = append(items, itemResult)
			allOK = false
			continue
		}

		selectedOption := unitOption
		if !hasUnitOption {
			selectedOption = defaultUnitOption
		}
		itemResult.SalesUnit = defaultString(requestedSalesUnit, defaultString(selectedOption.UnitName, defaultString(readJSONAttribute(record, "sales_unit"), "件")))

		businessPrice := positiveOr(selectedOption.Price, record.GetFloat("b_price"))
		if businessPrice <= 0 {
			businessPrice = record.GetFloat("c_price")
		}
		consumerPrice := positiveOr(selectedOption.Price, record.GetFloat("c_price"))
		if consumerPrice <= 0 {
			consumerPrice = record.GetFloat("b_price")
		}
		itemResult.BusinessPrice = roundAmount(businessPrice)
		itemResult.ConsumerPrice = roundAmount(consumerPrice)

		if requestItem.ExpectedPrice > 0 {
			currentPrice := positiveOr(businessPrice, consumerPrice)
			if priceChanged(requestItem.ExpectedPrice, currentPrice) {
				itemResult.Status = "price_changed"
				itemResult.Message = "supplier price updated"
				itemResult.OldPrice = roundAmount(requestItem.ExpectedPrice)
				itemResult.NewPrice = roundAmount(currentPrice)
				allOK = false
			}
		}

		stockQty, hasStock := supplierRecordAvailableStock(record, itemResult.SalesUnit)
		itemResult.AvailableQty = roundAmount(stockQty)
		if hasStock {
			switch {
			case stockQty <= 0:
				itemResult.Status = "insufficient_stock"
				itemResult.Message = defaultString(strings.TrimSpace(selectedOption.StockText), "supplier out of stock")
				allOK = false
			case quantity > stockQty:
				itemResult.Status = "insufficient_stock"
				itemResult.Message = "supplier stock is insufficient"
				allOK = false
			}
		}

		items = append(items, itemResult)
	}

	return ProcurementPrecheckResult{
		Ok:        allOK,
		CheckedAt: time.Now().Format(time.RFC3339),
		Items:     items,
	}, nil
}

func (s *Service) ExportProcurement(ctx context.Context, app core.App, req ProcurementRequest) (ProcurementExport, error) {
	summary, err := s.ProcurementSummary(ctx, app, req)
	if err != nil {
		return ProcurementExport{}, err
	}

	content, err := renderProcurementCSV(summary)
	if err != nil {
		return ProcurementExport{}, err
	}

	fileName := slugify(summary.ExternalRef)
	if fileName == "" {
		fileName = "procurement"
	}

	return ProcurementExport{
		FileName:    fileName + ".csv",
		ContentType: "text/csv; charset=utf-8",
		RowCount:    summary.ItemCount,
		CSV:         content,
		Summary:     summary,
	}, nil
}

func (s *Service) SubmitProcurement(ctx context.Context, app core.App, req ProcurementRequest) (ProcurementSubmitResponse, error) {
	summary, err := s.ProcurementSummary(ctx, app, req)
	if err != nil {
		return ProcurementSubmitResponse{}, err
	}

	submitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
	defer cancel()

	results := s.submitProcurementSummary(submitCtx, app, summary, nil)

	return ProcurementSubmitResponse{
		Summary: summary,
		Results: results,
	}, nil
}

func (s *Service) SubmitProcurementOrder(ctx context.Context, app core.App, id string, note string) (ProcurementOrder, error) {
	return s.SubmitProcurementOrderWithAudit(ctx, app, id, note, ProcurementActionActor{})
}

func (s *Service) SubmitProcurementOrderWithAudit(ctx context.Context, app core.App, id string, note string, actor ProcurementActionActor) (ProcurementOrder, error) {
	record, err := app.FindRecordById(CollectionProcurementOrders, id)
	if err != nil {
		return ProcurementOrder{}, err
	}

	currentStatus := strings.TrimSpace(record.GetString("status"))
	if currentStatus == "" {
		currentStatus = ProcurementStatusDraft
	}
	if currentStatus == ProcurementStatusReceived || currentStatus == ProcurementStatusCanceled {
		s.logProcurementAction(app, record, "submit_order", "error", fmt.Sprintf("procurement order in status %s cannot be submitted", currentStatus), actor, strings.TrimSpace(note), map[string]any{
			"status": currentStatus,
		})
		return ProcurementOrder{}, fmt.Errorf("procurement order in status %s cannot be submitted", currentStatus)
	}

	summary, err := decodeProcurementSummary(record.GetString("summary_json"))
	if err != nil {
		s.logProcurementAction(app, record, "submit_order", "error", "decode procurement summary failed", actor, strings.TrimSpace(note), map[string]any{
			"error": err.Error(),
		})
		return ProcurementOrder{}, err
	}

	results := s.submitProcurementSummary(ctx, app, summary, func(actionType string, status string, message string, details any) {
		s.logProcurementAction(app, record, actionType, status, message, actor, strings.TrimSpace(note), details)
	})
	if err := setJSONField(record, "results_json", results); err != nil {
		s.logProcurementAction(app, record, "submit_order", "error", "persist procurement results failed", actor, strings.TrimSpace(note), map[string]any{
			"error": err.Error(),
		})
		return ProcurementOrder{}, err
	}

	accepted := 0
	for _, item := range results {
		if item.Accepted {
			accepted++
		}
	}

	nextStatus := currentStatus
	if accepted > 0 {
		nextStatus = ProcurementStatusOrdered
	} else if currentStatus == ProcurementStatusDraft || currentStatus == ProcurementStatusReviewed {
		nextStatus = ProcurementStatusExported
	}

	normalizedNote := strings.TrimSpace(note)
	if normalizedNote == "" {
		if accepted > 0 {
			normalizedNote = fmt.Sprintf("submitted to supplier: %d/%d accepted", accepted, len(results))
		} else {
			normalizedNote = "supplier submission attempted; no supplier accepted order, keep manual follow-up"
		}
	}

	if err := applyProcurementStatus(record, nextStatus, normalizedNote); err != nil {
		s.logProcurementAction(app, record, "submit_order", "error", "apply procurement status failed", actor, strings.TrimSpace(note), map[string]any{
			"error":  err.Error(),
			"status": nextStatus,
		})
		return ProcurementOrder{}, err
	}

	if err := app.Save(record); err != nil {
		s.logProcurementAction(app, record, "submit_order", "error", "save procurement order failed", actor, strings.TrimSpace(note), map[string]any{
			"error":  err.Error(),
			"status": nextStatus,
		})
		return ProcurementOrder{}, err
	}
	s.logProcurementAction(app, record, "submit_order", "success", "submitted procurement order to supplier connector", actor, normalizedNote, map[string]any{
		"accepted": accepted,
		"total":    len(results),
		"status":   nextStatus,
	})

	return procurementOrderFromRecord(record)
}

func (s *Service) CreateProcurementOrder(ctx context.Context, app core.App, req ProcurementRequest) (ProcurementOrder, error) {
	summary, err := s.ProcurementSummary(ctx, app, req)
	if err != nil {
		return ProcurementOrder{}, err
	}

	collection, err := app.FindCollectionByNameOrId(CollectionProcurementOrders)
	if err != nil {
		return ProcurementOrder{}, err
	}

	record := core.NewRecord(collection)
	record.Set("external_ref", summary.ExternalRef)
	record.Set("status", ProcurementStatusDraft)
	record.Set("connector", summary.Connector)
	record.Set("delivery_address", summary.DeliveryAddress)
	record.Set("notes", summary.Notes)
	record.Set("supplier_count", summary.SupplierCount)
	record.Set("item_count", summary.ItemCount)
	record.Set("total_qty", summary.TotalQty)
	record.Set("total_cost_amount", summary.TotalCostAmount)
	record.Set("risky_item_count", summary.RiskyItemCount)
	if err := setJSONField(record, "summary_json", summary); err != nil {
		return ProcurementOrder{}, err
	}
	if err := setJSONField(record, "results_json", []supplier.PurchaseOrderResult{}); err != nil {
		return ProcurementOrder{}, err
	}

	if err := app.Save(record); err != nil {
		return ProcurementOrder{}, err
	}
	s.logProcurementAction(app, record, "create_order", "success", "created procurement draft order", ProcurementActionActor{}, strings.TrimSpace(req.Notes), map[string]any{
		"itemCount": summary.ItemCount,
	})

	return procurementOrderFromRecord(record)
}

func (s *Service) submitProcurementSummary(ctx context.Context, app core.App, summary ProcurementSummary, progress ProcurementProgressLogger) []supplier.PurchaseOrderResult {
	if s.miniappCartOrder != nil {
		return s.miniappCartOrder.Submit(ctx, app, summary, progress)
	}
	results := make([]supplier.PurchaseOrderResult, 0, len(summary.Suppliers))
	for _, supplierSummary := range summary.Suppliers {
		if progress != nil {
			progress("submit_order_progress", "running", "submitting purchase order via connector", map[string]any{
				"supplierCode": supplierSummary.SupplierCode,
				"itemCount":    len(supplierSummary.Items),
			})
		}
		order := supplier.PurchaseOrder{
			SupplierCode:    supplierSummary.SupplierCode,
			ExternalRef:     summary.ExternalRef,
			DeliveryAddress: summary.DeliveryAddress,
			Notes:           summary.Notes,
			Items:           make([]supplier.PurchaseOrderItem, 0, len(supplierSummary.Items)),
		}

		for _, item := range supplierSummary.Items {
			order.Items = append(order.Items, supplier.PurchaseOrderItem{
				SupplierCode: item.SupplierCode,
				OriginalSKU:  item.OriginalSKU,
				Quantity:     item.Quantity,
				SalesUnit:    item.SalesUnit,
			})
		}

		result, submitErr := s.connector.SubmitPurchaseOrder(ctx, order)
		if submitErr != nil {
			if progress != nil {
				progress("submit_order_progress", "error", submitErr.Error(), map[string]any{
					"supplierCode": supplierSummary.SupplierCode,
				})
			}
			results = append(results, supplier.PurchaseOrderResult{
				SupplierCode: supplierSummary.SupplierCode,
				ExternalRef:  summary.ExternalRef,
				Mode:         "error",
				Accepted:     false,
				Message:      submitErr.Error(),
			})
			continue
		}
		if strings.TrimSpace(result.SupplierCode) == "" {
			result.SupplierCode = supplierSummary.SupplierCode
		}
		if strings.TrimSpace(result.ExternalRef) == "" {
			result.ExternalRef = summary.ExternalRef
		}
		if progress != nil {
			progress("submit_order_progress", "success", "supplier connector returned result", result)
		}

		results = append(results, result)
	}

	return results
}

func (s *Service) ListProcurementOrders(_ context.Context, app core.App, limit int, status string) ([]ProcurementOrder, error) {
	filter := ""
	params := dbx.Params{}
	if strings.TrimSpace(status) != "" {
		filter = "status = {:status}"
		params["status"] = strings.TrimSpace(status)
	}

	if limit <= 0 {
		limit = 20
	}

	sortExpr, err := procurementOrderSortExpr(app)
	if err != nil {
		return nil, err
	}

	records, err := app.FindRecordsByFilter(CollectionProcurementOrders, filter, sortExpr, limit, 0, params)
	if err != nil {
		return nil, err
	}

	result := make([]ProcurementOrder, 0, len(records))
	for _, record := range records {
		order, orderErr := procurementOrderFromRecord(record)
		if orderErr != nil {
			return nil, orderErr
		}
		result = append(result, order)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := procurementOrderPrimarySortTime(result[i])
		right := procurementOrderPrimarySortTime(result[j])
		if left != right {
			return right < left
		}
		return result[j].ID < result[i].ID
	})

	return result, nil
}

func procurementOrderSortExpr(app core.App) (string, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionProcurementOrders)
	if err != nil {
		return "", err
	}

	if collection.Fields.GetByName("ordered_at") != nil {
		return "-ordered_at,-exported_at,-reviewed_at,-received_at,-canceled_at,-id", nil
	}

	return "-id", nil
}

func procurementOrderPrimarySortTime(order ProcurementOrder) string {
	for _, value := range []string{
		strings.TrimSpace(order.OrderedAt),
		strings.TrimSpace(order.ExportedAt),
		strings.TrimSpace(order.ReviewedAt),
		strings.TrimSpace(order.ReceivedAt),
		strings.TrimSpace(order.CanceledAt),
	} {
		if value != "" {
			return value
		}
	}
	return ""
}

func supplierProductSortExpr(app core.App) (string, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionSupplierProducts)
	if err != nil {
		return "", err
	}

	if collection.Fields.GetByName("updated") != nil {
		return "-updated", nil
	}
	if collection.Fields.GetByName("created") != nil {
		return "-created", nil
	}
	return "-id", nil
}

func (s *Service) GetProcurementOrder(_ context.Context, app core.App, id string) (ProcurementOrder, error) {
	record, err := app.FindRecordById(CollectionProcurementOrders, id)
	if err != nil {
		return ProcurementOrder{}, err
	}

	return procurementOrderFromRecord(record)
}

func (s *Service) ReviewProcurementOrder(ctx context.Context, app core.App, id string, note string) (ProcurementOrder, error) {
	return s.ReviewProcurementOrderWithAudit(ctx, app, id, note, ProcurementActionActor{})
}

func (s *Service) ReviewProcurementOrderWithAudit(ctx context.Context, app core.App, id string, note string, actor ProcurementActionActor) (ProcurementOrder, error) {
	return s.UpdateProcurementOrderStatusWithAudit(ctx, app, id, ProcurementStatusReviewed, note, actor)
}

func (s *Service) ExportProcurementOrder(ctx context.Context, app core.App, id string) (ProcurementOrder, error) {
	return s.ExportProcurementOrderWithAudit(ctx, app, id, ProcurementActionActor{}, "")
}

func (s *Service) ExportProcurementOrderWithAudit(ctx context.Context, app core.App, id string, actor ProcurementActionActor, note string) (ProcurementOrder, error) {
	record, err := app.FindRecordById(CollectionProcurementOrders, id)
	if err != nil {
		return ProcurementOrder{}, err
	}

	summary, err := decodeProcurementSummary(record.GetString("summary_json"))
	if err != nil {
		return ProcurementOrder{}, err
	}

	content, err := renderProcurementCSV(summary)
	if err != nil {
		return ProcurementOrder{}, err
	}

	record.Set("export_csv", content)
	if err := applyProcurementStatus(record, ProcurementStatusExported, "exported csv generated"); err != nil {
		return ProcurementOrder{}, err
	}
	if strings.TrimSpace(note) != "" {
		record.Set("last_action_note", strings.TrimSpace(note))
	}

	if err := app.Save(record); err != nil {
		return ProcurementOrder{}, err
	}
	s.logProcurementAction(app, record, "export_order", "success", "exported procurement csv", actor, note, nil)

	return procurementOrderFromRecord(record)
}

func (s *Service) UpdateProcurementOrderStatus(_ context.Context, app core.App, id string, status string, note string) (ProcurementOrder, error) {
	return s.UpdateProcurementOrderStatusWithAudit(context.Background(), app, id, status, note, ProcurementActionActor{})
}

func (s *Service) UpdateProcurementOrderStatusWithAudit(_ context.Context, app core.App, id string, status string, note string, actor ProcurementActionActor) (ProcurementOrder, error) {
	record, err := app.FindRecordById(CollectionProcurementOrders, id)
	if err != nil {
		return ProcurementOrder{}, err
	}

	if err := applyProcurementStatus(record, status, note); err != nil {
		return ProcurementOrder{}, err
	}

	if err := app.Save(record); err != nil {
		return ProcurementOrder{}, err
	}
	s.logProcurementAction(app, record, "update_status", "success", "updated procurement order status", actor, note, map[string]any{
		"status": status,
	})

	return procurementOrderFromRecord(record)
}

func (s *Service) ProcurementWorkbenchSummary(ctx context.Context, app core.App, limit int) (ProcurementWorkbenchSummary, error) {
	return s.ProcurementWorkbenchSummaryFiltered(ctx, app, limit, "", "", "", 1)
}

func (s *Service) ProcurementWorkbenchSummaryFiltered(ctx context.Context, app core.App, limit int, status string, risk string, query string, page int) (ProcurementWorkbenchSummary, error) {
	orders, err := s.ListProcurementOrders(ctx, app, 200, "")
	if err != nil {
		return ProcurementWorkbenchSummary{}, err
	}

	summary := ProcurementWorkbenchSummary{
		Page:         page,
		PageSize:     limit,
		FilterStatus: strings.TrimSpace(status),
		FilterRisk:   strings.TrimSpace(risk),
		Query:        strings.TrimSpace(query),
	}
	filteredOrders := make([]ProcurementOrder, 0, len(orders))
	query = strings.ToLower(strings.TrimSpace(query))
	for _, order := range orders {
		summary.TotalOrders++
		if order.Status != ProcurementStatusReceived && order.Status != ProcurementStatusCanceled {
			summary.OpenOrderCount++
			if order.RiskyItemCount > 0 {
				summary.OpenRiskyOrders++
			}
		}

		switch order.Status {
		case ProcurementStatusDraft:
			summary.DraftOrders++
		case ProcurementStatusReviewed:
			summary.ReviewedOrders++
		case ProcurementStatusExported:
			summary.ExportedOrders++
		case ProcurementStatusOrdered:
			summary.OrderedOrders++
		case ProcurementStatusReceived:
			summary.ReceivedOrders++
		case ProcurementStatusCanceled:
			summary.CanceledOrders++
		}

		if summary.FilterStatus != "" && !strings.EqualFold(order.Status, summary.FilterStatus) {
			continue
		}
		if !matchProcurementRiskFilter(order, summary.FilterRisk) {
			continue
		}
		if query != "" {
			search := strings.ToLower(strings.Join([]string{order.ExternalRef, order.ID, order.LastActionNote}, " "))
			if !strings.Contains(search, query) {
				continue
			}
		}
		filteredOrders = append(filteredOrders, order)
	}
	if summary.Page <= 0 {
		summary.Page = 1
	}
	if summary.PageSize <= 0 {
		summary.PageSize = 20
	}
	summary.Pages = 1
	if len(filteredOrders) > 0 {
		summary.Pages = len(filteredOrders) / summary.PageSize
		if len(filteredOrders)%summary.PageSize != 0 {
			summary.Pages++
		}
		if summary.Pages <= 0 {
			summary.Pages = 1
		}
	}
	start := (summary.Page - 1) * summary.PageSize
	if start > len(filteredOrders) {
		start = len(filteredOrders)
	}
	end := start + summary.PageSize
	if end > len(filteredOrders) {
		end = len(filteredOrders)
	}
	summary.RecentOrders = filteredOrders[start:end]
	summary.RecentActions, _ = s.listRecentProcurementActions(app, limit)

	return summary, nil
}

func matchProcurementRiskFilter(order ProcurementOrder, risk string) bool {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "", "all":
		return true
	case "has_risk":
		return order.RiskyItemCount > 0
	case "loss":
		return procurementSummaryHasRiskLevel(order.Summary, "loss")
	case "warning":
		return procurementSummaryHasRiskLevel(order.Summary, "warning")
	case "normal":
		return order.RiskyItemCount == 0
	default:
		return true
	}
}

func procurementSummaryHasRiskLevel(summary ProcurementSummary, level string) bool {
	for _, supplier := range summary.Suppliers {
		for _, item := range supplier.Items {
			if strings.EqualFold(item.RiskLevel, level) {
				return true
			}
		}
	}
	return false
}

func (s *Service) Harvest(ctx context.Context, app core.App) (Result, error) {
	return s.HarvestWithOptions(ctx, app, HarvestOptions{TriggerType: HarvestTriggerAPI})
}

func (s *Service) harvestExecutionTimeout() time.Duration {
	timeout := s.cfg.MiniApp.SourceTimeout * 12
	if timeout < 5*time.Minute {
		timeout = 5 * time.Minute
	}
	if timeout > 30*time.Minute {
		timeout = 30 * time.Minute
	}
	return timeout
}

func (s *Service) StartHarvest(ctx context.Context, app core.App, options HarvestOptions) (HarvestRun, bool, error) {
	s.lock.Lock()
	runningRun, err := FindRunningHarvestRun(app)
	if err != nil {
		s.lock.Unlock()
		return HarvestRun{}, false, err
	}
	if strings.TrimSpace(runningRun.ID) != "" {
		s.lock.Unlock()
		return runningRun, true, nil
	}

	runRecord, err := s.createHarvestRun(app, options)
	if err != nil {
		s.lock.Unlock()
		return HarvestRun{}, false, err
	}
	run := harvestRunFromRecord(runRecord)
	s.lock.Unlock()

	go func(runRecord *core.Record) {
		s.lock.Lock()
		defer s.lock.Unlock()

		startedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(runRecord.GetString("started_at")))
		if parseErr != nil || startedAt.IsZero() {
			startedAt = time.Now()
		}
		state := harvestExecutionState{
			Result:           Result{Action: "harvest"},
			startedAt:        startedAt,
			lastPersistAt:    startedAt,
			lastPersistCount: 0,
		}
		defer func() {
			if _, err := s.finalizeHarvestRun(app, runRecord, state); err != nil {
				app.Logger().Error("finalize harvest run failed", "error", err)
			}
		}()

		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.harvestExecutionTimeout())
		defer cancel()
		if err := s.executeHarvest(runCtx, app, runRecord, &state); err != nil {
			app.Logger().Error("harvest failed", "error", err)
		}
	}(runRecord)

	return run, false, nil
}

func (s *Service) HarvestWithOptions(ctx context.Context, app core.App, options HarvestOptions) (Result, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	state := harvestExecutionState{
		Result:           Result{Action: "harvest"},
		startedAt:        time.Now(),
		lastPersistAt:    time.Now(),
		lastPersistCount: 0,
	}
	runRecord, runErr := s.createHarvestRun(app, options)
	if runErr != nil {
		app.Logger().Error("create harvest run failed", "error", runErr)
	}
	defer func() {
		if runRecord == nil {
			return
		}
		if _, err := s.finalizeHarvestRun(app, runRecord, state); err != nil {
			app.Logger().Error("finalize harvest run failed", "error", err)
		}
	}()

	runCtx, cancel := context.WithTimeout(ctx, s.harvestExecutionTimeout())
	defer cancel()
	if err := s.executeHarvest(runCtx, app, runRecord, &state); err != nil {
		return state.Result, err
	}

	return state.Result, nil
}

func (s *Service) executeHarvest(ctx context.Context, app core.App, runRecord *core.Record, state *harvestExecutionState) error {
	items, err := s.connector.Fetch(ctx)
	if err != nil {
		state.errorMessage = err.Error()
		return err
	}

	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		key := s.recordKey(item.SupplierCode, item.OriginalSKU)
		seen[key] = struct{}{}

		changed, created, err := s.upsertSupplierProduct(ctx, app, item)
		if err != nil {
			state.Failed++
			state.failureItems = appendHarvestFailure(state.failureItems, HarvestFailureItem{
				SKU:   item.OriginalSKU,
				Step:  "upsert",
				Error: err.Error(),
			})
			if progressErr := s.updateHarvestRunProgress(app, runRecord, state, false); progressErr != nil {
				app.Logger().Error("persist harvest progress failed", "error", progressErr)
			}
			app.Logger().Error("harvest upsert failed", "sku", item.OriginalSKU, "error", err)
			continue
		}

		state.Processed++
		if created {
			state.Created++
			continue
		}

		if changed {
			state.Updated++
		} else {
			state.Skipped++
		}
		if progressErr := s.updateHarvestRunProgress(app, runRecord, state, false); progressErr != nil {
			app.Logger().Error("persist harvest progress failed", "error", progressErr)
		}
	}

	offlineSummary, err := s.markMissingProductsOffline(ctx, app, seen)
	if err != nil {
		state.errorMessage = err.Error()
		return err
	}
	state.Offline = offlineSummary.OfflineCount
	state.Failed += len(offlineSummary.FailureItems)
	state.failureItems = truncateHarvestFailureItems(append(state.failureItems, offlineSummary.FailureItems...))
	if progressErr := s.updateHarvestRunProgress(app, runRecord, state, true); progressErr != nil {
		app.Logger().Error("persist harvest progress failed", "error", progressErr)
	}
	return nil
}

func (s *Service) ProcessPending(ctx context.Context, app core.App, limit int) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"(sync_status = {:pending} || sync_status = {:error}) && raw_image_url != ''",
		"",
		limit,
		0,
		dbx.Params{
			"pending": StatusPending,
			"error":   StatusError,
		},
	)
	if err != nil {
		return Result{Action: "process"}, err
	}
	slices.SortFunc(records, func(left, right *core.Record) int {
		leftPriority := supplierProcessPriority(left)
		rightPriority := supplierProcessPriority(right)
		if leftPriority != rightPriority {
			return leftPriority - rightPriority
		}
		return strings.Compare(strings.TrimSpace(left.GetString("updated")), strings.TrimSpace(right.GetString("updated")))
	})

	result := Result{Action: "process"}
	for _, record := range records {
		if err := s.processRecord(ctx, app, record); err != nil {
			result.Failed++
			app.Logger().Error("image processing failed", "recordId", record.Id, "error", err)
			continue
		}

		result.Processed++
	}

	return result, nil
}

func (s *Service) ProcessRecord(ctx context.Context, app core.App, recordID string) error {
	record, err := app.FindRecordById(CollectionSupplierProducts, recordID)
	if err != nil {
		return err
	}

	return s.processRecord(ctx, app, record)
}

func (s *Service) readySyncStatus() string {
	if s.cfg.Workflow.AutoApproveReady {
		return StatusApproved
	}
	return StatusReady
}

func (s *Service) AdvanceProcessedPending(ctx context.Context, app core.App, limit int) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"sync_status = {:pending} && image_processing_status = {:processed}",
		"",
		limit,
		0,
		dbx.Params{
			"pending":   StatusPending,
			"processed": ImageStatusProcessed,
		},
	)
	if err != nil {
		return Result{Action: "advance"}, err
	}
	slices.SortFunc(records, func(left, right *core.Record) int {
		return strings.Compare(strings.TrimSpace(left.GetString("updated")), strings.TrimSpace(right.GetString("updated")))
	})

	result := Result{Action: "advance"}
	for _, record := range records {
		record.Set("sync_status", s.readySyncStatus())
		if err := app.Save(record); err != nil {
			result.Failed++
			app.Logger().Error("advance supplier product failed", "recordId", record.Id, "error", err)
			continue
		}
		result.Processed++
	}

	return result, nil
}

func (s *Service) ApproveReady(ctx context.Context, app core.App, limit int) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"sync_status = {:status}",
		"",
		limit,
		0,
		dbx.Params{"status": StatusReady},
	)
	if err != nil {
		return Result{Action: "approve"}, err
	}
	slices.SortFunc(records, func(left, right *core.Record) int {
		return strings.Compare(strings.TrimSpace(left.GetString("updated")), strings.TrimSpace(right.GetString("updated")))
	})

	result := Result{Action: "approve"}
	for _, record := range records {
		record.Set("sync_status", StatusApproved)
		if err := app.Save(record); err != nil {
			result.Failed++
			app.Logger().Error("approve supplier product failed", "recordId", record.Id, "error", err)
			continue
		}
		result.Processed++
	}

	return result, nil
}

func (s *Service) ScanSyncedSingleImageCandidates(ctx context.Context, app core.App, limit int) ([]SupplierSingleImageCandidate, error) {
	_ = ctx
	if limit <= 0 {
		limit = 200
	}
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"sync_status = {:synced} && vendure_product_id != ''",
		"",
		10000,
		0,
		dbx.Params{"synced": StatusSynced},
	)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(records, func(left, right *core.Record) int {
		return strings.Compare(strings.TrimSpace(right.GetString("updated")), strings.TrimSpace(left.GetString("updated")))
	})
	items := make([]SupplierSingleImageCandidate, 0, minInt(limit, len(records)))
	for _, record := range records {
		if !isSyncedSingleImageSkeletonRecord(record) {
			continue
		}
		items = append(items, SupplierSingleImageCandidate{
			ID:               record.Id,
			SupplierCode:     strings.TrimSpace(record.GetString("supplier_code")),
			OriginalSKU:      strings.TrimSpace(record.GetString("original_sku")),
			Title:            strings.TrimSpace(displayTitle(record)),
			SourceType:       strings.TrimSpace(defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type"))),
			SyncStatus:       strings.TrimSpace(record.GetString("sync_status")),
			VendureProductID: strings.TrimSpace(record.GetString("vendure_product_id")),
			GalleryCount:     len(supplierRecordGalleryURLs(record)),
			UpdatedAt:        strings.TrimSpace(record.GetString("updated")),
		})
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *Service) RequeueSingleImageCandidatesByIDs(ctx context.Context, app core.App, ids []string) (Result, error) {
	_ = ctx
	result := Result{Action: "requeue_single_image"}
	normalized := uniqueTrimmed(ids)
	for _, id := range normalized {
		record, err := app.FindRecordById(CollectionSupplierProducts, id)
		if err != nil {
			result.Failed++
			app.Logger().Error("requeue single image supplier product load failed", "recordId", id, "error", err)
			continue
		}
		if !isSyncedSingleImageSkeletonRecord(record) {
			result.Skipped++
			continue
		}
		record.Set("sync_status", StatusApproved)
		record.Set("last_sync_error", "")
		if err := app.Save(record); err != nil {
			result.Failed++
			app.Logger().Error("requeue single image supplier product failed", "recordId", record.Id, "error", err)
			continue
		}
		result.Processed++
	}
	return result, nil
}

func (s *Service) SyncApproved(ctx context.Context, app core.App, limit int) (Result, error) {
	return s.syncApprovedWithProgress(ctx, app, limit, nil)
}

func (s *Service) syncApprovedWithProgress(ctx context.Context, app core.App, limit int, onProgress func(total int, processed int, failed int, current string)) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"sync_status = {:status}",
		"",
		limit,
		0,
		dbx.Params{"status": StatusApproved},
	)
	if err != nil {
		return Result{Action: "sync"}, err
	}
	slices.SortFunc(records, func(left, right *core.Record) int {
		return strings.Compare(strings.TrimSpace(right.GetString("updated")), strings.TrimSpace(left.GetString("updated")))
	})
	if onProgress != nil {
		onProgress(len(records), 0, 0, "")
	}

	result := Result{Action: "sync"}
	for _, record := range records {
		current := strings.TrimSpace(record.GetString("supplier_code")) + "/" + strings.TrimSpace(record.GetString("original_sku"))
		if onProgress != nil {
			onProgress(len(records), result.Processed, result.Failed, current)
		}
		if err := s.syncRecord(ctx, app, record); err != nil {
			result.Failed++
			app.Logger().Error("vendure sync failed", "recordId", record.Id, "error", err)
			if onProgress != nil {
				onProgress(len(records), result.Processed, result.Failed, current)
			}
			continue
		}

		result.Processed++
		if onProgress != nil {
			onProgress(len(records), result.Processed, result.Failed, current)
		}
	}

	return result, nil
}

func (s *Service) StartSupplierSyncAsync(ctx context.Context, app core.App, limit int) (SupplierSyncProgress, bool, error) {
	runningRecord, err := findRunningSupplierSyncRun(app)
	if err != nil {
		return SupplierSyncProgress{}, false, err
	}
	if runningRecord != nil {
		return supplierSyncProgressFromRecord(runningRecord), true, nil
	}
	runRecord, err := s.createSupplierSyncRun(app)
	if err != nil {
		return SupplierSyncProgress{}, false, err
	}
	progress := supplierSyncProgressFromRecord(runRecord)

	go func() {
		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Minute)
		defer cancel()
		finalResult := Result{Action: "sync"}
		finalErr := ""
		result, err := s.syncApprovedWithProgress(runCtx, app, limit, func(total int, processed int, failed int, current string) {
			if progressErr := s.updateSupplierSyncRunProgress(app, runRecord, total, processed, failed, current); progressErr != nil {
				app.Logger().Error("persist supplier sync progress failed", "error", progressErr)
			}
		})
		finalResult = result
		if err != nil {
			finalErr = err.Error()
		}
		if finalizeErr := s.finalizeSupplierSyncRun(app, runRecord, runRecord.GetInt("total_count"), finalResult.Processed, finalResult.Failed, finalErr); finalizeErr != nil {
			app.Logger().Error("finalize supplier sync run failed", "error", finalizeErr)
		}
	}()

	return progress, false, nil
}

func (s *Service) SupplierSyncProgress(app core.App) SupplierSyncProgress {
	record, err := latestSupplierSyncRun(app)
	if err != nil || record == nil {
		return SupplierSyncProgress{}
	}
	return supplierSyncProgressFromRecord(record)
}

func (s *Service) SyncRecord(ctx context.Context, app core.App, recordID string) error {
	record, err := app.FindRecordById(CollectionSupplierProducts, recordID)
	if err != nil {
		return err
	}

	return s.syncRecord(ctx, app, record)
}

func (s *Service) upsertSupplierProduct(ctx context.Context, app core.App, item supplier.Product) (bool, bool, error) {
	record, err := app.FindFirstRecordByFilter(
		CollectionSupplierProducts,
		"supplier_code = {:supplier} && original_sku = {:sku}",
		dbx.Params{
			"supplier": item.SupplierCode,
			"sku":      item.OriginalSKU,
		},
	)

	created := false
	if err != nil {
		collection, findErr := app.FindCollectionByNameOrId(CollectionSupplierProducts)
		if findErr != nil {
			return false, false, findErr
		}
		record = core.NewRecord(collection)
		created = true
	}

	previousSnapshot := supplierProductSnapshotFromRecord(record)
	normalizedCategory, _ := s.resolveCategory(ctx, app, item.SupplierCode, item.RawCategory)

	record.Set("supplier_code", item.SupplierCode)
	record.Set("original_sku", item.OriginalSKU)
	record.Set("raw_title", item.RawTitle)
	record.Set("normalized_title", item.RawTitle)
	record.Set("raw_description", item.RawDescription)
	record.Set("raw_category", item.RawCategory)
	record.Set("normalized_category", normalizedCategory)
	record.Set("raw_image_url", item.RawImageURL)
	record.Set("cost_price", item.CostPrice)
	record.Set("b_price", item.BPrice)
	record.Set("c_price", item.CPrice)
	record.Set("currency_code", defaultString(item.CurrencyCode, "CNY"))
	record.Set("supplier_updated_at", item.SupplierUpdatedAt.Format(time.RFC3339))
	record.Set("last_seen_at", time.Now().Format(time.RFC3339))
	record.Set("offline_reason", "")

	if payload, err := json.Marshal(item.Payload); err == nil {
		record.Set("supplier_payload", string(payload))
	}
	record.Set("source_product_id", readJSONAttribute(record, "source_product_id"))
	record.Set("source_type", readJSONAttribute(record, "source_type"))

	nextSnapshot := supplierProductSnapshotFromRecord(record)
	diff := diffSupplierProductSnapshots(previousSnapshot, nextSnapshot)
	snapshotHash, err := supplierProductSnapshotHash(nextSnapshot)
	if err != nil {
		return false, created, err
	}
	if err := setJSONField(record, "last_snapshot_json", nextSnapshot); err != nil {
		return false, created, err
	}
	record.Set("source_snapshot_hash", snapshotHash)
	record.Set("supplier_status", supplierStatusFromSnapshot(nextSnapshot, diff))

	if created {
		record.Set("sync_status", StatusPending)
		record.Set("image_processing_status", ImageStatusPending)
		record.Set("last_price_sync_at", time.Now().Format(time.RFC3339))
		record.Set("last_stock_sync_at", time.Now().Format(time.RFC3339))
	} else if diff.hasChanges() {
		hasVendureProduct := strings.TrimSpace(record.GetString("vendure_product_id")) != ""
		switch {
		case diff.SpecChanged:
			if hasVendureProduct {
				record.Set("sync_status", s.readySyncStatus())
			} else {
				record.Set("sync_status", StatusPending)
			}
		case diff.ContentChanged:
			if hasVendureProduct {
				record.Set("sync_status", StatusApproved)
			} else {
				record.Set("sync_status", StatusPending)
			}
			record.Set("image_processing_status", ImageStatusPending)
		case diff.PriceChanged || diff.StockChanged:
			if hasVendureProduct {
				record.Set("sync_status", StatusApproved)
			} else {
				record.Set("sync_status", StatusPending)
			}
		}
		if diff.PriceChanged {
			record.Set("last_price_sync_at", time.Now().Format(time.RFC3339))
		}
		if diff.StockChanged {
			record.Set("last_stock_sync_at", time.Now().Format(time.RFC3339))
		}
		record.Set("last_sync_error", "")
		if diff.ContentChanged {
			record.Set("image_processing_error", "")
		}
	}

	if err := app.Save(record); err != nil {
		return false, created, err
	}

	return diff.hasChanges(), created, nil
}

func (s *Service) processRecord(ctx context.Context, app core.App, record *core.Record) error {
	record.Set("sync_status", StatusAIProcessing)
	record.Set("image_processing_status", ImageStatusProcessing)
	record.Set("image_processing_error", "")
	if err := app.Save(record); err != nil {
		return err
	}

	result, err := s.processor.Process(ctx, image.Request{
		SupplierCode: record.GetString("supplier_code"),
		SKU:          record.GetString("original_sku"),
		Title:        displayTitle(record),
		SourceURL:    record.GetString("raw_image_url"),
	})
	if err != nil {
		record.Set("sync_status", StatusError)
		record.Set("image_processing_status", ImageStatusFailed)
		record.Set("image_processing_error", err.Error())
		return app.Save(record)
	}

	record.Set("processed_image", result.File)
	record.Set("image_processing_status", ImageStatusProcessed)
	record.Set("image_processing_error", "")
	record.Set("sync_status", s.readySyncStatus())
	record.Set("processed_image_source", result.Source)
	return app.Save(record)
}

func (s *Service) syncRecord(ctx context.Context, app core.App, record *core.Record) error {
	if err := s.backfillMiniappCarouselIfNeeded(ctx, app, record); err != nil {
		app.Logger().Error("miniapp carousel backfill skipped", "recordId", record.Id, "error", err)
	}
	payload := s.buildVendurePayload(app, record)

	result, err := s.vendure.SyncProduct(ctx, payload)
	if err != nil {
		record.Set("last_sync_error", err.Error())
		record.Set("sync_status", StatusError)
		_ = app.Save(record)
		return err
	}

	productID := coalesce(result.ProductID, record.GetString("vendure_product_id"))
	variantID := coalesce(result.VariantID, record.GetString("vendure_variant_id"))
	if err := s.syncVendureCollections(ctx, app, record, productID); err != nil {
		record.Set("last_sync_error", err.Error())
		record.Set("sync_status", StatusError)
		_ = app.Save(record)
		return err
	}

	record.Set("vendure_product_id", productID)
	record.Set("vendure_variant_id", variantID)
	if err := setJSONField(record, "vendure_variants_json", result.Variants); err != nil {
		record.Set("last_sync_error", err.Error())
		record.Set("sync_status", StatusError)
		_ = app.Save(record)
		return err
	}
	record.Set("last_sync_error", "")
	record.Set("sync_status", StatusSynced)
	record.Set("last_synced_at", time.Now().Format(time.RFC3339))
	return app.Save(record)
}

func (s *Service) backfillMiniappCarouselIfNeeded(ctx context.Context, app core.App, record *core.Record) error {
	if !strings.EqualFold(strings.TrimSpace(s.cfg.Supplier.Connector), "miniapp_cart_order") {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(s.cfg.MiniApp.SourceMode), "raw") {
		return nil
	}

	sourceType := strings.ToLower(strings.TrimSpace(defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type"))))
	if sourceType == "raw_detail" || sourceType == "rr_detail" {
		return nil
	}
	if len(supplierRecordGalleryURLs(record)) > 1 {
		return nil
	}

	sourceProductID := strings.TrimSpace(defaultString(record.GetString("source_product_id"), readJSONAttribute(record, "source_product_id")))
	spuID, skuID := splitSourceProductID(sourceProductID)
	if strings.TrimSpace(spuID) == "" || strings.TrimSpace(skuID) == "" {
		return nil
	}

	miniapp := miniappservice.New(newMiniappActionSource(s.cfg), nil)
	resolveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
	defer cancel()

	product, err := miniapp.ResolveProduct(resolveCtx, spuID, skuID)
	if err != nil {
		return fmt.Errorf("resolve product %s/%s: %w", spuID, skuID, err)
	}
	if product == nil {
		return nil
	}

	galleryURLs := miniappGalleryURLs(s.cfg.MiniApp.RawAssetBaseURL, *product)
	if len(galleryURLs) <= 1 {
		return nil
	}

	payload := make(map[string]any)
	if raw := strings.TrimSpace(record.GetString("supplier_payload")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &payload)
	}
	payload["gallery_urls"] = galleryURLs
	if sourceType := strings.TrimSpace(product.SourceType); sourceType != "" {
		payload["source_type"] = sourceType
		record.Set("source_type", sourceType)
	}
	payload["source_product_id"] = sourceProductID
	record.Set("source_product_id", sourceProductID)

	coverURL := sanitizeURLWithBase(firstNonEmptyString(
		strings.TrimSpace(product.Summary.Cover),
		firstMiniappImageURL(product.Detail.Carousel),
		firstMiniappImageURL(product.Detail.DetailAssets),
	), s.cfg.MiniApp.RawAssetBaseURL)
	if coverURL != "" {
		record.Set("raw_image_url", coverURL)
	}

	if err := setJSONField(record, "supplier_payload", payload); err != nil {
		return err
	}
	return app.Save(record)
}

func (s *Service) buildVendurePayload(app core.App, record *core.Record) vendure.ProductPayload {
	assetURL := s.recordPrimaryAssetURL(record)
	cEndAssetURL := s.recordConsumerAssetURL(record)
	assetURLs, assetNames := s.recordGalleryAssetURLs(app, record)
	if assetURL == "" && len(assetURLs) > 0 {
		assetURL = assetURLs[0]
	}
	variants := supplierRecordVariantPayloads(record, s.cfg.Workflow.DefaultStockOnHand)
	defaultVariant := defaultVariantPayload(variants)

	return vendure.ProductPayload{
		Name:              displayTitle(record),
		Slug:              slugify(record.GetString("supplier_code") + "-" + record.GetString("original_sku") + "-" + displayTitle(record)),
		Description:       defaultString(record.GetString("marketing_description"), record.GetString("raw_description")),
		SKU:               defaultVariant.SKU,
		CurrencyCode:      defaultString(record.GetString("currency_code"), s.cfg.Vendure.CurrencyCode),
		ConsumerPrice:     defaultVariant.ConsumerPrice,
		AssetURL:          assetURL,
		AssetName:         assetFileName(assetURL),
		AssetURLs:         assetURLs,
		AssetNames:        assetNames,
		CEndAssetURL:      cEndAssetURL,
		CEndAssetName:     assetFileName(cEndAssetURL),
		BusinessPrice:     defaultVariant.BusinessPrice,
		SupplierCode:      record.GetString("supplier_code"),
		SupplierCostPrice: toMinorUnits(record.GetFloat("cost_price")),
		ConversionRate:    defaultVariant.ConversionRate,
		SourceProductID:   defaultString(record.GetString("source_product_id"), readJSONAttribute(record, "source_product_id")),
		SourceType:        defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type")),
		TargetAudience:    defaultString(record.GetString("target_audience"), "ALL"),
		DefaultStock:      defaultVariant.DefaultStock,
		SalesUnit:         defaultVariant.SalesUnit,
		VendureProduct:    record.GetString("vendure_product_id"),
		VendureVariant:    defaultVariant.VendureVariant,
		NeedColdChain:     strings.EqualFold(readJSONAttribute(record, "need_cold_chain"), "true"),
		Variants:          variants,
	}
}

func (s *Service) recordGalleryAssetURLs(app core.App, record *core.Record) ([]string, []string) {
	productID := strings.TrimSpace(record.GetString("source_product_id"))
	if productID == "" {
		productID = strings.TrimSpace(readJSONAttribute(record, "source_product_id"))
	}
	payloadURLs := supplierRecordGalleryURLs(record)
	if productID == "" {
		productID = strings.TrimSpace(record.GetString("product_id"))
	}
	if productID == "" {
		if len(payloadURLs) > 0 {
			names := make([]string, 0, len(payloadURLs))
			for _, item := range payloadURLs {
				names = append(names, assetFileName(item))
			}
			return payloadURLs, names
		}
		primary := strings.TrimSpace(s.recordPrimaryAssetURL(record))
		if primary == "" {
			return nil, nil
		}
		return []string{primary}, []string{assetFileName(primary)}
	}

	assets, err := app.FindRecordsByFilter(
		CollectionSourceAssets,
		"product_id = {:product_id}",
		"sort",
		100,
		0,
		dbx.Params{"product_id": productID},
	)
	if err != nil || len(assets) == 0 {
		if len(payloadURLs) > 0 {
			names := make([]string, 0, len(payloadURLs))
			for _, item := range payloadURLs {
				names = append(names, assetFileName(item))
			}
			return payloadURLs, names
		}
		primary := strings.TrimSpace(s.recordPrimaryAssetURL(record))
		if primary == "" {
			return nil, nil
		}
		return []string{primary}, []string{assetFileName(primary)}
	}

	urls := make([]string, 0, len(assets))
	names := make([]string, 0, len(assets))
	seen := make(map[string]struct{}, len(assets))

	appendAsset := func(rawURL string, name string) {
		value := strings.TrimSpace(rawURL)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		urls = append(urls, value)
		if strings.TrimSpace(name) == "" {
			name = assetFileName(value)
		}
		names = append(names, name)
	}

	for _, asset := range assets {
		role := strings.TrimSpace(asset.GetString("asset_role"))
		if role != "" && !strings.EqualFold(role, "cover") && !strings.EqualFold(role, "carousel") {
			continue
		}
		urlValue := strings.TrimSpace(asset.GetString("source_url"))
		if urlValue == "" {
			urlValue = strings.TrimSpace(s.recordFileURL(asset, "original_image"))
		}
		if urlValue == "" {
			continue
		}
		name := strings.TrimSpace(asset.GetString("asset_key"))
		if name == "" {
			name = assetFileName(urlValue)
		}
		appendAsset(urlValue, name)
	}

	for _, item := range payloadURLs {
		appendAsset(item, assetFileName(item))
	}

	if len(urls) == 0 {
		primary := strings.TrimSpace(s.recordPrimaryAssetURL(record))
		if primary == "" {
			return nil, nil
		}
		return []string{primary}, []string{assetFileName(primary)}
	}

	return urls, names
}

func (s *Service) syncVendureCollections(ctx context.Context, app core.App, record *core.Record, productID string) error {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("vendure product id is empty")
	}

	if err := s.normalizePublishedCategoryBranchForSupplierRecord(ctx, app, record); err != nil {
		return err
	}

	collectionIDs, err := s.backendCollectionIDsForSupplierRecord(app, record)
	if err != nil {
		return err
	}
	if len(collectionIDs) == 0 {
		categoryKey := supplierRecordReleaseCategoryKey(record)
		if categoryKey == "" {
			return nil
		}
		return fmt.Errorf("no published backend category mapping for source category %s", categoryKey)
	}
	branchIDs, err := s.backendBranchCollectionIDsForSupplierRecord(app, record)
	if err != nil {
		return err
	}
	return s.vendure.SyncProductCollectionsExact(ctx, productID, collectionIDs, branchIDs)
}

func (s *Service) normalizePublishedCategoryBranchForSupplierRecord(ctx context.Context, app core.App, record *core.Record) error {
	categoryKey := supplierRecordReleaseCategoryKey(record)
	if categoryKey == "" {
		return nil
	}

	rootKey, rootPath, err := s.sourceCategoryRoot(app, categoryKey)
	if err != nil {
		return nil
	}

	mappings, err := app.FindRecordsByFilter(
		CollectionBackendCategoryMappings,
		"publish_status = {:status}",
		"-published_at",
		500,
		0,
		dbx.Params{"status": "published"},
	)
	if err != nil {
		return err
	}

	var branch []*core.Record
	for _, mapping := range mappings {
		sourceKey := strings.TrimSpace(mapping.GetString("source_key"))
		sourcePath := strings.TrimSpace(mapping.GetString("source_path"))
		if sourceKey == "" || sourcePath == "" {
			continue
		}
		if sourceKey == rootKey || sourcePath == rootPath || strings.HasPrefix(sourcePath, rootPath+"/") {
			branch = append(branch, mapping)
		}
	}

	sort.SliceStable(branch, func(i, j int) bool {
		return len(strings.TrimSpace(branch[i].GetString("source_path"))) < len(strings.TrimSpace(branch[j].GetString("source_path")))
	})

	for _, mapping := range branch {
		if _, err := s.publishBackendCategoryRecursive(
			ctx,
			app,
			strings.TrimSpace(mapping.GetString("source_key")),
			strings.TrimSpace(mapping.GetString("backend_collection")),
			strings.TrimSpace(mapping.GetString("backend_path")),
			strings.TrimSpace(mapping.GetString("note")),
			map[string]bool{},
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) backendCollectionIDsForSupplierRecord(app core.App, record *core.Record) ([]string, error) {
	categoryKey := supplierRecordReleaseCategoryKey(record)
	categoryKeys := make([]string, 0, 8)
	if categoryKey != "" {
		categoryKeys = append(categoryKeys, categoryKey)
		currentKey := categoryKey
		for {
			sourceRecord, err := app.FindFirstRecordByFilter(CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": currentKey})
			if err != nil {
				break
			}
			parentKey := strings.TrimSpace(sourceRecord.GetString("parent_key"))
			if parentKey == "" || containsTrimmed(categoryKeys, parentKey) {
				break
			}
			categoryKeys = append(categoryKeys, parentKey)
			currentKey = parentKey
		}
	}

	for _, observedKey := range supplierRecordObservedCategoryKeys(record) {
		observedKey = strings.TrimSpace(observedKey)
		if observedKey == "" || containsTrimmed(categoryKeys, observedKey) {
			continue
		}
		sourceRecord, err := app.FindFirstRecordByFilter(CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": observedKey})
		if err != nil {
			continue
		}
		parentKey := strings.TrimSpace(sourceRecord.GetString("parent_key"))
		hasChildren := sourceRecord.GetBool("has_children")
		// Top-level terminal categories should keep all observed products in backend.
		if parentKey == "" && !hasChildren {
			categoryKeys = append(categoryKeys, observedKey)
		}
	}

	if len(categoryKeys) == 0 {
		return nil, nil
	}

	collectionIDs := make([]string, 0, len(categoryKeys))
	for _, key := range categoryKeys {
		mapping, err := app.FindFirstRecordByFilter(
			CollectionBackendCategoryMappings,
			"source_key = {:source_key} && publish_status = {:status}",
			dbx.Params{
				"source_key": key,
				"status":     "published",
			},
		)
		if err != nil {
			continue
		}
		if collectionID := strings.TrimSpace(mapping.GetString("backend_collection_id")); collectionID != "" {
			collectionIDs = append(collectionIDs, collectionID)
		}
	}
	return uniqueTrimmed(collectionIDs), nil
}

func (s *Service) backendBranchCollectionIDsForSupplierRecord(app core.App, record *core.Record) ([]string, error) {
	categoryKey := supplierRecordReleaseCategoryKey(record)
	if categoryKey == "" {
		return nil, nil
	}

	rootKey, rootPath, err := s.sourceCategoryRoot(app, categoryKey)
	if err != nil {
		return nil, err
	}

	mappings, err := app.FindRecordsByFilter(
		CollectionBackendCategoryMappings,
		"publish_status = {:status}",
		"-published_at",
		500,
		0,
		dbx.Params{"status": "published"},
	)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		sourceKey := strings.TrimSpace(mapping.GetString("source_key"))
		sourcePath := strings.TrimSpace(mapping.GetString("source_path"))
		if sourceKey == "" || sourcePath == "" {
			continue
		}
		if sourceKey == rootKey || sourcePath == rootPath || strings.HasPrefix(sourcePath, rootPath+"/") {
			if collectionID := strings.TrimSpace(mapping.GetString("backend_collection_id")); collectionID != "" {
				ids = append(ids, collectionID)
			}
		}
	}

	return uniqueTrimmed(ids), nil
}

func (s *Service) sourceCategoryRoot(app core.App, categoryKey string) (string, string, error) {
	currentKey := strings.TrimSpace(categoryKey)
	if currentKey == "" {
		return "", "", fmt.Errorf("category key is empty")
	}

	var lastRecord *core.Record
	for {
		record, err := app.FindFirstRecordByFilter(CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": currentKey})
		if err != nil {
			if lastRecord != nil {
				return strings.TrimSpace(lastRecord.GetString("source_key")), strings.TrimSpace(lastRecord.GetString("category_path")), nil
			}
			return "", "", err
		}
		lastRecord = record
		parentKey := strings.TrimSpace(record.GetString("parent_key"))
		if parentKey == "" {
			break
		}
		currentKey = parentKey
	}

	if lastRecord == nil {
		return "", "", fmt.Errorf("source category not found")
	}
	return strings.TrimSpace(lastRecord.GetString("source_key")), strings.TrimSpace(lastRecord.GetString("category_path")), nil
}

func (s *Service) BackendReleaseSummary(_ context.Context, app core.App, limit int) (BackendReleaseSummary, error) {
	return s.BackendReleaseSummaryFiltered(context.Background(), app, limit, BackendReleaseFilter{})
}

func normalizeBackendReleaseFilter(filter BackendReleaseFilter, fallbackPageSize int) BackendReleaseFilter {
	filter.SyncStatus = strings.TrimSpace(filter.SyncStatus)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.SortBy = normalizeBackendReleaseSortBy(filter.SortBy)
	filter.SortOrder = normalizeBackendReleaseSortOrder(filter.SortOrder)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if fallbackPageSize <= 0 {
		fallbackPageSize = 12
	}
	if filter.PageSize <= 0 {
		filter.PageSize = fallbackPageSize
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}
	return filter
}

func normalizeBackendReleaseSortBy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "updated", "updated_at", "pim_updated", "pim_updated_at":
		return "updated"
	case "created", "created_at":
		return "created"
	case "synced", "sync", "synced_at", "last_synced_at":
		return "last_synced_at"
	case "supplier_updated", "supplier_updated_at":
		return "supplier_updated_at"
	default:
		return "updated"
	}
}

func normalizeBackendReleaseSortOrder(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "asc", "ascending":
		return "asc"
	default:
		return "desc"
	}
}

func backendReleaseProductSortValue(record *core.Record, sortBy string) string {
	switch sortBy {
	case "created":
		return strings.TrimSpace(record.GetString("created"))
	case "last_synced_at":
		return strings.TrimSpace(record.GetString("last_synced_at"))
	case "supplier_updated_at":
		return strings.TrimSpace(record.GetString("supplier_updated_at"))
	default:
		return strings.TrimSpace(record.GetString("updated"))
	}
}

func compareBackendReleaseProducts(left, right *core.Record, sortBy, sortOrder string) int {
	leftValue := backendReleaseProductSortValue(left, sortBy)
	rightValue := backendReleaseProductSortValue(right, sortBy)

	if leftValue != rightValue {
		if leftValue == "" {
			return 1
		}
		if rightValue == "" {
			return -1
		}
		cmp := strings.Compare(leftValue, rightValue)
		if sortOrder == "desc" {
			cmp = -cmp
		}
		if cmp != 0 {
			return cmp
		}
	}

	updatedCmp := strings.Compare(strings.TrimSpace(right.GetString("updated")), strings.TrimSpace(left.GetString("updated")))
	if updatedCmp != 0 {
		return updatedCmp
	}
	return strings.Compare(right.Id, left.Id)
}

func matchesBackendReleaseProductFilter(filter BackendReleaseFilter, record *core.Record) bool {
	if status := strings.ToLower(strings.TrimSpace(filter.SyncStatus)); status != "" && status != "all" {
		recordStatus := strings.ToLower(strings.TrimSpace(record.GetString("sync_status")))
		if recordStatus != status {
			return false
		}
	}

	if query := strings.ToLower(strings.TrimSpace(filter.Query)); query != "" {
		search := strings.ToLower(strings.Join([]string{
			displayTitle(record),
			record.GetString("supplier_code"),
			record.GetString("original_sku"),
			record.GetString("normalized_category"),
			record.GetString("raw_category"),
			record.GetString("vendure_product_id"),
			record.GetString("vendure_variant_id"),
			record.GetString("last_sync_error"),
		}, " "))
		if !strings.Contains(search, query) {
			return false
		}
	}

	return true
}

func (s *Service) BackendReleaseSummaryFiltered(_ context.Context, app core.App, limit int, filter BackendReleaseFilter) (BackendReleaseSummary, error) {
	if limit <= 0 {
		limit = 12
	}
	filter = normalizeBackendReleaseFilter(filter, limit)
	summary := BackendReleaseSummary{}

	sourceCategories, err := app.FindAllRecords(CollectionSourceCategories)
	if err != nil {
		return summary, err
	}
	summary.CategoryCount = len(sourceCategories)
	bySourceKey := make(map[string]*core.Record, len(sourceCategories))
	for _, record := range sourceCategories {
		bySourceKey[record.GetString("source_key")] = record
	}

	categoryMappings, err := app.FindAllRecords(CollectionBackendCategoryMappings)
	if err == nil {
		for _, mapping := range categoryMappings {
			status := strings.ToLower(strings.TrimSpace(mapping.GetString("publish_status")))
			item := BackendCategoryMappingItem{
				ID:                  mapping.Id,
				SourceKey:           mapping.GetString("source_key"),
				SourcePath:          mapping.GetString("source_path"),
				BackendCollection:   mapping.GetString("backend_collection"),
				BackendCollectionID: mapping.GetString("backend_collection_id"),
				BackendPath:         mapping.GetString("backend_path"),
				PublishStatus:       mapping.GetString("publish_status"),
				LastError:           mapping.GetString("last_error"),
				Note:                mapping.GetString("note"),
				PublishedAt:         mapping.GetString("published_at"),
			}
			if source := bySourceKey[item.SourceKey]; source != nil {
				item.Label = source.GetString("label")
				if item.SourcePath == "" {
					item.SourcePath = source.GetString("category_path")
				}
			}
			switch status {
			case "mapped":
				summary.MappedCategoryCount++
				summary.PendingCategoryCount++
			case "published":
				summary.MappedCategoryCount++
				summary.PublishedCount++
			case "error":
				summary.ErrorCategoryCount++
			default:
				summary.PendingCategoryCount++
			}
			summary.Categories = append(summary.Categories, item)
		}
	}

	for _, source := range sourceCategories {
		sourceKey := strings.TrimSpace(source.GetString("source_key"))
		if sourceKey == "" {
			continue
		}
		found := false
		for _, item := range summary.Categories {
			if item.SourceKey == sourceKey {
				found = true
				break
			}
		}
		if !found {
			summary.PendingCategoryCount++
			summary.Categories = append(summary.Categories, BackendCategoryMappingItem{
				SourceKey:     sourceKey,
				Label:         source.GetString("label"),
				SourcePath:    source.GetString("category_path"),
				PublishStatus: "pending",
			})
		}
	}
	summary.SuggestedCategories = buildBackendCategorySuggestions(sourceCategories, summary.Categories, 8)
	summary.Branches = buildBackendCategoryBranchSummaries(sourceCategories, summary.Categories)
	for _, branch := range summary.Branches {
		if branch.PublishedCount > 0 {
			summary.PublishedRootCount++
		}
	}

	sortExpr, err := supplierProductSortExpr(app)
	if err != nil {
		return summary, err
	}

	records, err := app.FindRecordsByFilter(CollectionSupplierProducts, "sync_status != ''", sortExpr, limit, 0)
	if err != nil {
		return summary, err
	}
	allProducts, err := app.FindAllRecords(CollectionSupplierProducts)
	if err == nil {
		summary.ProductCount = len(allProducts)
		filtered := make([]*core.Record, 0, len(allProducts))
		for _, record := range allProducts {
			status := strings.ToLower(strings.TrimSpace(record.GetString("sync_status")))
			switch status {
			case StatusApproved:
				summary.ReadyProductCount++
			case StatusSynced:
				summary.SyncedProductCount++
			case StatusError:
				summary.ErrorProductCount++
			case StatusOffline:
				summary.OfflineProductCount++
			}
			if matchesBackendReleaseProductFilter(filter, record) {
				filtered = append(filtered, record)
			}
		}
		slices.SortFunc(filtered, func(left, right *core.Record) int {
			return compareBackendReleaseProducts(left, right, filter.SortBy, filter.SortOrder)
		})
		summary.FilteredProductCount = len(filtered)
		summary.ProductPage = filter.Page
		summary.ProductPageSize = filter.PageSize
		summary.ProductPages = totalPages(len(filtered), filter.PageSize)
		if summary.ProductPages <= 0 {
			summary.ProductPages = 1
		}
		if summary.ProductPage > summary.ProductPages {
			summary.ProductPage = summary.ProductPages
		}
		start := (summary.ProductPage - 1) * filter.PageSize
		if start < 0 {
			start = 0
		}
		if start > len(filtered) {
			start = len(filtered)
		}
		end := start + filter.PageSize
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, record := range filtered[start:end] {
			summary.Products = append(summary.Products, backendReleaseProductItemFromRecord(record))
		}
		if len(filtered) > 0 {
			summary.RecommendedProducts = pickRecommendedBackendReleaseProducts(filtered, 3)
		}
	}
	if len(summary.RecommendedProducts) == 0 {
		for _, record := range records {
			summary.Products = append(summary.Products, backendReleaseProductItemFromRecord(record))
		}
		summary.RecommendedProducts = pickRecommendedBackendReleaseProducts(records, 3)
	}

	slices.SortFunc(summary.Categories, func(a, b BackendCategoryMappingItem) int {
		return strings.Compare(a.SourcePath, b.SourcePath)
	})

	return summary, nil
}

func backendReleaseProductItemFromRecord(record *core.Record) BackendReleaseProductItem {
	hasProcessedImage := strings.TrimSpace(sourceAssetConsumerImageURL(record)) != ""
	hasConsumerImage := strings.TrimSpace(sourceAssetConsumerImageURL(record)) != "" || strings.TrimSpace(sourceAssetPrimaryImageURL(record)) != ""
	conversionRate := sourceConversionRateFromSupplierRecord(record)
	return BackendReleaseProductItem{
		ID:                 record.Id,
		SupplierCode:       record.GetString("supplier_code"),
		SKU:                record.GetString("original_sku"),
		Title:              displayTitle(record),
		NormalizedCategory: defaultString(record.GetString("normalized_category"), record.GetString("raw_category")),
		TargetAudience:     defaultString(record.GetString("target_audience"), "ALL"),
		ConversionRate:     conversionRate,
		SyncStatus:         record.GetString("sync_status"),
		SupplierStatus:     record.GetString("supplier_status"),
		VendureProductID:   record.GetString("vendure_product_id"),
		VendureVariantID:   record.GetString("vendure_variant_id"),
		CreatedAt:          record.GetString("created"),
		SupplierUpdatedAt:  record.GetString("supplier_updated_at"),
		UpdatedAt:          record.GetString("updated"),
		LastSyncedAt:       record.GetString("last_synced_at"),
		LastSeenAt:         record.GetString("last_seen_at"),
		OfflineAt:          record.GetString("offline_at"),
		LastSyncError:      record.GetString("last_sync_error"),
		HasProcessedImage:  hasProcessedImage,
		HasConsumerImage:   hasConsumerImage,
		ReadyForPreview:    strings.TrimSpace(record.GetString("original_sku")) != "" && strings.TrimSpace(displayTitle(record)) != "",
	}
}

func buildBackendCategorySuggestions(sourceCategories []*core.Record, existing []BackendCategoryMappingItem, limit int) []BackendCategoryMappingSuggestion {
	if limit <= 0 {
		limit = 8
	}
	mapped := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		if strings.TrimSpace(item.SourceKey) == "" {
			continue
		}
		if strings.TrimSpace(item.BackendCollection) != "" || strings.TrimSpace(item.BackendPath) != "" {
			mapped[strings.TrimSpace(item.SourceKey)] = struct{}{}
		}
	}
	suggestions := make([]BackendCategoryMappingSuggestion, 0, limit)
	for _, source := range sourceCategories {
		sourceKey := strings.TrimSpace(source.GetString("source_key"))
		if sourceKey == "" {
			continue
		}
		if _, ok := mapped[sourceKey]; ok {
			continue
		}
		sourcePath := strings.TrimSpace(source.GetString("category_path"))
		segments := splitCategoryPath(sourcePath, source.GetString("label"))
		collection := strings.Join(slugifySegments(segments), "/")
		level := source.GetInt("level")
		reason := "建议先为顶级分类建立 Collection，再逐级补子分类。"
		if level >= 3 {
			reason = "末级分类建议直接映射到最终 Collection，便于后续商品发布。"
		} else if level == 2 {
			reason = "二级分类建议保留为中间 Collection，保持导航层级。"
		}
		suggestions = append(suggestions, BackendCategoryMappingSuggestion{
			SourceKey:            sourceKey,
			Label:                source.GetString("label"),
			SourcePath:           sourcePath,
			SourceLevel:          level,
			SuggestedCollection:  collection,
			SuggestedBackendPath: strings.Join(segments, "/"),
			Reason:               reason,
		})
		if len(suggestions) >= limit {
			break
		}
	}
	return suggestions
}

func buildBackendCategoryBranchSummaries(sourceCategories []*core.Record, existing []BackendCategoryMappingItem) []BackendCategoryBranchSummary {
	topLevelByLabel := make(map[string]string)
	for _, source := range sourceCategories {
		label := strings.TrimSpace(source.GetString("label"))
		if label == "" {
			continue
		}
		level := source.GetInt("level")
		path := splitCategoryPath(source.GetString("category_path"), label)
		if level <= 1 || len(path) <= 1 {
			topLevelByLabel[path[0]] = strings.TrimSpace(source.GetString("source_key"))
		}
	}

	type accumulator struct {
		BackendCategoryBranchSummary
	}
	branches := make(map[string]*accumulator)
	for _, source := range sourceCategories {
		sourceKey := strings.TrimSpace(source.GetString("source_key"))
		path := splitCategoryPath(source.GetString("category_path"), source.GetString("label"))
		if sourceKey == "" || len(path) == 0 {
			continue
		}
		rootLabel := path[0]
		rootKey := topLevelByLabel[rootLabel]
		if rootKey == "" {
			rootKey = sourceKey
		}
		entry := branches[rootKey]
		if entry == nil {
			entry = &accumulator{BackendCategoryBranchSummary: BackendCategoryBranchSummary{
				RootKey: rootKey,
				Label:   rootLabel,
			}}
			branches[rootKey] = entry
		}
		entry.TotalCount++
	}

	statusBySourceKey := make(map[string]string, len(existing))
	for _, item := range existing {
		sourceKey := strings.TrimSpace(item.SourceKey)
		if sourceKey == "" {
			continue
		}
		statusBySourceKey[sourceKey] = strings.ToLower(strings.TrimSpace(item.PublishStatus))
	}

	for _, source := range sourceCategories {
		sourceKey := strings.TrimSpace(source.GetString("source_key"))
		path := splitCategoryPath(source.GetString("category_path"), source.GetString("label"))
		if sourceKey == "" || len(path) == 0 {
			continue
		}
		rootLabel := path[0]
		rootKey := topLevelByLabel[rootLabel]
		if rootKey == "" {
			rootKey = sourceKey
		}
		entry := branches[rootKey]
		if entry == nil {
			continue
		}
		switch statusBySourceKey[sourceKey] {
		case "published":
			entry.PublishedCount++
		case "error":
			entry.ErrorCount++
		default:
			entry.PendingCount++
		}
	}

	items := make([]BackendCategoryBranchSummary, 0, len(branches))
	for _, branch := range branches {
		items = append(items, branch.BackendCategoryBranchSummary)
	}
	slices.SortFunc(items, func(left, right BackendCategoryBranchSummary) int {
		if left.PublishedCount != right.PublishedCount {
			if left.PublishedCount > right.PublishedCount {
				return -1
			}
			return 1
		}
		return strings.Compare(left.Label, right.Label)
	})
	return items
}

func pickRecommendedBackendReleaseProducts(records []*core.Record, limit int) []BackendReleaseProductItem {
	if limit <= 0 {
		limit = 3
	}
	type candidate struct {
		item   BackendReleaseProductItem
		score  int
		record *core.Record
	}
	candidates := make([]candidate, 0, len(records))
	for _, record := range records {
		item := backendReleaseProductItemFromRecord(record)
		score := 0
		reasons := make([]string, 0, 3)
		if item.ConversionRate > 1 {
			score += 3
			reasons = append(reasons, "多单位换算")
		}
		if item.HasProcessedImage {
			score += 2
			reasons = append(reasons, "已有处理图")
		}
		if strings.EqualFold(item.SyncStatus, StatusApproved) {
			score += 2
			reasons = append(reasons, "待发布验证")
		}
		if item.TargetAudience != "" && !strings.EqualFold(item.TargetAudience, "ALL") {
			score += 1
			reasons = append(reasons, "客群分流")
		}
		if score == 0 {
			reasons = append(reasons, "基础联调样例")
		}
		item.Reason = strings.Join(reasons, " / ")
		candidates = append(candidates, candidate{item: item, score: score, record: record})
	}
	slices.SortFunc(candidates, func(left, right candidate) int {
		if left.score != right.score {
			if left.score > right.score {
				return -1
			}
			return 1
		}
		return strings.Compare(left.item.Title, right.item.Title)
	})
	recommended := make([]BackendReleaseProductItem, 0, minInt(limit, len(candidates)))
	for _, item := range candidates {
		if len(recommended) >= limit {
			break
		}
		recommended = append(recommended, item.item)
	}
	return recommended
}

func (s *Service) SaveBackendCategoryMapping(_ context.Context, app core.App, sourceKey string, backendCollection string, backendPath string, note string) error {
	sourceKey = strings.TrimSpace(sourceKey)
	if sourceKey == "" {
		return fmt.Errorf("source_key is required")
	}
	backendCollection = strings.TrimSpace(backendCollection)
	backendPath = strings.TrimSpace(backendPath)
	if backendCollection == "" && backendPath == "" {
		return fmt.Errorf("backend collection or backend path is required")
	}
	source, err := app.FindFirstRecordByFilter(CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": sourceKey})
	if err != nil {
		return fmt.Errorf("load source category %s: %w", sourceKey, err)
	}
	_, err = upsertByFilter(app, CollectionBackendCategoryMappings, "source_key = {:source_key}", dbx.Params{"source_key": sourceKey}, func(record *core.Record, created bool) error {
		record.Set("source_key", sourceKey)
		record.Set("source_path", source.GetString("category_path"))
		record.Set("backend_collection", backendCollection)
		record.Set("backend_path", backendPath)
		record.Set("note", strings.TrimSpace(note))
		record.Set("last_error", "")
		record.Set("publish_status", "mapped")
		return nil
	})
	return err
}

func (s *Service) PublishBackendCategory(ctx context.Context, app core.App, sourceKey string, backendCollection string, backendPath string, note string) (BackendCategoryMappingItem, error) {
	sourceKey = strings.TrimSpace(sourceKey)
	if sourceKey == "" {
		return BackendCategoryMappingItem{}, fmt.Errorf("source_key is required")
	}
	visited := map[string]bool{}
	return s.publishBackendCategoryRecursive(ctx, app, sourceKey, strings.TrimSpace(backendCollection), strings.TrimSpace(backendPath), strings.TrimSpace(note), visited)
}

func (s *Service) PublishBackendCategoriesBatch(ctx context.Context, app core.App, sourceKeys []string) (BackendCategoryPublishBatchResult, error) {
	keys := uniqueNonEmptyStrings(sourceKeys)
	result := BackendCategoryPublishBatchResult{
		Requested:    len(keys),
		RequestedIDs: keys,
		Items:        make([]BackendCategoryMappingItem, 0, len(keys)),
		Errors:       make([]string, 0),
	}
	if len(keys) == 0 {
		return result, fmt.Errorf("missing backend category source keys")
	}

	for _, sourceKey := range keys {
		item, err := s.PublishBackendCategory(ctx, app, sourceKey, "", "", "")
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", sourceKey, strings.TrimSpace(err.Error())))
			continue
		}
		result.Published++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (s *Service) publishBackendCategoryRecursive(ctx context.Context, app core.App, sourceKey string, backendCollection string, backendPath string, note string, visited map[string]bool) (BackendCategoryMappingItem, error) {
	if visited[sourceKey] {
		return BackendCategoryMappingItem{}, fmt.Errorf("detected category publish cycle for %s", sourceKey)
	}
	visited[sourceKey] = true

	sourceRecord, err := app.FindFirstRecordByFilter(CollectionSourceCategories, "source_key = {:source_key}", dbx.Params{"source_key": sourceKey})
	if err != nil {
		return BackendCategoryMappingItem{}, fmt.Errorf("load source category %s: %w", sourceKey, err)
	}

	mappingRecord, _ := app.FindFirstRecordByFilter(CollectionBackendCategoryMappings, "source_key = {:source_key}", dbx.Params{"source_key": sourceKey})
	if mappingRecord == nil {
		collection, findErr := app.FindCollectionByNameOrId(CollectionBackendCategoryMappings)
		if findErr != nil {
			return BackendCategoryMappingItem{}, findErr
		}
		mappingRecord = core.NewRecord(collection)
		mappingRecord.Set("source_key", sourceKey)
	}

	defaultCollection, defaultPath := defaultBackendCategoryMappingValues(sourceRecord)
	if backendCollection == "" {
		backendCollection = strings.TrimSpace(mappingRecord.GetString("backend_collection"))
	}
	if backendPath == "" {
		backendPath = strings.TrimSpace(mappingRecord.GetString("backend_path"))
	}
	if backendCollection == "" {
		backendCollection = defaultCollection
	}
	if backendPath == "" {
		backendPath = defaultPath
	}
	if backendCollection == "" && backendPath == "" {
		return BackendCategoryMappingItem{}, fmt.Errorf("backend collection or backend path is required")
	}

	parentCollectionID := ""
	parentKey := strings.TrimSpace(sourceRecord.GetString("parent_key"))
	if parentKey != "" {
		parentResult, parentErr := s.publishBackendCategoryRecursive(ctx, app, parentKey, "", "", "", visited)
		if parentErr != nil {
			recordBackendCategoryPublishFailure(mappingRecord, sourceRecord, backendCollection, backendPath, note, parentErr)
			_ = app.Save(mappingRecord)
			return BackendCategoryMappingItem{}, parentErr
		}
		parentCollectionID = strings.TrimSpace(parentResult.BackendCollectionID)
	}

	payload := vendure.CollectionPayload{
		SourceCategoryKey:   sourceKey,
		SourceCategoryPath:  sourceRecord.GetString("category_path"),
		SourceCategoryLevel: sourceRecord.GetInt("depth"),
		Name:                defaultString(lastCategoryPathSegment(backendPath), sourceRecord.GetString("label")),
		Slug:                backendCollectionSlug(backendCollection, backendPath, sourceKey),
		Description:         defaultString(sourceRecord.GetString("category_path"), sourceRecord.GetString("label")),
		ParentCollectionID:  parentCollectionID,
	}

	result, vendureErr := s.vendure.EnsureCollection(ctx, payload)
	if vendureErr != nil {
		recordBackendCategoryPublishFailure(mappingRecord, sourceRecord, backendCollection, backendPath, note, vendureErr)
		_ = app.Save(mappingRecord)
		return BackendCategoryMappingItem{}, vendureErr
	}

	mappingRecord.Set("source_path", sourceRecord.GetString("category_path"))
	mappingRecord.Set("backend_collection", backendCollection)
	mappingRecord.Set("backend_path", backendPath)
	mappingRecord.Set("backend_collection_id", result.CollectionID)
	mappingRecord.Set("publish_status", "published")
	mappingRecord.Set("last_error", "")
	mappingRecord.Set("note", note)
	mappingRecord.Set("published_at", time.Now().Format(time.RFC3339))
	if err := app.Save(mappingRecord); err != nil {
		return BackendCategoryMappingItem{}, err
	}

	return BackendCategoryMappingItem{
		ID:                  mappingRecord.Id,
		SourceKey:           sourceKey,
		Label:               sourceRecord.GetString("label"),
		SourcePath:          sourceRecord.GetString("category_path"),
		BackendCollection:   backendCollection,
		BackendCollectionID: result.CollectionID,
		BackendPath:         backendPath,
		PublishStatus:       "published",
		LastError:           "",
		Note:                note,
		PublishedAt:         mappingRecord.GetString("published_at"),
	}, nil
}

func recordBackendCategoryPublishFailure(mappingRecord *core.Record, sourceRecord *core.Record, backendCollection string, backendPath string, note string, cause error) {
	mappingRecord.Set("source_path", sourceRecord.GetString("category_path"))
	mappingRecord.Set("backend_collection", backendCollection)
	mappingRecord.Set("backend_path", backendPath)
	mappingRecord.Set("publish_status", "error")
	mappingRecord.Set("last_error", strings.TrimSpace(cause.Error()))
	mappingRecord.Set("note", note)
}

func defaultBackendCategoryMappingValues(sourceRecord *core.Record) (string, string) {
	sourcePath := strings.TrimSpace(sourceRecord.GetString("category_path"))
	segments := splitCategoryPath(sourcePath, sourceRecord.GetString("label"))
	return strings.Join(slugifySegments(segments), "/"), strings.Join(segments, "/")
}

func lastCategoryPathSegment(pathValue string) string {
	segments := splitCategoryPath(pathValue, "")
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

func backendCollectionSlug(collectionPath string, backendPath string, sourceKey string) string {
	candidate := collectionPath
	if strings.TrimSpace(candidate) == "" {
		candidate = backendPath
	}
	slug := strings.Join(slugifySegments(splitCategoryPath(candidate, sourceKey)), "-")
	if slug == "" {
		slug = slugify(sourceKey)
	}
	return slug
}

func (s *Service) PreviewBackendReleasePayload(_ context.Context, app core.App, recordID string) (BackendReleasePayloadPreview, error) {
	record, err := app.FindRecordById(CollectionSupplierProducts, strings.TrimSpace(recordID))
	if err != nil {
		return BackendReleasePayloadPreview{}, err
	}
	payload := s.buildVendurePayload(app, record)
	preview := map[string]any{
		"name":              payload.Name,
		"slug":              payload.Slug,
		"description":       payload.Description,
		"sku":               payload.SKU,
		"currencyCode":      payload.CurrencyCode,
		"consumerPrice":     payload.ConsumerPrice,
		"businessPrice":     payload.BusinessPrice,
		"supplierCode":      payload.SupplierCode,
		"supplierCostPrice": payload.SupplierCostPrice,
		"conversionRate":    payload.ConversionRate,
		"sourceProductId":   payload.SourceProductID,
		"sourceType":        payload.SourceType,
		"targetAudience":    payload.TargetAudience,
		"defaultStock":      payload.DefaultStock,
		"salesUnit":         payload.SalesUnit,
		"assetURL":          payload.AssetURL,
		"assetURLs":         payload.AssetURLs,
		"cEndAssetURL":      payload.CEndAssetURL,
		"vendureProductId":  payload.VendureProduct,
		"vendureVariantId":  payload.VendureVariant,
		"variants":          payload.Variants,
	}
	return BackendReleasePayloadPreview{
		RecordID: record.Id,
		Payload:  preview,
	}, nil
}

func (s *Service) resolveProcurementItems(_ context.Context, app core.App, requested []ProcurementItemRequest) ([]procurementCatalogItem, error) {
	if len(requested) == 0 {
		return nil, fmt.Errorf("procurement items are required")
	}

	items := make([]procurementCatalogItem, 0, len(requested))
	for _, requestItem := range requested {
		quantity := requestItem.Quantity
		if quantity <= 0 {
			return nil, fmt.Errorf("procurement quantity must be positive for sku %s", strings.TrimSpace(requestItem.OriginalSKU))
		}

		supplierCode := strings.TrimSpace(requestItem.SupplierCode)
		if supplierCode == "" {
			supplierCode = s.cfg.Supplier.Code
		}

		sku := strings.TrimSpace(requestItem.OriginalSKU)
		if sku == "" {
			return nil, fmt.Errorf("procurement sku is required")
		}

		record, err := app.FindFirstRecordByFilter(
			CollectionSupplierProducts,
			"supplier_code = {:supplier} && original_sku = {:sku}",
			dbx.Params{
				"supplier": supplierCode,
				"sku":      sku,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load procurement item %s/%s: %w", supplierCode, sku, err)
		}

		requestedSalesUnit := strings.TrimSpace(requestItem.SalesUnit)
		unitOption := supplierRecordUnitOption(record, requestedSalesUnit)
		salesUnit := defaultString(requestedSalesUnit, defaultString(unitOption.UnitName, defaultString(readJSONAttribute(record, "sales_unit"), "件")))
		basePrice := positiveOr(unitOption.Price, record.GetFloat("b_price"))
		if basePrice <= 0 {
			basePrice = record.GetFloat("c_price")
		}

		items = append(items, procurementCatalogItem{
			SupplierCode:       supplierCode,
			OriginalSKU:        sku,
			Title:              displayTitle(record),
			NormalizedCategory: defaultString(record.GetString("normalized_category"), record.GetString("raw_category")),
			Quantity:           quantity,
			SalesUnit:          salesUnit,
			// Historical source capture sets cost_price to 0 for many records.
			// Fallback to b_price to avoid zero-cost procurement summaries.
			CostPrice:     positiveOr(record.GetFloat("cost_price"), basePrice),
			BusinessPrice: positiveOr(basePrice, record.GetFloat("b_price")),
			ConsumerPrice: positiveOr(basePrice, record.GetFloat("c_price")),
			NeedColdChain: strings.EqualFold(readJSONAttribute(record, "need_cold_chain"), "true"),
		})
	}

	return items, nil
}

func procurementOrderFromRecord(record *core.Record) (ProcurementOrder, error) {
	summary, err := decodeProcurementSummary(record.GetString("summary_json"))
	if err != nil {
		return ProcurementOrder{}, err
	}

	results, err := decodeProcurementResults(record.GetString("results_json"))
	if err != nil {
		return ProcurementOrder{}, err
	}

	return ProcurementOrder{
		ID:              record.Id,
		ExternalRef:     record.GetString("external_ref"),
		Status:          record.GetString("status"),
		Connector:       record.GetString("connector"),
		Capabilities:    summary.Capabilities,
		DeliveryAddress: record.GetString("delivery_address"),
		Notes:           record.GetString("notes"),
		LastActionNote:  record.GetString("last_action_note"),
		SupplierCount:   record.GetInt("supplier_count"),
		ItemCount:       record.GetInt("item_count"),
		TotalQty:        record.GetFloat("total_qty"),
		TotalCostAmount: record.GetFloat("total_cost_amount"),
		RiskyItemCount:  record.GetInt("risky_item_count"),
		Summary:         summary,
		Results:         results,
		ExportCSV:       record.GetString("export_csv"),
		Created:         record.GetString("created"),
		Updated:         record.GetString("updated"),
		ReviewedAt:      record.GetString("reviewed_at"),
		ExportedAt:      record.GetString("exported_at"),
		OrderedAt:       record.GetString("ordered_at"),
		ReceivedAt:      record.GetString("received_at"),
		CanceledAt:      record.GetString("canceled_at"),
	}, nil
}

func decodeProcurementSummary(raw string) (ProcurementSummary, error) {
	var summary ProcurementSummary
	if strings.TrimSpace(raw) == "" {
		return summary, fmt.Errorf("missing procurement summary")
	}

	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		return ProcurementSummary{}, fmt.Errorf("decode procurement summary: %w", err)
	}

	return summary, nil
}

func decodeProcurementResults(raw string) ([]supplier.PurchaseOrderResult, error) {
	if strings.TrimSpace(raw) == "" {
		return []supplier.PurchaseOrderResult{}, nil
	}

	var results []supplier.PurchaseOrderResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		return nil, fmt.Errorf("decode procurement results: %w", err)
	}

	return results, nil
}

func setJSONField(record *core.Record, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}

	record.Set(key, string(encoded))
	return nil
}

func applyProcurementStatus(record *core.Record, status string, note string) error {
	status = strings.TrimSpace(status)
	if !isValidProcurementStatus(status) {
		return fmt.Errorf("unsupported procurement status: %s", status)
	}

	current := record.GetString("status")
	if current == "" {
		current = ProcurementStatusDraft
	}

	if !isAllowedProcurementTransition(current, status) {
		return fmt.Errorf("invalid procurement status transition: %s -> %s", current, status)
	}

	record.Set("status", status)
	record.Set("last_action_note", strings.TrimSpace(note))

	now := time.Now().Format(time.RFC3339)
	switch status {
	case ProcurementStatusReviewed:
		record.Set("reviewed_at", now)
	case ProcurementStatusExported:
		record.Set("exported_at", now)
	case ProcurementStatusOrdered:
		record.Set("ordered_at", now)
	case ProcurementStatusReceived:
		record.Set("received_at", now)
	case ProcurementStatusCanceled:
		record.Set("canceled_at", now)
	}

	return nil
}

func isValidProcurementStatus(status string) bool {
	switch status {
	case ProcurementStatusDraft, ProcurementStatusReviewed, ProcurementStatusExported, ProcurementStatusOrdered, ProcurementStatusReceived, ProcurementStatusCanceled:
		return true
	default:
		return false
	}
}

func isAllowedProcurementTransition(current string, next string) bool {
	if current == next {
		return true
	}

	switch current {
	case ProcurementStatusDraft:
		return next == ProcurementStatusReviewed || next == ProcurementStatusExported || next == ProcurementStatusOrdered || next == ProcurementStatusCanceled
	case ProcurementStatusReviewed:
		return next == ProcurementStatusExported || next == ProcurementStatusOrdered || next == ProcurementStatusCanceled
	case ProcurementStatusExported:
		return next == ProcurementStatusOrdered || next == ProcurementStatusCanceled
	case ProcurementStatusOrdered:
		return next == ProcurementStatusReceived || next == ProcurementStatusCanceled
	case ProcurementStatusReceived, ProcurementStatusCanceled:
		return false
	default:
		return false
	}
}

func (s *Service) markMissingProductsOffline(ctx context.Context, app core.App, seen map[string]struct{}) (harvestOfflineSummary, error) {
	records, err := app.FindAllRecords(CollectionSupplierProducts)
	if err != nil {
		return harvestOfflineSummary{}, err
	}

	summary := harvestOfflineSummary{}
	now := time.Now()
	for _, record := range records {
		if record.GetString("supplier_code") != s.cfg.Supplier.Code {
			continue
		}

		key := s.recordKey(record.GetString("supplier_code"), record.GetString("original_sku"))
		if _, ok := seen[key]; ok {
			continue
		}

		newlyOffline, changed := markRecordMissingFromSupplier(record, now)
		if changed {
			if err := app.Save(record); err != nil {
				return summary, err
			}
		}

		if vendureID := record.GetString("vendure_product_id"); vendureID != "" {
			if err := s.disableVendureProductForOffline(ctx, vendureID); err != nil {
				summary.FailureItems = appendHarvestFailure(summary.FailureItems, HarvestFailureItem{
					SKU:       record.GetString("original_sku"),
					ProductID: vendureID,
					Step:      "disable_product",
					Error:     err.Error(),
				})
				app.Logger().Error("disable vendure product failed", "productId", vendureID, "error", err)
			}
		}

		if newlyOffline {
			summary.OfflineCount++
		}
	}

	return summary, nil
}

func (s *Service) disableVendureProductForOffline(ctx context.Context, productID string) error {
	timeout := s.cfg.Vendure.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	disableCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	return s.vendure.DisableProduct(disableCtx, productID)
}

func markRecordMissingFromSupplier(record *core.Record, now time.Time) (bool, bool) {
	newlyOffline := false
	changed := false

	if record.GetString("sync_status") != StatusOffline {
		record.Set("sync_status", StatusOffline)
		newlyOffline = true
		changed = true
	}
	if record.GetString("supplier_status") != SupplierStatusOffline {
		record.Set("supplier_status", SupplierStatusOffline)
		changed = true
	}
	if strings.TrimSpace(record.GetString("offline_reason")) != "missing_from_supplier_feed" {
		record.Set("offline_reason", "missing_from_supplier_feed")
		changed = true
	}
	if strings.TrimSpace(record.GetString("offline_at")) == "" {
		record.Set("offline_at", now.Format(time.RFC3339))
		changed = true
	}

	return newlyOffline, changed
}

func (s *Service) resolveCategory(_ context.Context, app core.App, supplierCode string, rawCategory string) (string, error) {
	record, err := app.FindFirstRecordByFilter(
		CollectionCategoryMappings,
		"supplier_code = {:supplier} && supplier_category = {:category}",
		dbx.Params{
			"supplier": supplierCode,
			"category": rawCategory,
		},
	)
	if err != nil {
		return rawCategory, nil
	}

	if value := strings.TrimSpace(record.GetString("normalized_category")); value != "" {
		return value, nil
	}

	return rawCategory, nil
}

func (s *Service) recordAssetURL(record *core.Record) string {
	fileName := strings.TrimSpace(record.GetString("processed_image"))
	if fileName == "" {
		return ""
	}

	base := strings.TrimRight(s.cfg.App.PublicURL, "/")
	return fmt.Sprintf("%s/api/files/%s/%s/%s", base, record.Collection().Id, record.Id, url.PathEscape(fileName))
}

func (s *Service) recordPrimaryAssetURL(record *core.Record) string {
	if value := strings.TrimSpace(record.GetString("raw_image_url")); value != "" {
		return value
	}
	if isMockImageSource(record.GetString("processed_image_source")) {
		return ""
	}
	return s.recordAssetURL(record)
}

func (s *Service) recordConsumerAssetURL(record *core.Record) string {
	if isMockImageSource(record.GetString("processed_image_source")) {
		return s.recordPrimaryAssetURL(record)
	}
	if value := s.recordAssetURL(record); value != "" {
		return value
	}
	return s.recordPrimaryAssetURL(record)
}

func isMockImageSource(source string) bool {
	return strings.EqualFold(strings.TrimSpace(source), "mock")
}

func sourceAssetPrimaryImageURL(record *core.Record) string {
	if value := strings.TrimSpace(record.GetString("raw_image_url")); value != "" {
		return value
	}
	return ""
}

func sourceAssetConsumerImageURL(record *core.Record) string {
	if value := strings.TrimSpace(record.GetString("processed_image")); value != "" {
		return value
	}
	return ""
}

func sourceConversionRateFromSupplierRecord(record *core.Record) float64 {
	if value := record.GetFloat("conversion_rate"); value > 0 {
		return value
	}
	if value := readJSONNumber(record, "conversion_rate"); value > 0 {
		return value
	}
	return 1
}

func (s *Service) recordKey(supplierCode string, sku string) string {
	return strings.TrimSpace(supplierCode) + "::" + strings.TrimSpace(sku)
}

func splitCategoryPath(pathValue string, fallback string) []string {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		trimmed = strings.TrimSpace(fallback)
	}
	if trimmed == "" {
		return []string{"unclassified"}
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '>' || r == '\\'
	})
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return []string{"unclassified"}
	}
	return result
}

func supplierProcessPriority(record *core.Record) int {
	imageStatus := strings.ToLower(strings.TrimSpace(record.GetString("image_processing_status")))
	switch imageStatus {
	case ImageStatusPending, ImageStatusFailed:
		return 0
	case ImageStatusProcessing:
		return 1
	case ImageStatusProcessed:
		return 2
	default:
		return 3
	}
}

func slugifySegments(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := slugify(item)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return []string{"unclassified"}
	}
	return result
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func uniqueTrimmed(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeDuplicateNameKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func (s *Service) linkedVendureProductIDs(app core.App) (map[string]struct{}, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"vendure_product_id != ''",
		"",
		20000,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(records))
	for _, record := range records {
		productID := strings.TrimSpace(record.GetString("vendure_product_id"))
		if productID == "" {
			continue
		}
		result[productID] = struct{}{}
	}
	return result, nil
}

func containsTrimmed(items []string, target string) bool {
	needle := strings.TrimSpace(target)
	if needle == "" {
		return false
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	return false
}

func buildProcurementSummary(
	connector string,
	capabilities supplier.ConnectorCapabilities,
	externalRef string,
	deliveryAddress string,
	notes string,
	items []procurementCatalogItem,
) ProcurementSummary {
	summary := ProcurementSummary{
		Connector:       connector,
		Capabilities:    capabilities,
		ExternalRef:     externalRef,
		DeliveryAddress: deliveryAddress,
		Notes:           notes,
		Suppliers:       make([]ProcurementSupplierSummary, 0),
	}

	supplierIndexes := make(map[string]int, len(items))
	for _, item := range items {
		summary.ItemCount++
		summary.TotalQty += item.Quantity

		supplierIndex, ok := supplierIndexes[item.SupplierCode]
		if !ok {
			supplierIndex = len(summary.Suppliers)
			supplierIndexes[item.SupplierCode] = supplierIndex
			summary.Suppliers = append(summary.Suppliers, ProcurementSupplierSummary{
				SupplierCode: item.SupplierCode,
				Items:        make([]ProcurementSummaryItem, 0),
			})
		}

		supplierSummary := &summary.Suppliers[supplierIndex]
		line := procurementSummaryLine(item)
		supplierSummary.ItemCount++
		supplierSummary.TotalQty += line.Quantity
		supplierSummary.TotalCostAmount += line.CostAmount
		supplierSummary.TotalBusinessAmount += line.BusinessAmount
		supplierSummary.TotalConsumerAmount += line.ConsumerAmount
		if line.RiskLevel != "normal" {
			supplierSummary.RiskyItemCount++
			summary.RiskyItemCount++
		}
		supplierSummary.Items = append(supplierSummary.Items, line)

		summary.TotalCostAmount += line.CostAmount
		summary.TotalBusinessAmount += line.BusinessAmount
		summary.TotalConsumerAmount += line.ConsumerAmount
	}

	summary.SupplierCount = len(summary.Suppliers)
	return summary
}

func procurementSummaryLine(item procurementCatalogItem) ProcurementSummaryItem {
	costAmount := roundAmount(item.CostPrice * item.Quantity)
	businessAmount := roundAmount(item.BusinessPrice * item.Quantity)
	consumerAmount := roundAmount(item.ConsumerPrice * item.Quantity)

	return ProcurementSummaryItem{
		SupplierCode:       item.SupplierCode,
		OriginalSKU:        item.OriginalSKU,
		Title:              item.Title,
		NormalizedCategory: item.NormalizedCategory,
		Quantity:           item.Quantity,
		SalesUnit:          item.SalesUnit,
		CostPrice:          roundAmount(item.CostPrice),
		CostAmount:         costAmount,
		BusinessPrice:      roundAmount(item.BusinessPrice),
		BusinessAmount:     businessAmount,
		ConsumerPrice:      roundAmount(item.ConsumerPrice),
		ConsumerAmount:     consumerAmount,
		MarginRatio:        roundAmount(procurementMarginRatio(item.CostPrice, item.ConsumerPrice)),
		RiskLevel:          procurementRiskLevel(item.CostPrice, item.ConsumerPrice),
		NeedColdChain:      item.NeedColdChain,
	}
}

func renderProcurementCSV(summary ProcurementSummary) (string, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	rows := [][]string{
		{
			"supplier_code",
			"external_ref",
			"original_sku",
			"title",
			"normalized_category",
			"quantity",
			"sales_unit",
			"cost_price",
			"cost_amount",
			"b_price",
			"b_amount",
			"c_price",
			"c_amount",
			"margin_ratio",
			"risk_level",
			"need_cold_chain",
		},
	}

	for _, supplierSummary := range summary.Suppliers {
		for _, item := range supplierSummary.Items {
			rows = append(rows, []string{
				item.SupplierCode,
				summary.ExternalRef,
				item.OriginalSKU,
				item.Title,
				item.NormalizedCategory,
				fmt.Sprintf("%.2f", item.Quantity),
				item.SalesUnit,
				fmt.Sprintf("%.2f", item.CostPrice),
				fmt.Sprintf("%.2f", item.CostAmount),
				fmt.Sprintf("%.2f", item.BusinessPrice),
				fmt.Sprintf("%.2f", item.BusinessAmount),
				fmt.Sprintf("%.2f", item.ConsumerPrice),
				fmt.Sprintf("%.2f", item.ConsumerAmount),
				fmt.Sprintf("%.2f", item.MarginRatio),
				item.RiskLevel,
				fmt.Sprintf("%t", item.NeedColdChain),
			})
		}
	}

	if err := writer.WriteAll(rows); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func defaultProcurementExternalRef(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	return fmt.Sprintf("procurement-%d", time.Now().UnixMilli())
}

func positiveOr(value float64, fallback float64) float64 {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func procurementMarginRatio(costPrice float64, consumerPrice float64) float64 {
	if consumerPrice <= 0 {
		return 0
	}

	return 1 - (costPrice / consumerPrice)
}

func procurementRiskLevel(costPrice float64, consumerPrice float64) string {
	if consumerPrice <= 0 {
		return "missing_consumer_price"
	}

	ratio := costPrice / consumerPrice
	switch {
	case ratio >= 1:
		return "loss"
	case ratio >= 0.8:
		return "warning"
	default:
		return "normal"
	}
}

func roundAmount(value float64) float64 {
	return math.Round(value*100) / 100
}

func displayTitle(record *core.Record) string {
	return defaultString(record.GetString("normalized_title"), record.GetString("raw_title"))
}

func (diff supplierProductDiff) hasChanges() bool {
	return diff.ContentChanged || diff.PriceChanged || diff.StockChanged || diff.SpecChanged
}

func supplierProductSnapshotFromRecord(record *core.Record) supplierProductSnapshot {
	defaultOption := supplierRecordUnitOption(record, "")
	return supplierProductSnapshot{
		Title:            displayTitle(record),
		Category:         defaultString(record.GetString("normalized_category"), record.GetString("raw_category")),
		ImageURL:         strings.TrimSpace(record.GetString("raw_image_url")),
		GalleryURLs:      supplierRecordGalleryURLs(record),
		CostPrice:        roundAmount(record.GetFloat("cost_price")),
		BPrice:           roundAmount(record.GetFloat("b_price")),
		CPrice:           roundAmount(record.GetFloat("c_price")),
		CurrencyCode:     defaultString(record.GetString("currency_code"), "CNY"),
		SourceProductID:  defaultString(record.GetString("source_product_id"), readJSONAttribute(record, "source_product_id")),
		SourceType:       defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type")),
		SalesUnit:        defaultString(readJSONAttribute(record, "sales_unit"), "件"),
		ConversionRate:   roundAmount(sourceConversionRateFromSupplierRecord(record)),
		DefaultStockQty:  roundAmount(defaultOption.StockQty),
		DefaultStockText: strings.TrimSpace(defaultOption.StockText),
		UnitOptions:      normalizedSupplierUnitOptions(supplierRecordUnitOptions(record)),
	}
}

func supplierProductSnapshotHash(snapshot supplierProductSnapshot) (string, error) {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}

	sum := fnv.New128a()
	if _, err := sum.Write(encoded); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func diffSupplierProductSnapshots(previous supplierProductSnapshot, next supplierProductSnapshot) supplierProductDiff {
	diff := supplierProductDiff{
		ContentChanged: previous.Title != next.Title ||
			previous.Category != next.Category ||
			previous.ImageURL != next.ImageURL ||
			!slices.Equal(previous.GalleryURLs, next.GalleryURLs) ||
			previous.CurrencyCode != next.CurrencyCode,
		PriceChanged: priceChanged(previous.CostPrice, next.CostPrice) ||
			priceChanged(previous.BPrice, next.BPrice) ||
			priceChanged(previous.CPrice, next.CPrice),
		StockChanged: stockChanged(previous.DefaultStockQty, next.DefaultStockQty) ||
			strings.TrimSpace(previous.DefaultStockText) != strings.TrimSpace(next.DefaultStockText),
	}

	if len(previous.UnitOptions) != len(next.UnitOptions) {
		diff.SpecChanged = true
	}

	count := minInt(len(previous.UnitOptions), len(next.UnitOptions))
	for index := 0; index < count; index++ {
		left := previous.UnitOptions[index]
		right := next.UnitOptions[index]
		if strings.TrimSpace(left.UnitName) != strings.TrimSpace(right.UnitName) ||
			strings.TrimSpace(left.BaseUnit) != strings.TrimSpace(right.BaseUnit) ||
			roundAmount(left.Rate) != roundAmount(right.Rate) ||
			left.IsDefault != right.IsDefault {
			diff.SpecChanged = true
		}
		if priceChanged(left.Price, right.Price) {
			diff.PriceChanged = true
		}
		if stockChanged(left.StockQty, right.StockQty) || strings.TrimSpace(left.StockText) != strings.TrimSpace(right.StockText) {
			diff.StockChanged = true
		}
	}

	if previous.SourceProductID != next.SourceProductID ||
		previous.SourceType != next.SourceType ||
		previous.SalesUnit != next.SalesUnit ||
		roundAmount(previous.ConversionRate) != roundAmount(next.ConversionRate) {
		diff.SpecChanged = true
	}

	return diff
}

func supplierStatusFromSnapshot(snapshot supplierProductSnapshot, diff supplierProductDiff) string {
	if snapshotHasTrackedStock(snapshot) && snapshot.DefaultStockQty <= 0 {
		return SupplierStatusOutOfStock
	}
	switch {
	case diff.SpecChanged:
		return SupplierStatusSpecChanged
	case diff.PriceChanged:
		return SupplierStatusPriceChanged
	default:
		return SupplierStatusActive
	}
}

func signature(record *core.Record) string {
	snapshot, err := supplierProductSnapshotHash(supplierProductSnapshotFromRecord(record))
	if err != nil {
		return ""
	}
	return snapshot
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "_", "-", ".", "-", ",", "-", "(", "", ")", "", "[", "", "]", "")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func defaultString(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}

	return fallback
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}

func toMinorUnits(value float64) int {
	return int(math.Round(value * 100))
}

func readJSONAttribute(record *core.Record, key string) string {
	raw := strings.TrimSpace(record.GetString("supplier_payload"))
	if raw == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}

	if value, ok := payload[key]; ok {
		return fmt.Sprintf("%v", value)
	}

	return ""
}

func readJSONNumber(record *core.Record, key string) float64 {
	raw := strings.TrimSpace(record.GetString("supplier_payload"))
	if raw == "" {
		return 0
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0
	}

	value, ok := payload[key]
	if !ok {
		return 0
	}

	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		number, _ := v.Float64()
		return number
	default:
		return 0
	}
}

func normalizedSupplierUnitOptions(options []supplierPayloadUnitOption) []supplierPayloadUnitOption {
	normalized := make([]supplierPayloadUnitOption, 0, len(options))
	for _, option := range options {
		normalized = append(normalized, supplierPayloadUnitOption{
			UnitName:  strings.TrimSpace(option.UnitName),
			Price:     roundAmount(option.Price),
			BaseUnit:  strings.TrimSpace(option.BaseUnit),
			Rate:      roundAmount(option.Rate),
			IsDefault: option.IsDefault,
			StockQty:  roundAmount(option.StockQty),
			StockText: strings.TrimSpace(option.StockText),
		})
	}
	slices.SortStableFunc(normalized, func(left, right supplierPayloadUnitOption) int {
		leftKey := fmt.Sprintf("%s|%.2f|%t", left.UnitName, left.Rate, left.IsDefault)
		rightKey := fmt.Sprintf("%s|%.2f|%t", right.UnitName, right.Rate, right.IsDefault)
		return strings.Compare(leftKey, rightKey)
	})
	return normalized
}

func priceChanged(previous float64, next float64) bool {
	return roundAmount(previous) != roundAmount(next)
}

func stockChanged(previous float64, next float64) bool {
	return roundAmount(previous) != roundAmount(next)
}

func snapshotHasTrackedStock(snapshot supplierProductSnapshot) bool {
	if snapshot.DefaultStockQty > 0 || strings.TrimSpace(snapshot.DefaultStockText) != "" {
		return true
	}
	return len(snapshot.UnitOptions) > 0
}

func supplierRecordHasUnitOptions(record *core.Record) bool {
	return len(supplierRecordUnitOptions(record)) > 0
}

func supplierRecordUnitOptionExact(record *core.Record, unitName string) (supplierPayloadUnitOption, bool) {
	trimmed := strings.TrimSpace(unitName)
	if trimmed == "" {
		option := supplierRecordUnitOption(record, "")
		return option, strings.TrimSpace(option.UnitName) != ""
	}
	for _, option := range supplierRecordUnitOptions(record) {
		if strings.EqualFold(strings.TrimSpace(option.UnitName), trimmed) {
			return option, true
		}
	}
	return supplierPayloadUnitOption{}, false
}

func supplierRecordAvailableStock(record *core.Record, unitName string) (float64, bool) {
	if option, ok := supplierRecordUnitOptionExact(record, unitName); ok {
		return roundAmount(option.StockQty), true
	}
	if supplierRecordHasUnitOptions(record) {
		option := supplierRecordUnitOption(record, "")
		return roundAmount(option.StockQty), true
	}
	raw := strings.TrimSpace(record.GetString("supplier_payload"))
	if raw == "" {
		return 0, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0, false
	}

	for _, key := range []string{"stockQty", "stock_qty"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch cast := value.(type) {
		case float64:
			return roundAmount(cast), true
		case int:
			return roundAmount(float64(cast)), true
		case int64:
			return roundAmount(float64(cast)), true
		case json.Number:
			number, _ := cast.Float64()
			return roundAmount(number), true
		}
	}
	return 0, false
}

func readJSONArrayAttributes(record *core.Record, key string) []string {
	raw := strings.TrimSpace(record.GetString("supplier_payload"))
	if raw == "" {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}

	value, ok := payload[key]
	if !ok {
		return nil
	}

	items, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return uniqueTrimmed(result)
}

func supplierRecordGalleryURLs(record *core.Record) []string {
	return uniqueTrimmed(readJSONArrayAttributes(record, "gallery_urls"))
}

func isSyncedSingleImageSkeletonRecord(record *core.Record) bool {
	if !strings.EqualFold(strings.TrimSpace(record.GetString("sync_status")), StatusSynced) {
		return false
	}
	if strings.TrimSpace(record.GetString("vendure_product_id")) == "" {
		return false
	}
	sourceType := strings.ToLower(strings.TrimSpace(defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type"))))
	if sourceType == "raw_detail" || sourceType == "rr_detail" {
		return false
	}
	if !strings.Contains(sourceType, "skeleton") {
		return false
	}
	return len(supplierRecordGalleryURLs(record)) <= 1
}

type supplierPayloadUnitOption struct {
	UnitName  string  `json:"unitName"`
	Price     float64 `json:"price"`
	BaseUnit  string  `json:"baseUnit"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"isDefault"`
	StockQty  float64 `json:"stockQty"`
	StockText string  `json:"stockText"`
}

type supplierPayloadOrderUnit struct {
	UnitID      string  `json:"unitId"`
	UnitName    string  `json:"unitName"`
	Rate        float64 `json:"rate"`
	IsBase      bool    `json:"isBase"`
	IsDefault   bool    `json:"isDefault"`
	AllowOrder  bool    `json:"allowOrder"`
	MinOrderQty float64 `json:"minOrderQty"`
	MaxOrderQty float64 `json:"maxOrderQty"`
}

type vendureVariantState struct {
	Key              string  `json:"key"`
	UnitName         string  `json:"unitName"`
	Rate             float64 `json:"rate"`
	SKU              string  `json:"sku"`
	Price            int     `json:"price"`
	BusinessPrice    int     `json:"businessPrice"`
	IsDefault        bool    `json:"isDefault"`
	VendureVariantID string  `json:"vendureVariantId"`
}

func supplierRecordVariantPayloads(record *core.Record, fallbackStock int) []vendure.ProductVariantPayload {
	sourceProductID := defaultString(record.GetString("source_product_id"), readJSONAttribute(record, "source_product_id"))
	sourceType := defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type"))
	supplierCode := record.GetString("supplier_code")
	currencyCode := defaultString(record.GetString("currency_code"), "CNY")
	defaultUnit := defaultString(readJSONAttribute(record, "sales_unit"), "件")
	defaultRate := sourceConversionRateFromSupplierRecord(record)
	defaultConsumerPrice := toMinorUnits(record.GetFloat("c_price"))
	defaultBusinessPrice := toMinorUnits(record.GetFloat("b_price"))
	supplierCostPrice := toMinorUnits(record.GetFloat("cost_price"))
	baseSKU := strings.TrimSpace(record.GetString("original_sku"))
	storedVariants := supplierRecordVendureVariantStates(record)
	storedByKey := make(map[string]vendureVariantState, len(storedVariants))
	for _, item := range storedVariants {
		if strings.TrimSpace(item.Key) == "" {
			continue
		}
		storedByKey[item.Key] = item
	}

	unitOptions := supplierRecordUnitOptions(record)
	if len(unitOptions) == 0 {
		key := supplierVariantKey(sourceProductID, defaultUnit, defaultRate)
		return []vendure.ProductVariantPayload{
			{
				Key:               key,
				Name:              defaultUnit,
				SKU:               baseSKU,
				ConsumerPrice:     defaultConsumerPrice,
				BusinessPrice:     positiveIntOr(defaultBusinessPrice, defaultConsumerPrice),
				ConversionRate:    positiveFloatOr(defaultRate, 1),
				DefaultStock:      fallbackStock,
				SalesUnit:         defaultUnit,
				VendureVariant:    record.GetString("vendure_variant_id"),
				IsDefault:         true,
				SourceProductID:   sourceProductID,
				SourceType:        sourceType,
				SupplierCode:      supplierCode,
				SupplierCostPrice: supplierCostPrice,
				CurrencyCode:      currencyCode,
			},
		}
	}

	variants := make([]vendure.ProductVariantPayload, 0, len(unitOptions))
	for _, option := range unitOptions {
		unitName := defaultString(option.UnitName, defaultUnit)
		rate := positiveFloatOr(option.Rate, 1)
		isDefault := option.IsDefault
		if !isDefault && unitName == defaultUnit && rate == positiveFloatOr(defaultRate, 1) {
			isDefault = true
		}
		price := toMinorUnits(option.Price)
		if isDefault {
			price = positiveIntOr(defaultConsumerPrice, price)
		}
		businessPrice := price
		if isDefault {
			businessPrice = positiveIntOr(defaultBusinessPrice, price)
		}
		key := supplierVariantKey(sourceProductID, unitName, rate)
		stored := storedByKey[key]
		vendureVariantID := strings.TrimSpace(stored.VendureVariantID)
		if vendureVariantID == "" && isDefault {
			vendureVariantID = strings.TrimSpace(record.GetString("vendure_variant_id"))
		}
		sku := defaultString(strings.TrimSpace(stored.SKU), variantSKU(baseSKU, key, isDefault))
		variants = append(variants, vendure.ProductVariantPayload{
			Key:               key,
			Name:              unitName,
			SKU:               sku,
			ConsumerPrice:     positiveIntOr(price, defaultConsumerPrice),
			BusinessPrice:     positiveIntOr(businessPrice, positiveIntOr(price, defaultBusinessPrice)),
			ConversionRate:    rate,
			DefaultStock:      unitStock(option.StockQty, fallbackStock),
			SalesUnit:         unitName,
			VendureVariant:    vendureVariantID,
			IsDefault:         isDefault,
			SourceProductID:   sourceProductID,
			SourceType:        sourceType,
			SupplierCode:      supplierCode,
			SupplierCostPrice: supplierCostPrice,
			CurrencyCode:      currencyCode,
		})
	}

	if !hasDefaultVariant(variants) && len(variants) > 0 {
		variants[0].IsDefault = true
		if strings.TrimSpace(variants[0].VendureVariant) == "" {
			variants[0].VendureVariant = strings.TrimSpace(record.GetString("vendure_variant_id"))
		}
	}
	return variants
}

func supplierRecordUnitOptions(record *core.Record) []supplierPayloadUnitOption {
	raw := strings.TrimSpace(record.GetString("supplier_payload"))
	if raw == "" {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	value, ok := payload["unit_options"]
	if !ok {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	options := make([]supplierPayloadUnitOption, 0)
	if err := json.Unmarshal(encoded, &options); err != nil {
		return nil
	}
	slices.SortStableFunc(options, func(a, b supplierPayloadUnitOption) int {
		if a.IsDefault == b.IsDefault {
			return strings.Compare(strings.TrimSpace(a.UnitName), strings.TrimSpace(b.UnitName))
		}
		if a.IsDefault {
			return -1
		}
		return 1
	})
	return options
}

func supplierRecordUnitOption(record *core.Record, unitName string) supplierPayloadUnitOption {
	unitName = strings.TrimSpace(unitName)
	options := supplierRecordUnitOptions(record)
	if unitName != "" {
		for _, option := range options {
			if strings.EqualFold(strings.TrimSpace(option.UnitName), unitName) {
				return option
			}
		}
	}
	for _, option := range options {
		if option.IsDefault {
			return option
		}
	}
	if len(options) > 0 {
		return options[0]
	}
	return supplierPayloadUnitOption{}
}

func supplierRecordOrderUnits(record *core.Record) []supplierPayloadOrderUnit {
	raw := strings.TrimSpace(record.GetString("order_units_json"))
	units := make([]supplierPayloadOrderUnit, 0)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &units); err != nil {
			return nil
		}
	} else {
		payloadRaw := strings.TrimSpace(record.GetString("supplier_payload"))
		if payloadRaw == "" {
			return nil
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
			return nil
		}
		value, ok := payload["order_units"]
		if !ok {
			return nil
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(encoded, &units); err != nil {
			return nil
		}
	}
	slices.SortStableFunc(units, func(a, b supplierPayloadOrderUnit) int {
		if a.IsDefault == b.IsDefault {
			return strings.Compare(strings.TrimSpace(a.UnitName), strings.TrimSpace(b.UnitName))
		}
		if a.IsDefault {
			return -1
		}
		return 1
	})
	return units
}

func supplierRecordOrderUnit(record *core.Record, unitName string) supplierPayloadOrderUnit {
	unitName = strings.TrimSpace(unitName)
	units := supplierRecordOrderUnits(record)
	if unitName != "" {
		for _, unit := range units {
			if strings.EqualFold(strings.TrimSpace(unit.UnitName), unitName) {
				return unit
			}
		}
	}
	for _, unit := range units {
		if unit.IsDefault {
			return unit
		}
	}
	if len(units) > 0 {
		return units[0]
	}
	return supplierPayloadOrderUnit{}
}

func supplierRecordVendureVariantStates(record *core.Record) []vendureVariantState {
	raw := strings.TrimSpace(record.GetString("vendure_variants_json"))
	if raw == "" {
		return nil
	}
	states := make([]vendureVariantState, 0)
	if err := json.Unmarshal([]byte(raw), &states); err != nil {
		return nil
	}
	return states
}

func defaultVariantPayload(variants []vendure.ProductVariantPayload) vendure.ProductVariantPayload {
	for _, variant := range variants {
		if variant.IsDefault {
			return variant
		}
	}
	if len(variants) > 0 {
		return variants[0]
	}
	return vendure.ProductVariantPayload{}
}

func hasDefaultVariant(variants []vendure.ProductVariantPayload) bool {
	for _, variant := range variants {
		if variant.IsDefault {
			return true
		}
	}
	return false
}

func supplierVariantKey(sourceProductID string, unitName string, rate float64) string {
	sourceProductID = strings.TrimSpace(sourceProductID)
	unitName = strings.TrimSpace(unitName)
	if sourceProductID == "" || unitName == "" {
		return ""
	}
	return fmt.Sprintf("%s#%s#%.6f", sourceProductID, unitName, rate)
}

func variantSKU(baseSKU string, unitKey string, isDefault bool) string {
	baseSKU = strings.TrimSpace(baseSKU)
	if isDefault || baseSKU == "" {
		return baseSKU
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(unitKey))
	sum := hasher.Sum(nil)
	return baseSKU + "__unit_" + strings.ToLower(hex.EncodeToString(sum[:4]))
}

func unitStock(stockQty float64, fallback int) int {
	if stockQty > 0 {
		return int(math.Ceil(stockQty))
	}
	return fallback
}

func positiveIntOr(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func positiveFloatOr(value float64, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func supplierRecordReleaseCategoryKey(record *core.Record) string {
	return defaultString(
		readJSONAttribute(record, "release_category_key"),
		readJSONAttribute(record, "category_key"),
	)
}

func supplierRecordObservedCategoryKeys(record *core.Record) []string {
	keys := readJSONArrayAttributes(record, "observed_category_keys")
	if len(keys) == 0 {
		keys = readJSONArrayAttributes(record, "category_keys")
	}
	return uniqueTrimmed(append(keys, supplierRecordReleaseCategoryKey(record)))
}

func assetFileName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return path.Base(raw)
}

func (s *Service) logProcurementAction(app core.App, record *core.Record, actionType string, status string, message string, actor ProcurementActionActor, note string, details any) {
	collection, err := app.FindCollectionByNameOrId(CollectionProcurementActionLogs)
	if err != nil {
		return
	}
	logRecord := core.NewRecord(collection)
	logRecord.Set("order_id", record.Id)
	logRecord.Set("external_ref", strings.TrimSpace(record.GetString("external_ref")))
	logRecord.Set("action_type", strings.TrimSpace(actionType))
	logRecord.Set("status", strings.TrimSpace(status))
	logRecord.Set("message", strings.TrimSpace(message))
	logRecord.Set("actor_email", strings.TrimSpace(actor.Email))
	logRecord.Set("actor_name", strings.TrimSpace(actor.Name))
	logRecord.Set("note", strings.TrimSpace(note))
	if details != nil {
		if err := setJSONField(logRecord, "details_json", details); err != nil {
			return
		}
	}
	_ = app.Save(logRecord)
}

func (s *Service) listRecentProcurementActions(app core.App, limit int) ([]ProcurementActionLog, error) {
	if limit <= 0 {
		limit = 8
	}
	records, err := app.FindRecordsByFilter(CollectionProcurementActionLogs, "", "-created", limit, 0, nil)
	if err != nil {
		return nil, err
	}
	items := make([]ProcurementActionLog, 0, len(records))
	for _, record := range records {
		items = append(items, ProcurementActionLog{
			ID:          record.Id,
			OrderID:     record.GetString("order_id"),
			ExternalRef: record.GetString("external_ref"),
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

func (s *Service) ListProcurementActions(app core.App, orderID string, limit int) ([]ProcurementActionLog, error) {
	if limit <= 0 {
		limit = 20
	}
	filter := ""
	params := dbx.Params{}
	orderID = strings.TrimSpace(orderID)
	if orderID != "" {
		filter = "order_id = {:order_id}"
		params["order_id"] = orderID
	}
	sortExpr, err := procurementActionSortExpr(app)
	if err != nil {
		return nil, err
	}
	records, err := app.FindRecordsByFilter(CollectionProcurementActionLogs, filter, sortExpr, limit, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]ProcurementActionLog, 0, len(records))
	for _, record := range records {
		items = append(items, ProcurementActionLog{
			ID:          record.Id,
			OrderID:     record.GetString("order_id"),
			ExternalRef: record.GetString("external_ref"),
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

func procurementActionSortExpr(app core.App) (string, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionProcurementActionLogs)
	if err != nil {
		return "", err
	}
	if collection.Fields.GetByName("created") != nil {
		return "-created", nil
	}
	return "-id", nil
}

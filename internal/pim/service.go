package pim

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
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

	StatusPending      = "pending"
	StatusAIProcessing = "ai_processing"
	StatusReady        = "ready"
	StatusApproved     = "approved"
	StatusSynced       = "synced"
	StatusOffline      = "offline"
	StatusError        = "error"

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

type ProcurementItemRequest struct {
	SupplierCode string  `json:"supplierCode"`
	OriginalSKU  string  `json:"originalSku"`
	Quantity     float64 `json:"quantity"`
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

type ProcurementSubmitResponse struct {
	Summary ProcurementSummary             `json:"summary"`
	Results []supplier.PurchaseOrderResult `json:"results"`
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
	VendureProductID   string  `json:"vendureProductId"`
	VendureVariantID   string  `json:"vendureVariantId"`
	Reason             string  `json:"reason"`
	HasProcessedImage  bool    `json:"hasProcessedImage"`
	HasConsumerImage   bool    `json:"hasConsumerImage"`
	ReadyForPreview    bool    `json:"readyForPreview"`
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
	PublishedRootCount   int                                `json:"publishedRootCount"`
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
	cfg               config.Config
	connector         supplier.Connector
	processor         image.Processor
	vendure           *vendure.Client
	lock              sync.Mutex
	targetSyncMu      sync.Mutex
	activeTargetSyncs map[string]string
	sourceAssetMu     sync.Mutex
	activeAssetLoads  map[string]*SourceAssetDownloadProgress
	activeAssetProcs  map[string]*SourceAssetProcessProgress
}

func NewService(cfg config.Config) *Service {
	var connector supplier.Connector
	switch strings.ToLower(cfg.Supplier.Connector) {
	case "file":
		connector = supplier.NewFileConnector(cfg.Supplier.FilePath, cfg.Supplier.Code)
	default:
		connector = supplier.NewFileConnector(cfg.Supplier.FilePath, cfg.Supplier.Code)
	}

	return &Service{
		cfg:               cfg,
		connector:         connector,
		processor:         image.NewProcessor(cfg.Image),
		vendure:           vendure.NewClient(cfg.Vendure),
		activeTargetSyncs: make(map[string]string),
		activeAssetLoads:  make(map[string]*SourceAssetDownloadProgress),
		activeAssetProcs:  make(map[string]*SourceAssetProcessProgress),
	}
}

func (s *Service) ConnectorCapabilities() supplier.ConnectorCapabilities {
	return s.connector.Capabilities()
}

func (s *Service) ProcurementSummary(ctx context.Context, app core.App, req ProcurementRequest) (ProcurementSummary, error) {
	items, err := s.resolveProcurementItems(ctx, app, req.Items)
	if err != nil {
		return ProcurementSummary{}, err
	}

	summary := buildProcurementSummary(
		s.cfg.Supplier.Connector,
		s.connector.Capabilities(),
		defaultProcurementExternalRef(req.ExternalRef),
		strings.TrimSpace(req.DeliveryAddress),
		strings.TrimSpace(req.Notes),
		items,
	)

	return summary, nil
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

	results := make([]supplier.PurchaseOrderResult, 0, len(summary.Suppliers))
	for _, supplierSummary := range summary.Suppliers {
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
			results = append(results, supplier.PurchaseOrderResult{
				SupplierCode: supplierSummary.SupplierCode,
				ExternalRef:  summary.ExternalRef,
				Mode:         "error",
				Accepted:     false,
				Message:      submitErr.Error(),
			})
			continue
		}

		results = append(results, result)
	}

	return ProcurementSubmitResponse{
		Summary: summary,
		Results: results,
	}, nil
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

	return result, nil
}

func procurementOrderSortExpr(app core.App) (string, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionProcurementOrders)
	if err != nil {
		return "", err
	}

	if collection.Fields.GetByName("created") != nil {
		return "-created", nil
	}

	return "-id", nil
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
	s.lock.Lock()
	defer s.lock.Unlock()

	items, err := s.connector.Fetch(ctx)
	if err != nil {
		return Result{Action: "harvest"}, err
	}

	result := Result{Action: "harvest"}
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		key := s.recordKey(item.SupplierCode, item.OriginalSKU)
		seen[key] = struct{}{}

		changed, created, err := s.upsertSupplierProduct(ctx, app, item)
		if err != nil {
			result.Failed++
			app.Logger().Error("harvest upsert failed", "sku", item.OriginalSKU, "error", err)
			continue
		}

		result.Processed++
		if created {
			result.Created++
			continue
		}

		if changed {
			result.Updated++
		} else {
			result.Skipped++
		}
	}

	offlineCount, err := s.markMissingProductsOffline(ctx, app, seen)
	if err != nil {
		return result, err
	}
	result.Offline = offlineCount

	return result, nil
}

func (s *Service) ProcessPending(ctx context.Context, app core.App, limit int) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"(sync_status = {:pending} || sync_status = {:error}) && raw_image_url != ''",
		"updated",
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

func (s *Service) SyncApproved(ctx context.Context, app core.App, limit int) (Result, error) {
	records, err := app.FindRecordsByFilter(
		CollectionSupplierProducts,
		"sync_status = {:status}",
		"-updated",
		limit,
		0,
		dbx.Params{"status": StatusApproved},
	)
	if err != nil {
		return Result{Action: "sync"}, err
	}

	result := Result{Action: "sync"}
	for _, record := range records {
		if err := s.syncRecord(ctx, app, record); err != nil {
			result.Failed++
			app.Logger().Error("vendure sync failed", "recordId", record.Id, "error", err)
			continue
		}

		result.Processed++
	}

	return result, nil
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

	previousSignature := signature(record)
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

	if payload, err := json.Marshal(item.Payload); err == nil {
		record.Set("supplier_payload", string(payload))
	}

	if created {
		record.Set("sync_status", StatusPending)
		record.Set("image_processing_status", ImageStatusPending)
	} else if previousSignature != signature(record) {
		if strings.TrimSpace(record.GetString("vendure_product_id")) != "" {
			record.Set("sync_status", StatusApproved)
		} else {
			record.Set("sync_status", StatusPending)
		}
		record.Set("image_processing_status", ImageStatusPending)
		record.Set("last_sync_error", "")
		record.Set("image_processing_error", "")
	}

	if err := app.Save(record); err != nil {
		return false, created, err
	}

	return previousSignature != signature(record), created, nil
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
	record.Set("sync_status", StatusReady)
	record.Set("processed_image_source", result.Source)
	return app.Save(record)
}

func (s *Service) syncRecord(ctx context.Context, app core.App, record *core.Record) error {
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
	record.Set("last_sync_error", "")
	record.Set("sync_status", StatusSynced)
	record.Set("last_synced_at", time.Now().Format(time.RFC3339))
	return app.Save(record)
}

func (s *Service) buildVendurePayload(app core.App, record *core.Record) vendure.ProductPayload {
	assetURL := s.recordPrimaryAssetURL(record)
	cEndAssetURL := s.recordConsumerAssetURL(record)
	assetURLs, assetNames := s.recordGalleryAssetURLs(app, record)
	if assetURL == "" && len(assetURLs) > 0 {
		assetURL = assetURLs[0]
	}
	return vendure.ProductPayload{
		Name:              displayTitle(record),
		Slug:              slugify(record.GetString("supplier_code") + "-" + record.GetString("original_sku") + "-" + displayTitle(record)),
		Description:       defaultString(record.GetString("marketing_description"), record.GetString("raw_description")),
		SKU:               record.GetString("original_sku"),
		CurrencyCode:      defaultString(record.GetString("currency_code"), s.cfg.Vendure.CurrencyCode),
		ConsumerPrice:     toMinorUnits(record.GetFloat("c_price")),
		AssetURL:          assetURL,
		AssetName:         assetFileName(assetURL),
		AssetURLs:         assetURLs,
		AssetNames:        assetNames,
		CEndAssetURL:      cEndAssetURL,
		CEndAssetName:     assetFileName(cEndAssetURL),
		BusinessPrice:     toMinorUnits(record.GetFloat("b_price")),
		SupplierCode:      record.GetString("supplier_code"),
		SupplierCostPrice: toMinorUnits(record.GetFloat("cost_price")),
		ConversionRate:    sourceConversionRateFromSupplierRecord(record),
		SourceProductID:   defaultString(record.GetString("source_product_id"), readJSONAttribute(record, "source_product_id")),
		SourceType:        defaultString(record.GetString("source_type"), readJSONAttribute(record, "source_type")),
		TargetAudience:    defaultString(record.GetString("target_audience"), "ALL"),
		DefaultStock:      s.cfg.Workflow.DefaultStockOnHand,
		SalesUnit:         defaultString(readJSONAttribute(record, "sales_unit"), "件"),
		VendureProduct:    record.GetString("vendure_product_id"),
		VendureVariant:    record.GetString("vendure_variant_id"),
		NeedColdChain:     strings.EqualFold(readJSONAttribute(record, "need_cold_chain"), "true"),
	}
}

func (s *Service) recordGalleryAssetURLs(app core.App, record *core.Record) ([]string, []string) {
	productID := strings.TrimSpace(record.GetString("source_product_id"))
	if productID == "" {
		productID = strings.TrimSpace(record.GetString("product_id"))
	}
	if productID == "" {
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
	if categoryKey == "" {
		return nil, nil
	}

	categoryKeys := []string{categoryKey}
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
	if limit <= 0 {
		limit = 12
	}
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
		for _, record := range allProducts {
			status := strings.ToLower(strings.TrimSpace(record.GetString("sync_status")))
			switch status {
			case StatusApproved:
				summary.ReadyProductCount++
			case StatusSynced:
				summary.SyncedProductCount++
			case StatusError:
				summary.ErrorProductCount++
			}
		}
	}
	for _, record := range records {
		summary.Products = append(summary.Products, backendReleaseProductItemFromRecord(record))
	}
	summary.RecommendedProducts = pickRecommendedBackendReleaseProducts(records, 3)

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
		VendureProductID:   record.GetString("vendure_product_id"),
		VendureVariantID:   record.GetString("vendure_variant_id"),
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

		items = append(items, procurementCatalogItem{
			SupplierCode:       supplierCode,
			OriginalSKU:        sku,
			Title:              displayTitle(record),
			NormalizedCategory: defaultString(record.GetString("normalized_category"), record.GetString("raw_category")),
			Quantity:           quantity,
			SalesUnit:          defaultString(readJSONAttribute(record, "sales_unit"), "件"),
			CostPrice:          record.GetFloat("cost_price"),
			BusinessPrice:      record.GetFloat("b_price"),
			ConsumerPrice:      record.GetFloat("c_price"),
			NeedColdChain:      strings.EqualFold(readJSONAttribute(record, "need_cold_chain"), "true"),
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
		return next == ProcurementStatusReviewed || next == ProcurementStatusExported || next == ProcurementStatusCanceled
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

func (s *Service) markMissingProductsOffline(ctx context.Context, app core.App, seen map[string]struct{}) (int, error) {
	records, err := app.FindAllRecords(CollectionSupplierProducts)
	if err != nil {
		return 0, err
	}

	offlineCount := 0
	for _, record := range records {
		if record.GetString("supplier_code") != s.cfg.Supplier.Code {
			continue
		}

		key := s.recordKey(record.GetString("supplier_code"), record.GetString("original_sku"))
		if _, ok := seen[key]; ok {
			continue
		}

		if record.GetString("sync_status") == StatusOffline {
			continue
		}

		record.Set("sync_status", StatusOffline)
		record.Set("offline_at", time.Now().Format(time.RFC3339))
		if err := app.Save(record); err != nil {
			return offlineCount, err
		}

		if vendureID := record.GetString("vendure_product_id"); vendureID != "" {
			if err := s.vendure.DisableProduct(ctx, vendureID); err != nil {
				app.Logger().Error("disable vendure product failed", "productId", vendureID, "error", err)
			}
		}

		offlineCount++
	}

	return offlineCount, nil
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
	return s.recordAssetURL(record)
}

func (s *Service) recordConsumerAssetURL(record *core.Record) string {
	if value := s.recordAssetURL(record); value != "" {
		return value
	}
	return s.recordPrimaryAssetURL(record)
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

func signature(record *core.Record) string {
	return strings.Join([]string{
		record.GetString("raw_title"),
		record.GetString("raw_category"),
		record.GetString("raw_image_url"),
		fmt.Sprintf("%.2f", record.GetFloat("cost_price")),
		fmt.Sprintf("%.2f", record.GetFloat("b_price")),
		fmt.Sprintf("%.2f", record.GetFloat("c_price")),
	}, "|")
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

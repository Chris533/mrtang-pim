package pim

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"path"
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
	CollectionSupplierProducts = "supplier_products"
	CollectionCategoryMappings = "category_mappings"

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

type Service struct {
	cfg       config.Config
	connector supplier.Connector
	processor image.Processor
	vendure   *vendure.Client
	lock      sync.Mutex
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
		cfg:       cfg,
		connector: connector,
		processor: image.NewProcessor(cfg.Image),
		vendure:   vendure.NewClient(cfg.Vendure),
	}
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
	payload := vendure.ProductPayload{
		Name:           displayTitle(record),
		Slug:           slugify(record.GetString("supplier_code") + "-" + record.GetString("original_sku") + "-" + displayTitle(record)),
		Description:    defaultString(record.GetString("marketing_description"), record.GetString("raw_description")),
		SKU:            record.GetString("original_sku"),
		CurrencyCode:   defaultString(record.GetString("currency_code"), s.cfg.Vendure.CurrencyCode),
		ConsumerPrice:  toMinorUnits(record.GetFloat("c_price")),
		AssetURL:       s.recordAssetURL(record),
		AssetName:      path.Base(s.recordAssetURL(record)),
		BusinessPrice:  toMinorUnits(record.GetFloat("b_price")),
		DefaultStock:   s.cfg.Workflow.DefaultStockOnHand,
		SalesUnit:      defaultString(readJSONAttribute(record, "sales_unit"), "件"),
		VendureProduct: record.GetString("vendure_product_id"),
		VendureVariant: record.GetString("vendure_variant_id"),
		NeedColdChain:  strings.EqualFold(readJSONAttribute(record, "need_cold_chain"), "true"),
	}

	result, err := s.vendure.SyncProduct(ctx, payload)
	if err != nil {
		record.Set("last_sync_error", err.Error())
		record.Set("sync_status", StatusError)
		_ = app.Save(record)
		return err
	}

	record.Set("vendure_product_id", coalesce(result.ProductID, record.GetString("vendure_product_id")))
	record.Set("vendure_variant_id", coalesce(result.VariantID, record.GetString("vendure_variant_id")))
	record.Set("last_sync_error", "")
	record.Set("sync_status", StatusSynced)
	record.Set("last_synced_at", time.Now().Format(time.RFC3339))
	return app.Save(record)
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

func (s *Service) recordKey(supplierCode string, sku string) string {
	return strings.TrimSpace(supplierCode) + "::" + strings.TrimSpace(sku)
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

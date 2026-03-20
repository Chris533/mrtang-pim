package pim

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappapi "mrtang-pim/internal/miniapp/api"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/supplier"
)

type miniappCartOrderSubmitter struct {
	cfg     config.Config
	miniapp *miniappservice.Service
}

type miniappCartOrderLine struct {
	SupplierCode string
	OriginalSKU  string
	SalesUnit    string
	SpuID        string
	SkuID        string
	SkuName      string
	UnitID       string
	UnitRate     float64
	FreightQty   float64
	Quantity     float64
	UnitPrice    float64
}

type rawCartLine struct {
	ID     string
	SkuID  string
	UnitID string
	Num    float64
	Cost   float64
}

type rawDetailLine struct {
	ID       string
	SkuID    string
	UnitName string
	Num      float64
	TotPrice float64
}

func newMiniappCartOrderSubmitter(cfg config.Config) *miniappCartOrderSubmitter {
	if !strings.EqualFold(strings.TrimSpace(cfg.Supplier.Connector), "miniapp_cart_order") {
		return nil
	}
	return &miniappCartOrderSubmitter{
		cfg:     cfg,
		miniapp: miniappservice.New(newMiniappActionSource(cfg), nil),
	}
}

func (s *miniappCartOrderSubmitter) Capabilities() supplier.ConnectorCapabilities {
	return supplier.ConnectorCapabilities{
		FetchProducts:       false,
		SubmitPurchaseOrder: true,
		ExportPurchaseOrder: false,
	}
}

func (s *miniappCartOrderSubmitter) Submit(ctx context.Context, app core.App, summary ProcurementSummary, progress ProcurementProgressLogger) []supplier.PurchaseOrderResult {
	results := make([]supplier.PurchaseOrderResult, 0, len(summary.Suppliers))
	for _, supplierSummary := range summary.Suppliers {
		result, err := s.submitSupplierSummary(ctx, app, summary, supplierSummary, progress)
		if err != nil {
			if progress != nil {
				progress("submit_order_progress", "error", err.Error(), map[string]any{
					"supplierCode": supplierSummary.SupplierCode,
				})
			}
			results = append(results, supplier.PurchaseOrderResult{
				SupplierCode: supplierSummary.SupplierCode,
				ExternalRef:  summary.ExternalRef,
				Mode:         "miniapp_cart_order_error",
				Accepted:     false,
				Message:      err.Error(),
			})
			continue
		}
		results = append(results, result)
	}
	return results
}

func (s *miniappCartOrderSubmitter) submitSupplierSummary(ctx context.Context, app core.App, summary ProcurementSummary, supplierSummary ProcurementSupplierSummary, progress ProcurementProgressLogger) (supplier.PurchaseOrderResult, error) {
	if s.miniapp == nil {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("miniapp cart-order submitter is not initialized")
	}
	if !strings.EqualFold(strings.TrimSpace(s.cfg.MiniApp.SourceMode), "raw") {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("SUPPLIER_CONNECTOR=miniapp_cart_order requires MINIAPP_SOURCE_MODE=raw")
	}
	customerID := strings.TrimSpace(s.cfg.MiniApp.RawCustomerID)
	if customerID == "" {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("MINIAPP_RAW_CUSTOMER_ID is required for miniapp cart-order submit")
	}

	targetLines, err := s.loadSupplierLines(ctx, app, supplierSummary)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	targetLines = aggregateMiniappCartOrderLines(targetLines)
	if progress != nil {
		progress("submit_order_progress", "running", "resolved supplier lines", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
			"lineCount":    len(targetLines),
		})
	}
	if len(targetLines) == 0 {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("no supplier lines resolved for %s", supplierSummary.SupplierCode)
	}

	if progress != nil {
		progress("submit_order_progress", "running", "fetching supplier cart list", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	if _, err := s.fetchCartList(ctx); err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	if progress != nil {
		progress("submit_order_progress", "running", "adding supplier items to cart", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	if _, err := s.miniapp.ExecuteCartOperation(ctx, "add", buildMiniappAddCartBody(targetLines)); err != nil {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("add supplier items to cart: %w", err)
	}

	if progress != nil {
		progress("submit_order_progress", "running", "reloading supplier cart list", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	cartLines, err := s.fetchCartList(ctx)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	if progress != nil {
		progress("submit_order_progress", "running", "normalizing supplier cart quantities", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	if err := s.normalizeCartQuantities(ctx, targetLines, cartLines); err != nil {
		return supplier.PurchaseOrderResult{}, err
	}

	if progress != nil {
		progress("submit_order_progress", "running", "loading supplier cart detail", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	detailLines, err := s.fetchCartDetail(ctx)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	cartIDs, freightItems, goodsAmount, err := matchCartDetailLines(targetLines, detailLines)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}

	if progress != nil {
		progress("submit_order_progress", "running", "loading supplier delivery address", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	address, err := s.fetchDeliveryAddress(ctx, customerID)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	deliveryMethodID := firstNonEmptyString(
		stringFromAny(address["deliveryMethodId"]),
		stringFromAny(address["deliveryId"]),
	)
	if deliveryMethodID == "" {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("supplier delivery method is empty")
	}

	if progress != nil {
		progress("submit_order_progress", "running", "loading supplier freight cost", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
		})
	}
	freight, err := s.fetchFreightCost(ctx, customerID, deliveryMethodID, address, freightItems)
	if err != nil {
		return supplier.PurchaseOrderResult{}, err
	}
	submitBody := buildMiniappSubmitBody(summary, customerID, cartIDs, freight, goodsAmount, deliveryMethodID, address, s.cfg.MiniApp.RawOpenID)
	if progress != nil {
		progress("submit_order_progress", "running", "submitting supplier cart-order", map[string]any{
			"supplierCode": supplierSummary.SupplierCode,
			"cartIdCount":  len(cartIDs),
		})
	}
	submitResponse, err := s.miniapp.ExecuteOrderOperation(ctx, "submit", submitBody)
	if err != nil {
		return supplier.PurchaseOrderResult{}, fmt.Errorf("submit supplier cart-order: %w", err)
	}

	code, message, data := rawResponseEnvelope(submitResponse.Response)
	if !strings.EqualFold(code, "SYS_0000") {
		if message == "" {
			message = "supplier cart-order submit failed"
		}
		return supplier.PurchaseOrderResult{}, fmt.Errorf("%s", message)
	}

	externalRef := firstNonEmptyString(
		stringFromAny(data["billId"]),
		stringFromAny(data["id"]),
		summary.ExternalRef,
	)
	return supplier.PurchaseOrderResult{
		SupplierCode: supplierSummary.SupplierCode,
		ExternalRef:  externalRef,
		Mode:         "miniapp_cart_order",
		Accepted:     true,
		Message:      firstNonEmptyString(message, "submitted via supplier cart-order"),
	}, nil
}

func (s *miniappCartOrderSubmitter) loadSupplierLines(ctx context.Context, app core.App, supplierSummary ProcurementSupplierSummary) ([]miniappCartOrderLine, error) {
	lines := make([]miniappCartOrderLine, 0, len(supplierSummary.Items))
	for _, item := range supplierSummary.Items {
		record, err := app.FindFirstRecordByFilter(
			CollectionSupplierProducts,
			"supplier_code = {:supplier} && original_sku = {:sku}",
			dbx.Params{
				"supplier": item.SupplierCode,
				"sku":      item.OriginalSKU,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load supplier product %s/%s: %w", item.SupplierCode, item.OriginalSKU, err)
		}

		sourceProductID := firstNonEmptyString(
			record.GetString("source_product_id"),
			readJSONAttribute(record, "source_product_id"),
		)
		spuID, skuID := splitSourceProductID(sourceProductID)
		if skuID == "" {
			skuID = strings.TrimSpace(item.OriginalSKU)
		}
		orderUnit := supplierRecordOrderUnit(record, item.SalesUnit)
		if spuID == "" || skuID == "" {
			return nil, fmt.Errorf("supplier item %s/%s missing spuId/skuId for salesUnit=%s", item.SupplierCode, item.OriginalSKU, item.SalesUnit)
		}
		resolvedProduct, resolvedErr := s.miniapp.ResolveProduct(ctx, spuID, skuID)
		resolvedOrderUnit := resolvedMiniappOrderUnit(resolvedProduct, item.SalesUnit)
		unitOption := supplierRecordUnitOption(record, item.SalesUnit)
		unitID := firstNonEmptyString(
			resolvedOrderUnit.UnitID,
			orderUnit.UnitID,
			record.GetString("default_unit_id"),
		)
		if unitID == "" && resolvedProduct != nil {
			unitID = firstNonEmptyString(resolvedProduct.Context.DefaultUnitID, resolvedProduct.Detail.DefaultUnitID)
		}
		if unitID == "" && resolvedErr != nil {
			return nil, fmt.Errorf("resolve supplier product %s/%s: %w", item.SupplierCode, item.OriginalSKU, resolvedErr)
		}
		unitPrice := positiveOr(unitOption.Price, positiveOr(item.BusinessPrice, item.CostPrice))
		if resolvedPrice := resolvedMiniappUnitPrice(resolvedProduct, resolvedOrderUnit, item.SalesUnit); resolvedPrice > 0 {
			unitPrice = resolvedPrice
		}
		if unitPrice <= 0 {
			unitPrice = item.ConsumerPrice
		}
		unitRate := positiveOr(resolvedOrderUnit.Rate, positiveOr(orderUnit.Rate, positiveOr(unitOption.Rate, 1)))
		salesUnit := firstNonEmptyString(
			item.SalesUnit,
			resolvedOrderUnit.UnitName,
			orderUnit.UnitName,
			unitOption.UnitName,
		)
		if salesUnit == "" && resolvedProduct != nil {
			salesUnit = firstNonEmptyString(resolvedProduct.Detail.DefaultUnit, resolvedProduct.Summary.DefaultUnit)
		}
		freightQty := roundMiniappNumber(item.Quantity * positiveOr(unitRate, 1))
		if freightQty <= 0 {
			freightQty = item.Quantity
		}

		lines = append(lines, miniappCartOrderLine{
			SupplierCode: item.SupplierCode,
			OriginalSKU:  item.OriginalSKU,
			SalesUnit:    salesUnit,
			SpuID:        spuID,
			SkuID:        skuID,
			SkuName:      resolvedMiniappSkuName(resolvedProduct),
			UnitID:       unitID,
			UnitRate:     positiveOr(unitRate, 1),
			FreightQty:   freightQty,
			Quantity:     item.Quantity,
			UnitPrice:    unitPrice,
		})
	}
	return lines, nil
}

func (s *miniappCartOrderSubmitter) fetchCartList(ctx context.Context) (map[string]rawCartLine, error) {
	operation, err := s.miniapp.ExecuteCartOperation(ctx, "list", map[string]any{
		"isBuyLogo":      true,
		"levelId":        nil,
		"isIntegralList": false,
		"page": map[string]any{
			"pageNum":  1,
			"pageSize": 200,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("load supplier cart list: %w", err)
	}
	lines := make(map[string]rawCartLine)
	response, _ := operation.Response.(map[string]any)
	data, _ := response["data"].(map[string]any)
	cartSpuVOList, _ := data["cartSpuVOList"].(map[string]any)
	items, _ := cartSpuVOList["list"].([]any)
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		line := rawCartLine{
			ID:     stringFromAny(item["id"]),
			SkuID:  stringFromAny(item["skuId"]),
			UnitID: stringFromAny(item["unitId"]),
			Num:    floatFromAny(item["num"]),
			Cost:   positiveOr(floatFromAny(item["cost"]), floatFromAny(item["discountCost"])),
		}
		if line.SkuID == "" || line.UnitID == "" || line.ID == "" {
			continue
		}
		lines[miniappCartLineKey(line.SkuID, line.UnitID)] = line
	}
	return lines, nil
}

func (s *miniappCartOrderSubmitter) normalizeCartQuantities(ctx context.Context, targetLines []miniappCartOrderLine, cartLines map[string]rawCartLine) error {
	for _, line := range targetLines {
		current, ok := cartLines[miniappCartLineKey(line.SkuID, line.UnitID)]
		if !ok && strings.TrimSpace(line.UnitID) == "" {
			current, ok = findCartLineBySKU(cartLines, line.SkuID)
		}
		if !ok || strings.TrimSpace(current.ID) == "" {
			return fmt.Errorf("supplier cart line not found for skuId=%s unitId=%s", line.SkuID, line.UnitID)
		}
		diff := roundMiniappNumber(line.Quantity - current.Num)
		if almostZero(diff) {
			continue
		}
		discountCost := current.Cost
		if discountCost <= 0 {
			discountCost = line.UnitPrice
		}
		if _, err := s.miniapp.ExecuteCartOperation(ctx, "change-num", map[string]any{
			"id":             current.ID,
			"changeNum":      numericValue(diff),
			"isIntegralList": false,
			"discountCost":   numericValue(discountCost),
		}); err != nil {
			return fmt.Errorf("adjust supplier cart quantity for skuId=%s unitId=%s: %w", line.SkuID, line.UnitID, err)
		}
	}
	return nil
}

func (s *miniappCartOrderSubmitter) fetchCartDetail(ctx context.Context) ([]rawDetailLine, error) {
	operation, err := s.miniapp.ExecuteCartOperation(ctx, "detail", nil)
	if err != nil {
		return nil, fmt.Errorf("load supplier cart detail: %w", err)
	}
	response, _ := operation.Response.(map[string]any)
	data, _ := response["data"].(map[string]any)
	spuDetail, _ := data["spuDetail"].(map[string]any)
	items, _ := spuDetail["spuList"].([]any)
	lines := make([]rawDetailLine, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		lines = append(lines, rawDetailLine{
			ID:       stringFromAny(item["id"]),
			SkuID:    stringFromAny(item["skuId"]),
			UnitName: firstNonEmptyString(stringFromAny(item["unitName"]), stringFromAny(item["salesUnit"])),
			Num:      floatFromAny(item["num"]),
			TotPrice: positiveOr(floatFromAny(item["totPrice"]), floatFromAny(item["discountAfterTotal"])),
		})
	}
	return lines, nil
}

func (s *miniappCartOrderSubmitter) fetchDeliveryAddress(ctx context.Context, customerID string) (map[string]any, error) {
	operation, err := s.miniapp.ExecuteOrderOperation(ctx, "default-delivery", map[string]any{
		"businessId": customerID,
	})
	if err == nil {
		if _, _, data := rawResponseEnvelope(operation.Response); len(data) > 0 {
			return normalizeMiniappDeliveryAddress(data, customerID), nil
		}
	}

	operation, err = s.miniapp.ExecuteOrderOperation(ctx, "deliveries", map[string]any{
		"businessId": customerID,
		"showNum":    false,
	})
	if err != nil {
		return nil, fmt.Errorf("load supplier delivery addresses: %w", err)
	}
	response, _ := operation.Response.(map[string]any)
	data, _ := response["data"].([]any)
	for _, raw := range data {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if boolFromAny(item["isDefault"]) {
			return normalizeMiniappDeliveryAddress(item, customerID), nil
		}
	}
	for _, raw := range data {
		item, ok := raw.(map[string]any)
		if ok {
			return normalizeMiniappDeliveryAddress(item, customerID), nil
		}
	}
	return nil, fmt.Errorf("supplier default delivery address is empty")
}

func (s *miniappCartOrderSubmitter) fetchFreightCost(ctx context.Context, customerID string, deliveryMethodID string, address map[string]any, items []map[string]any) (float64, error) {
	operation, err := s.miniapp.ExecuteFreightCost(ctx, "selected_delivery", map[string]any{
		"deliveryMethodId": deliveryMethodID,
		"customerId":       customerID,
		"province":         address["province"],
		"city":             address["city"],
		"district":         address["district"],
		"qty":              formatMiniappDecimal(totalMiniappQty(items, "qty")),
		"weight":           "0",
		"giftWeight":       0,
		"giftQty":          0,
		"price":            formatMiniappDecimal(totalMiniappQty(items, "price")),
		"skuList":          items,
	})
	if err != nil {
		return 0, fmt.Errorf("load supplier freight cost: %w", err)
	}
	response, _ := operation.Response.(map[string]any)
	return floatFromAny(response["data"]), nil
}

func buildMiniappAddCartBody(lines []miniappCartOrderLine) []map[string]any {
	body := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		body = append(body, map[string]any{
			"cost":         numericValue(line.UnitPrice),
			"discountCost": numericValue(line.UnitPrice),
			"num":          formatMiniappDecimal(line.Quantity),
			"isDeputy":     0,
			"remark":       "",
			"skuName":      line.SkuName,
			"skuId":        line.SkuID,
			"spuId":        line.SpuID,
			"unitId":       line.UnitID,
			"priceSource":  5,
			"warehouseId":  0,
			"isFromBuyNow": 0,
		})
	}
	return body
}

func buildMiniappSubmitBody(summary ProcurementSummary, customerID string, cartIDs []string, freight float64, goodsAmount float64, deliveryMethodID string, address map[string]any, openID string) map[string]any {
	return map[string]any{
		"deliveryTime":          0,
		"customerId":            customerID,
		"cartIdList":            cartIDs,
		"freight":               numericValue(freight),
		"preferentialMoneyList": []any{},
		"remark":                strings.TrimSpace(summary.Notes),
		"receiveAddressInfo":    address,
		"deliveryMethodId":      deliveryMethodID,
		"exemptionFromPostage":  0,
		"invoiceType":           0,
		"billAnnexList":         []any{},
		"giftCheckMap":          map[string]any{},
		"giftCheckMapRemark":    map[string]any{},
		"weightTotal":           "0",
		"giftWeightTotal":       "0",
		"dueMoney":              formatMiniappDecimal(goodsAmount + freight),
		"deductIntegral":        0,
		"deductMoney":           0,
		"deductGuids":           []any{},
		"useDeductIntegral":     0,
		"billClassification":    0,
		"giftCheckMapQty":       map[string]any{},
		"isFromBuyNow":          0,
		"specialAttr":           0,
		"wappEnv": map[string]any{
			"openId":  strings.TrimSpace(openID),
			"netWork": "wifi",
			"device": map[string]any{
				"brand":    "mrtang",
				"model":    "mrtang",
				"system":   "PIM",
				"platform": "server",
			},
		},
	}
}

func matchCartDetailLines(targetLines []miniappCartOrderLine, detailLines []rawDetailLine) ([]string, []map[string]any, float64, error) {
	targets := make(map[string]miniappCartOrderLine, len(targetLines))
	for _, line := range targetLines {
		targets[miniappCartLineKey(line.SkuID, line.UnitID)] = line
	}
	cartIDs := make([]string, 0, len(targetLines))
	freightItems := make([]map[string]any, 0, len(targetLines))
	totalAmount := 0.0
	for _, detail := range detailLines {
		for _, line := range targetLines {
			if strings.TrimSpace(detail.SkuID) != strings.TrimSpace(line.SkuID) {
				continue
			}
			if strings.TrimSpace(detail.UnitName) != "" && strings.TrimSpace(line.SalesUnit) != "" &&
				!strings.EqualFold(strings.TrimSpace(detail.UnitName), strings.TrimSpace(line.SalesUnit)) {
				continue
			}
			cartIDs = append(cartIDs, detail.ID)
			price := positiveOr(detail.TotPrice, line.UnitPrice*line.Quantity)
			totalAmount += price
			freightItems = append(freightItems, map[string]any{
				"skuId":      line.SkuID,
				"qty":        formatMiniappDecimal(positiveOr(line.FreightQty, line.Quantity)),
				"weight":     "0",
				"giftQty":    0,
				"giftWeight": "0",
				"price":      formatMiniappDecimal(price),
			})
			delete(targets, miniappCartLineKey(line.SkuID, line.UnitID))
			break
		}
	}
	if len(targets) > 0 {
		missing := make([]string, 0, len(targets))
		for _, line := range targets {
			missing = append(missing, fmt.Sprintf("%s/%s", line.SkuID, line.SalesUnit))
		}
		return nil, nil, 0, fmt.Errorf("supplier cart detail missing requested lines: %s", strings.Join(missing, ", "))
	}
	return cartIDs, freightItems, totalAmount, nil
}

func aggregateMiniappCartOrderLines(lines []miniappCartOrderLine) []miniappCartOrderLine {
	combined := make(map[string]miniappCartOrderLine, len(lines))
	order := make([]string, 0, len(lines))
	for _, line := range lines {
		key := miniappCartLineKey(line.SkuID, line.UnitID)
		existing, ok := combined[key]
		if !ok {
			combined[key] = line
			order = append(order, key)
			continue
		}
		existing.Quantity = roundMiniappNumber(existing.Quantity + line.Quantity)
		existing.FreightQty = roundMiniappNumber(existing.FreightQty + line.FreightQty)
		combined[key] = existing
	}
	result := make([]miniappCartOrderLine, 0, len(order))
	for _, key := range order {
		result = append(result, combined[key])
	}
	return result
}

func resolvedMiniappOrderUnit(product *miniappmodel.ProductPage, salesUnit string) miniappmodel.ProductOrderUnit {
	if product == nil {
		return miniappmodel.ProductOrderUnit{}
	}
	salesUnit = strings.TrimSpace(salesUnit)
	if salesUnit != "" {
		for _, option := range product.Context.UnitOptions {
			if strings.EqualFold(strings.TrimSpace(option.UnitName), salesUnit) {
				return option
			}
		}
	}
	for _, option := range product.Context.UnitOptions {
		if option.IsDefault {
			return option
		}
	}
	if len(product.Context.UnitOptions) > 0 {
		return product.Context.UnitOptions[0]
	}
	return miniappmodel.ProductOrderUnit{}
}

func resolvedMiniappUnitPrice(product *miniappmodel.ProductPage, orderUnit miniappmodel.ProductOrderUnit, salesUnit string) float64 {
	if product == nil {
		return 0
	}
	salesUnit = strings.TrimSpace(salesUnit)
	for _, option := range product.Pricing.UnitOptions {
		if salesUnit != "" && strings.EqualFold(strings.TrimSpace(option.UnitName), salesUnit) {
			return option.Price
		}
	}
	if strings.TrimSpace(orderUnit.UnitID) != "" && strings.EqualFold(strings.TrimSpace(orderUnit.UnitID), strings.TrimSpace(product.Detail.DefaultUnitID)) {
		return product.Pricing.DefaultPrice
	}
	if salesUnit != "" && strings.EqualFold(salesUnit, strings.TrimSpace(product.Detail.DefaultUnit)) {
		return product.Pricing.DefaultPrice
	}
	if len(product.Pricing.UnitOptions) > 0 && product.Pricing.UnitOptions[0].Price > 0 {
		return product.Pricing.UnitOptions[0].Price
	}
	return 0
}

func resolvedMiniappSkuName(product *miniappmodel.ProductPage) string {
	if product == nil {
		return ""
	}
	return firstNonEmptyString(product.Detail.SkuName, product.Summary.SkuName)
}

func newMiniappActionSource(cfg config.Config) miniappapi.Source {
	snapshot := miniappapi.NewSnapshotSource(
		cfg.MiniApp.HomepageSnapshotFile,
		cfg.MiniApp.CategorySnapshotFile,
		cfg.MiniApp.ProductSnapshotFile,
		cfg.MiniApp.CartOrderSnapshotFile,
	)
	var base miniappapi.Source = snapshot
	if strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "raw") {
		base = miniappapi.NewRawSource(miniappapi.RawSourceConfig{
			BaseURL:             cfg.MiniApp.SourceURL,
			AuthorizedAccountID: cfg.MiniApp.AuthorizedAccountID,
			UserAgent:           cfg.MiniApp.UserAgent,
			TemplateID:          cfg.MiniApp.RawTemplateID,
			Referer:             cfg.MiniApp.RawReferer,
			OpenID:              cfg.MiniApp.RawOpenID,
			ContactsID:          cfg.MiniApp.RawContactsID,
			CustomerID:          cfg.MiniApp.RawCustomerID,
			IsDistributor:       cfg.MiniApp.RawIsDistributor,
			Timeout:             cfg.MiniApp.SourceTimeout,
			Concurrency:         cfg.MiniApp.RawConcurrency,
			MinInterval:         cfg.MiniApp.RawMinInterval,
			RetryMax:            cfg.MiniApp.RawRetryMax,
			WarmupMinInterval:   cfg.MiniApp.RawWarmupMinInterval,
			WarmupMaxInterval:   cfg.MiniApp.RawWarmupMaxInterval,
		}, snapshot)
	}
	return miniappapi.NewOverlaySource(base)
}

func normalizeMiniappDeliveryAddress(address map[string]any, customerID string) map[string]any {
	normalized := cloneMiniappMap(address)
	normalized["customerId"] = firstNonEmptyString(stringFromAny(normalized["customerId"]), customerID)
	normalized["businessId"] = firstNonEmptyString(stringFromAny(normalized["businessId"]), stringFromAny(normalized["addressId"]))
	normalized["addressId"] = firstNonEmptyString(stringFromAny(normalized["addressId"]), stringFromAny(normalized["businessId"]))
	normalized["deliveryMethodId"] = firstNonEmptyString(stringFromAny(normalized["deliveryMethodId"]), stringFromAny(normalized["deliveryId"]))
	normalized["deliveryMethodName"] = firstNonEmptyString(stringFromAny(normalized["deliveryMethodName"]), stringFromAny(normalized["deliveryName"]))
	return normalized
}

func miniappCartLineKey(skuID string, unitID string) string {
	return strings.TrimSpace(skuID) + "::" + strings.TrimSpace(unitID)
}

func findCartLineBySKU(lines map[string]rawCartLine, skuID string) (rawCartLine, bool) {
	skuID = strings.TrimSpace(skuID)
	if skuID == "" {
		return rawCartLine{}, false
	}
	var matched rawCartLine
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line.SkuID) != skuID {
			continue
		}
		if found {
			return rawCartLine{}, false
		}
		matched = line
		found = true
	}
	return matched, found
}

func splitSourceProductID(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	chunks := strings.Split(value, "_")
	if len(chunks) < 2 {
		return "", value
	}
	return strings.TrimSpace(chunks[0]), strings.TrimSpace(chunks[len(chunks)-1])
}

func rawResponseEnvelope(response any) (string, string, map[string]any) {
	mapped, _ := response.(map[string]any)
	code := stringFromAny(mapped["code"])
	message := stringFromAny(mapped["message"])
	data, _ := mapped["data"].(map[string]any)
	return code, message, data
}

func totalMiniappQty(items []map[string]any, key string) float64 {
	total := 0.0
	for _, item := range items {
		total += floatFromAny(item[key])
	}
	return roundMiniappNumber(total)
}

func formatMiniappDecimal(value float64) string {
	value = roundMiniappNumber(value)
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return fmt.Sprintf("%.0f", value)
	}
	text := fmt.Sprintf("%.6f", value)
	text = strings.TrimRight(text, "0")
	return strings.TrimRight(text, ".")
}

func roundMiniappNumber(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}

func almostZero(value float64) bool {
	return math.Abs(value) < 0.000001
}

func numericValue(value float64) any {
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return int(math.Round(value))
	}
	return roundMiniappNumber(value)
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		number, _ := typed.Float64()
		return number
	case string:
		number := 0.0
		fmt.Sscanf(strings.TrimSpace(typed), "%f", &number)
		return number
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		raw := strings.ToLower(strings.TrimSpace(typed))
		return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
	default:
		return false
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func cloneMiniappMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

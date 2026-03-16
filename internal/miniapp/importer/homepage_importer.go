package importer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mrtang-pim/internal/miniapp/model"
)

type HomepageImporter struct{}

func NewHomepageImporter() *HomepageImporter {
	return &HomepageImporter{}
}

func (i *HomepageImporter) Homepage(dataset *model.Dataset) model.HomepageAggregate {
	if dataset == nil {
		return model.HomepageAggregate{}
	}

	homepage := dataset.Homepage
	homepage.Sections = copyHomepageSections(homepage.Sections)
	return homepage
}

func (i *HomepageImporter) CategoryPage(dataset *model.Dataset) model.CategoryPageAggregate {
	if dataset == nil {
		return model.CategoryPageAggregate{}
	}

	categoryPage := dataset.CategoryPage
	categoryPage.Sections = copyCategorySections(categoryPage.Sections)
	return categoryPage
}

func (i *HomepageImporter) ProductPage(dataset *model.Dataset) model.ProductPageAggregate {
	if dataset == nil {
		return model.ProductPageAggregate{}
	}

	productPage := dataset.ProductPage
	productPage.Products = copyProductPages(productPage.Products)
	return productPage
}

func (i *HomepageImporter) CartOrder(dataset *model.Dataset) model.CartOrderAggregate {
	if dataset == nil {
		return model.CartOrderAggregate{}
	}

	return deepCopy(dataset.CartOrder)
}

func (i *HomepageImporter) Cart(dataset *model.Dataset) model.CartAggregate {
	if dataset == nil {
		return model.CartAggregate{}
	}

	return deepCopy(dataset.CartOrder.Cart)
}

func (i *HomepageImporter) Order(dataset *model.Dataset) model.OrderAggregate {
	if dataset == nil {
		return model.OrderAggregate{}
	}

	return deepCopy(dataset.CartOrder.Order)
}

func (i *HomepageImporter) Section(dataset *model.Dataset, id string) *model.HomepageSection {
	if dataset == nil {
		return nil
	}

	for _, section := range dataset.Homepage.Sections {
		if strings.EqualFold(section.ID, id) {
			copySection := section
			copySection.Products = copyHomepageProducts(copySection.Products)
			return &copySection
		}
	}

	return nil
}

func (i *HomepageImporter) CategorySection(dataset *model.Dataset, id string) *model.CategorySection {
	if dataset == nil {
		return nil
	}

	for _, section := range dataset.CategoryPage.Sections {
		if strings.EqualFold(section.ID, id) {
			copySection := section
			copySection.Products = copyHomepageProducts(copySection.Products)
			return &copySection
		}
	}

	return nil
}

func (i *HomepageImporter) Product(dataset *model.Dataset, id string) *model.ProductPage {
	if dataset == nil {
		return nil
	}

	for _, product := range dataset.ProductPage.Products {
		if strings.EqualFold(product.ID, id) {
			copyProduct := product
			copyProduct.ID = normalizedProductID(copyProduct.ID, copyProduct.SpuID, copyProduct.SkuID)
			copyProduct.Summary = normalizeHomepageProduct(copyProduct.Summary)
			return &copyProduct
		}
	}

	return nil
}

func (i *HomepageImporter) CartOperation(dataset *model.Dataset, id string) *model.OperationSnapshot {
	if dataset == nil {
		return nil
	}

	var operation model.OperationSnapshot
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "add", "add-cart", "add_cart":
		operation = dataset.CartOrder.Cart.Add
	case "change-num", "change_num", "change":
		operation = dataset.CartOrder.Cart.ChangeNum
	case "list":
		operation = dataset.CartOrder.Cart.List
	case "detail":
		operation = dataset.CartOrder.Cart.Detail
	case "settle":
		operation = dataset.CartOrder.Cart.Settle
	default:
		return nil
	}

	copied := deepCopy(operation)
	return &copied
}

func (i *HomepageImporter) OrderOperation(dataset *model.Dataset, id string) *model.OperationSnapshot {
	if dataset == nil {
		return nil
	}

	var operation model.OperationSnapshot
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "default-delivery", "default_delivery", "default":
		operation = dataset.CartOrder.Order.DefaultDelivery
	case "deliveries", "delivery-list", "delivery_list":
		operation = dataset.CartOrder.Order.Deliveries
	case "analyse-address", "analyse_address", "address-analyse", "address_analyse":
		operation = dataset.CartOrder.Order.AnalyseAddress
	case "add-delivery", "add_delivery", "address-add", "address_add":
		operation = dataset.CartOrder.Order.AddDelivery
	case "submit", "save":
		operation = dataset.CartOrder.Order.Submit
	default:
		return nil
	}

	copied := deepCopy(operation)
	return &copied
}

func (i *HomepageImporter) FreightCost(dataset *model.Dataset, scenario string) *model.ScenarioAction {
	if dataset == nil {
		return nil
	}

	target := strings.ToLower(strings.TrimSpace(scenario))
	if target == "" {
		target = "preview"
	}

	for _, item := range dataset.CartOrder.Order.FreightCosts {
		if strings.EqualFold(item.Scenario, target) {
			copied := deepCopy(item)
			return &copied
		}
	}

	return nil
}

func (i *HomepageImporter) CartDetailSummary(dataset *model.Dataset) model.CartDetailSummary {
	if dataset == nil {
		return model.CartDetailSummary{}
	}

	response, ok := dataset.CartOrder.Cart.Detail.Response.(map[string]any)
	if !ok {
		return model.CartDetailSummary{}
	}

	data, ok := nestedMap(response, "data")
	if !ok {
		return model.CartDetailSummary{}
	}

	spuDetail, _ := nestedMap(data, "spuDetail")
	itemsRaw, _ := nestedSlice(spuDetail, "spuList")
	couponList, _ := nestedSlice(data, "couponList")

	summary := model.CartDetailSummary{
		VarietyNum:       intValue(spuDetail["varietyNum"]),
		ItemCount:        len(itemsRaw),
		TotalQty:         floatValue(spuDetail["num"]),
		BaseUnitTotalQty: floatValue(data["baseUnitTotNum"]),
		TotalAmount:      floatValue(data["totMoney"]),
		TaxRate:          floatValue(data["taxRate"]),
		ExemptionFreight: floatValue(data["exemptionFromPostage"]),
		CouponCount:      len(couponList),
	}

	items := make([]model.CartDetailItemSummary, 0, len(itemsRaw))
	cartIDs := make([]string, 0, len(itemsRaw))
	for _, raw := range itemsRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		cartID := stringValue(item["id"])
		spuID := stringValue(item["spuId"])
		skuID := stringValue(item["skuId"])
		if cartID != "" {
			cartIDs = append(cartIDs, cartID)
		}

		items = append(items, model.CartDetailItemSummary{
			CartID:         cartID,
			ProductID:      normalizedProductID("", spuID, skuID),
			SpuID:          spuID,
			SkuID:          skuID,
			Name:           firstNonEmptyString(item["name"], item["spuName"]),
			SkuName:        stringValue(item["skuName"]),
			UnitName:       stringValue(item["unitName"]),
			Qty:            floatValue(item["num"]),
			UnitPrice:      floatValue(item["discountCost"]),
			LineAmount:     floatValue(item["totPrice"]),
			DefaultUnitID:  stringValue(item["defaultUnitId"]),
			BaseUnitID:     stringValue(item["baseUnitId"]),
			PromotionCount: len(sliceValue(item["promotionList"])),
		})
	}

	summary.CartIDs = cartIDs
	summary.Items = items
	return summary
}

func (i *HomepageImporter) CartListSummary(dataset *model.Dataset) model.CartListSummary {
	if dataset == nil {
		return model.CartListSummary{}
	}

	response, ok := dataset.CartOrder.Cart.List.Response.(map[string]any)
	if !ok {
		return model.CartListSummary{}
	}

	data, ok := nestedMap(response, "data")
	if !ok {
		return model.CartListSummary{}
	}

	cartSpuVOList, _ := nestedMap(data, "cartSpuVOList")
	itemsRaw, _ := nestedSlice(cartSpuVOList, "list")
	cartTotVO, _ := nestedMap(data, "cartTotVO")

	summary := model.CartListSummary{
		VarietyNum:  intValue(cartTotVO["varietyNum"]),
		ItemCount:   len(itemsRaw),
		TotalQty:    floatValue(cartTotVO["totNum"]),
		TotalAmount: floatValue(cartTotVO["totPrice"]),
		TaxAmount:   floatValue(cartTotVO["totTaxPrice"]),
	}

	items := make([]model.CartListItemSummary, 0, len(itemsRaw))
	for _, raw := range itemsRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		stockTexts := make([]string, 0, len(sliceValue(item["stockList"])))
		for _, stockRaw := range sliceValue(item["stockList"]) {
			stock, ok := stockRaw.(map[string]any)
			if !ok {
				continue
			}
			text := stringValue(stock["displayText"])
			if text != "" {
				stockTexts = append(stockTexts, text)
			}
		}

		promotionTexts := stringSlice(item["promotionTexts"])
		if len(promotionTexts) == 0 {
			for _, promotionRaw := range sliceValue(item["promotionList"]) {
				promotion, ok := promotionRaw.(map[string]any)
				if !ok {
					continue
				}
				name := stringValue(promotion["name"])
				if name != "" {
					promotionTexts = append(promotionTexts, name)
				}
			}
		}

		spuID := stringValue(item["spuId"])
		skuID := stringValue(item["skuId"])
		items = append(items, model.CartListItemSummary{
			CartID:         stringValue(item["id"]),
			ProductID:      normalizedProductID("", spuID, skuID),
			SpuID:          spuID,
			SkuID:          skuID,
			Name:           firstNonEmptyString(item["name"], item["spuName"]),
			SkuName:        stringValue(item["skuName"]),
			UnitName:       stringValue(item["unitName"]),
			Qty:            floatValue(item["num"]),
			UnitPrice:      floatValue(item["cost"]),
			LineAmount:     floatValue(item["totPrice"]),
			BaseUnitName:   stringValue(item["baseUnitName"]),
			UnitRate:       floatValue(item["unitRate"]),
			HasMultiUnit:   boolValue(item["whetherMultipleUnits"]),
			StockTexts:     stockTexts,
			PromotionTexts: promotionTexts,
		})
	}

	summary.Items = items
	return summary
}

func (i *HomepageImporter) OrderSubmitSummary(dataset *model.Dataset) model.OrderSubmitSummary {
	if dataset == nil {
		return model.OrderSubmitSummary{}
	}

	request, _ := dataset.CartOrder.Order.Submit.RequestBody.(map[string]any)
	response, _ := dataset.CartOrder.Order.Submit.Response.(map[string]any)
	responseData, _ := nestedMap(response, "data")
	receiveAddress, _ := nestedMap(request, "receiveAddressInfo")

	cartIDsRaw, _ := request["cartIdList"].([]any)
	cartIDs := make([]string, 0, len(cartIDsRaw))
	for _, raw := range cartIDsRaw {
		if id := stringValue(raw); id != "" {
			cartIDs = append(cartIDs, id)
		}
	}

	paymentsRaw, _ := responseData["openWxPayList"].([]any)
	payments := make([]model.OrderPaymentOption, 0, len(paymentsRaw))
	for _, raw := range paymentsRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		payments = append(payments, model.OrderPaymentOption{
			Name:         stringValue(item["name"]),
			Type:         intValue(item["type"]),
			PayRecommend: intValue(item["payRecommend"]),
		})
	}

	return model.OrderSubmitSummary{
		Message:          stringValue(response["message"]),
		BillID:           stringValue(responseData["billId"]),
		CustomerID:       stringValue(request["customerId"]),
		CustomerName:     firstNonEmptyString(responseData["customerName"], receiveAddress["customerName"]),
		AddressID:        stringValue(receiveAddress["addressId"]),
		DeliveryMethodID: firstNonEmptyString(stringValue(request["deliveryMethodId"]), stringValue(receiveAddress["deliveryMethodId"])),
		CartIDs:          cartIDs,
		DueAmount:        floatValue(responseData["dueMoney"]),
		FreightAmount:    floatValue(request["freight"]),
		RequiresPayment:  boolValue(responseData["whetherOpenWxPay"]),
		DeadlineTime:     int64Value(responseData["deadlineTime"]),
		BillType:         intValue(responseData["billType"]),
		PaymentOptions:   payments,
		ReceiveAddress: model.OrderReceiveAddressSummary{
			AddressID:    stringValue(receiveAddress["addressId"]),
			CustomerName: stringValue(receiveAddress["customerName"]),
			Phone:        stringValue(receiveAddress["phone"]),
			FullAddress:  stringValue(receiveAddress["fullAddress"]),
			DeliveryID:   stringValue(receiveAddress["deliveryId"]),
			DeliveryName: stringValue(receiveAddress["deliveryName"]),
			Longitude:    floatValue(receiveAddress["longitude"]),
			Latitude:     floatValue(receiveAddress["latitude"]),
		},
	}
}

func (i *HomepageImporter) FreightSummary(dataset *model.Dataset) model.FreightSummary {
	if dataset == nil {
		return model.FreightSummary{}
	}

	summary := model.FreightSummary{
		Scenarios: make([]model.FreightScenarioSummary, 0, len(dataset.CartOrder.Order.FreightCosts)),
	}

	for _, item := range dataset.CartOrder.Order.FreightCosts {
		request, _ := item.RequestBody.(map[string]any)
		skuList := sliceValue(request["skuList"])
		summary.Scenarios = append(summary.Scenarios, model.FreightScenarioSummary{
			Scenario:         item.Scenario,
			Label:            item.Label,
			DeliveryMethodID: stringValue(request["deliveryMethodId"]),
			CustomerID:       stringValue(request["customerId"]),
			Qty:              floatValue(request["qty"]),
			TotalAmount:      floatValue(request["price"]),
			FreightAmount:    responseDataValue(item.Response),
			SkuCount:         len(skuList),
		})
	}

	return summary
}

func (i *HomepageImporter) DefaultDeliverySummary(dataset *model.Dataset) model.DefaultDeliverySummary {
	if dataset == nil {
		return model.DefaultDeliverySummary{}
	}

	if address, ok := extractDeliveryAddress(dataset.CartOrder.Order.DefaultDelivery.Response); ok {
		return model.DefaultDeliverySummary{
			Found:   true,
			Source:  "default_delivery",
			Address: &address,
		}
	}

	if address, ok := extractDeliveryAddress(dataset.CartOrder.Order.AddDelivery.Response); ok {
		return model.DefaultDeliverySummary{
			Found:   true,
			Source:  "add_delivery_fallback",
			Address: &address,
		}
	}

	return model.DefaultDeliverySummary{
		Found:  false,
		Source: "none",
	}
}

func (i *HomepageImporter) DeliveriesSummary(dataset *model.Dataset) model.DeliveriesSummary {
	if dataset == nil {
		return model.DeliveriesSummary{}
	}

	items := make([]model.DeliveryAddressSummary, 0)
	response, _ := dataset.CartOrder.Order.Deliveries.Response.(map[string]any)
	if data, ok := response["data"].([]any); ok {
		for _, raw := range data {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, deliveryAddressSummary(item))
		}
	}

	if len(items) == 0 {
		if address, ok := extractDeliveryAddress(dataset.CartOrder.Order.AddDelivery.Response); ok {
			items = append(items, address)
		}
	}

	summary := model.DeliveriesSummary{
		Count: len(items),
		Items: items,
	}
	for _, item := range items {
		if item.IsDefault {
			summary.DefaultAddressID = item.AddressID
			break
		}
	}
	if summary.DefaultAddressID == "" && len(items) > 0 {
		summary.DefaultAddressID = items[0].AddressID
	}

	return summary
}

func (i *HomepageImporter) CheckoutSummary(dataset *model.Dataset) model.CheckoutSummary {
	return model.CheckoutSummary{
		CartList:        i.CartListSummary(dataset),
		CartDetail:      i.CartDetailSummary(dataset),
		DefaultDelivery: i.DefaultDeliverySummary(dataset),
		Deliveries:      i.DeliveriesSummary(dataset),
		Freight:         i.FreightSummary(dataset),
		Submit:          i.OrderSubmitSummary(dataset),
	}
}

func (i *HomepageImporter) ProductCoverage(dataset *model.Dataset) []model.ProductCoverage {
	if dataset == nil {
		return nil
	}

	products := copyProductPages(dataset.ProductPage.Products)
	coverage := make([]model.ProductCoverage, 0, len(products))
	for _, product := range products {
		entry := model.ProductCoverage{
			ProductID:      product.ID,
			SpuID:          product.SpuID,
			SkuID:          product.SkuID,
			Name:           product.Summary.Name,
			SourceType:     product.SourceType,
			SourceSections: append([]string(nil), product.SourceSections...),
			UnitCount:      len(product.Summary.UnitOptions),
			HasMultiUnit:   len(product.Summary.UnitOptions) > 1,
			Priority:       productPriority(product),
		}
		coverage = append(coverage, entry)
	}

	sort.SliceStable(coverage, func(a int, b int) bool {
		left := priorityRank(coverage[a].Priority)
		right := priorityRank(coverage[b].Priority)
		if left != right {
			return left < right
		}
		return coverage[a].Name < coverage[b].Name
	})

	return coverage
}

func (i *HomepageImporter) ProductCoverageSummary(dataset *model.Dataset) model.ProductCoverageSummary {
	coverage := i.ProductCoverage(dataset)
	summary := model.ProductCoverageSummary{
		TotalProducts: len(coverage),
	}
	if len(coverage) == 0 {
		return summary
	}

	grouped := make(map[string][]model.ProductCoverage)
	priorityOrder := []string{
		"homepage_dual_unit",
		"category_dual_unit",
		"visible_single_unit",
		"done_rr_detail",
	}

	for _, item := range coverage {
		if item.HasMultiUnit {
			summary.MultiUnitTotal++
		}
		grouped[item.Priority] = append(grouped[item.Priority], item)
	}

	buckets := make([]model.ProductCoverageBucket, 0, len(priorityOrder))
	for _, priority := range priorityOrder {
		items := grouped[priority]
		if len(items) == 0 {
			continue
		}
		buckets = append(buckets, model.ProductCoverageBucket{
			Priority: priority,
			Count:    len(items),
			Items:    items,
		})
	}
	summary.ByPriority = buckets
	summary.FirstBatch = append([]model.ProductCoverage(nil), grouped["homepage_dual_unit"]...)

	return summary
}

func (i *HomepageImporter) Contracts(dataset *model.Dataset, localPathPrefix string) []model.Contract {
	if dataset == nil {
		return nil
	}

	prefix := strings.TrimSpace(localPathPrefix)
	if prefix == "" {
		return append([]model.Contract(nil), dataset.Contracts...)
	}

	filtered := make([]model.Contract, 0, len(dataset.Contracts))
	for _, contract := range dataset.Contracts {
		if strings.HasPrefix(contract.LocalPath, prefix) {
			filtered = append(filtered, contract)
		}
	}

	return filtered
}

func copyHomepageSections(sections []model.HomepageSection) []model.HomepageSection {
	if len(sections) == 0 {
		return sections
	}

	copied := make([]model.HomepageSection, len(sections))
	for idx, section := range sections {
		copied[idx] = section
		copied[idx].Products = copyHomepageProducts(section.Products)
	}

	return copied
}

func copyCategorySections(sections []model.CategorySection) []model.CategorySection {
	if len(sections) == 0 {
		return sections
	}

	copied := make([]model.CategorySection, len(sections))
	for idx, section := range sections {
		copied[idx] = section
		copied[idx].Products = copyHomepageProducts(section.Products)
	}

	return copied
}

func copyHomepageProducts(products []model.HomepageProduct) []model.HomepageProduct {
	if len(products) == 0 {
		return products
	}

	copied := make([]model.HomepageProduct, len(products))
	for idx, product := range products {
		copied[idx] = normalizeHomepageProduct(product)
	}

	return copied
}

func copyProductPages(products []model.ProductPage) []model.ProductPage {
	if len(products) == 0 {
		return products
	}

	copied := make([]model.ProductPage, len(products))
	for idx, product := range products {
		copied[idx] = product
		copied[idx].ID = normalizedProductID(product.ID, product.SpuID, product.SkuID)
		copied[idx].Summary = normalizeHomepageProduct(product.Summary)
	}

	return copied
}

func normalizeHomepageProduct(product model.HomepageProduct) model.HomepageProduct {
	product.ProductID = normalizedProductID(product.ProductID, product.SpuID, product.SkuID)
	return product
}

func normalizedProductID(current string, spuID string, skuID string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	if strings.TrimSpace(spuID) == "" || strings.TrimSpace(skuID) == "" {
		return ""
	}
	return spuID + "_" + skuID
}

func productPriority(product model.ProductPage) string {
	if strings.EqualFold(product.SourceType, "rr_detail") {
		return "done_rr_detail"
	}

	hasMultiUnit := len(product.Summary.UnitOptions) > 1
	hasHomepage := containsAny(product.SourceSections, "new", "hot")
	hasCategory := containsAny(product.SourceSections, "chicken")

	switch {
	case hasHomepage && hasMultiUnit:
		return "homepage_dual_unit"
	case hasCategory && hasMultiUnit:
		return "category_dual_unit"
	default:
		return "visible_single_unit"
	}
}

func priorityRank(priority string) int {
	switch priority {
	case "homepage_dual_unit":
		return 0
	case "category_dual_unit":
		return 1
	case "visible_single_unit":
		return 2
	case "done_rr_detail":
		return 3
	default:
		return 4
	}
}

func containsAny(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if strings.EqualFold(value, target) {
				return true
			}
		}
	}
	return false
}

func deepCopy[T any](value T) T {
	body, err := json.Marshal(value)
	if err != nil {
		return value
	}

	var copied T
	if err := json.Unmarshal(body, &copied); err != nil {
		return value
	}

	return copied
}

func nestedMap(input map[string]any, key string) (map[string]any, bool) {
	value, ok := input[key]
	if !ok {
		return nil, false
	}
	result, ok := value.(map[string]any)
	return result, ok
}

func nestedSlice(input map[string]any, key string) ([]any, bool) {
	value, ok := input[key]
	if !ok {
		return nil, false
	}
	result, ok := value.([]any)
	return result, ok
}

func sliceValue(value any) []any {
	result, _ := value.([]any)
	return result
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case json.Number:
		number, _ := typed.Float64()
		return number
	case string:
		number, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number
	default:
		number, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", value)), 64)
		return number
	}
}

func intValue(value any) int {
	return int(floatValue(value))
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case json.Number:
		number, _ := typed.Int64()
		return number
	case string:
		number, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return number
	default:
		return int64(floatValue(value))
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		result, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return result
	default:
		return false
	}
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		text := stringValue(value)
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func stringSlice(value any) []string {
	raw := sliceValue(value)
	if len(raw) == 0 {
		return nil
	}

	result := make([]string, 0, len(raw))
	for _, item := range raw {
		text := stringValue(item)
		if text != "" && text != "<nil>" {
			result = append(result, text)
		}
	}
	return result
}

func responseDataValue(payload any) float64 {
	response, ok := payload.(map[string]any)
	if !ok {
		return 0
	}
	return floatValue(response["data"])
}

func extractDeliveryAddress(payload any) (model.DeliveryAddressSummary, bool) {
	response, ok := payload.(map[string]any)
	if !ok {
		return model.DeliveryAddressSummary{}, false
	}

	data, ok := response["data"].(map[string]any)
	if !ok || len(data) == 0 {
		return model.DeliveryAddressSummary{}, false
	}

	return deliveryAddressSummary(data), true
}

func deliveryAddressSummary(data map[string]any) model.DeliveryAddressSummary {
	return model.DeliveryAddressSummary{
		AddressID:     firstNonEmptyString(data["addressId"], data["businessId"]),
		CustomerID:    stringValue(data["customerId"]),
		CustomerName:  stringValue(data["customerName"]),
		Phone:         stringValue(data["phone"]),
		FullAddress:   stringValue(data["fullAddress"]),
		DetailAddress: stringValue(data["detailAddress"]),
		ProvinceName:  stringValue(data["provinceName"]),
		CityName:      stringValue(data["cityName"]),
		DistrictName:  stringValue(data["districtName"]),
		DeliveryID:    firstNonEmptyString(data["deliveryMethodId"], data["deliveryId"]),
		DeliveryName:  firstNonEmptyString(data["deliveryMethodName"], data["deliveryName"]),
		IsDefault:     boolValue(data["isDefault"]),
		Longitude:     floatValue(data["longitude"]),
		Latitude:      floatValue(data["latitude"]),
	}
}

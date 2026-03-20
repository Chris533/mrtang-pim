package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"mrtang-pim/internal/miniapp/model"
)

const (
	rawCategoryTreePath = "/gateway/goodsservice/api/v1/wx/category/get_have_goods_category"
	rawGoodsListPath    = "/gateway/gateway-mall-service/api/v1/wx/goods/list"
	rawPriceStockPath   = "/gateway/goodsservice/api/v1/wx/goods/sku/list/price_stock"
)

type RawSourceConfig struct {
	BaseURL             string
	AuthorizedAccountID string
	UserAgent           string
	TemplateID          string
	Referer             string
	OpenID              string
	ContactsID          string
	CustomerID          string
	IsDistributor       bool
	Timeout             time.Duration
	Concurrency         int
	MinInterval         time.Duration
	RetryMax            int
	WarmupMinInterval   time.Duration
	WarmupMaxInterval   time.Duration
}

type RawSource struct {
	cfg                  RawSourceConfig
	client               *http.Client
	fallback             Source
	workerLimit          int
	retryMax             int
	requestMu            sync.Mutex
	lastRequestAt        time.Time
	warmupMu             sync.Mutex
	warmupStatus         model.RawAuthStatus
	warmupMinTTL         time.Duration
	warmupMaxTTL         time.Duration
	nextWarmupAt         time.Time
	rng                  *rand.Rand
	categoryTreeMu       sync.RWMutex
	categoryTreeCache    []model.CategoryNode
	categoryTreeCachedAt time.Time
	categorySectionMu    sync.RWMutex
	categorySectionCache map[string]model.CategorySection
}

func NewRawSource(cfg RawSourceConfig, fallback Source) *RawSource {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	workerLimit := cfg.Concurrency
	if workerLimit <= 0 {
		workerLimit = 4
	}
	retryMax := cfg.RetryMax
	if retryMax < 0 {
		retryMax = 0
	}
	warmupMinTTL := cfg.WarmupMinInterval
	warmupMaxTTL := cfg.WarmupMaxInterval
	if warmupMinTTL <= 0 {
		warmupMinTTL = 30 * time.Minute
	}
	if warmupMaxTTL <= 0 {
		warmupMaxTTL = 60 * time.Minute
	}
	if warmupMaxTTL < warmupMinTTL {
		warmupMaxTTL = warmupMinTTL
	}

	return &RawSource{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
		fallback:             fallback,
		workerLimit:          workerLimit,
		retryMax:             retryMax,
		warmupMinTTL:         warmupMinTTL,
		warmupMaxTTL:         warmupMaxTTL,
		rng:                  rand.New(rand.NewSource(time.Now().UnixNano())),
		categorySectionCache: make(map[string]model.CategorySection),
		warmupStatus: model.RawAuthStatus{
			Enabled: strings.TrimSpace(cfg.AuthorizedAccountID) != "",
			Status:  "idle",
			OpenID:  strings.TrimSpace(cfg.OpenID),
		},
	}
}

func (s *RawSource) RawAuthStatus() model.RawAuthStatus {
	s.warmupMu.Lock()
	defer s.warmupMu.Unlock()
	status := s.warmupStatus
	status.Enabled = strings.TrimSpace(s.cfg.AuthorizedAccountID) != ""
	status.OpenID = strings.TrimSpace(s.cfg.OpenID)
	return status
}

func (s *RawSource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	if strings.TrimSpace(s.cfg.BaseURL) == "" {
		return nil, fmt.Errorf("miniapp raw source url is empty")
	}
	if s.fallback == nil {
		return nil, fmt.Errorf("miniapp raw source fallback is nil")
	}
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	base, err := s.fallback.FetchDataset(ctx)
	if err != nil {
		return nil, fmt.Errorf("load raw source fallback dataset: %w", err)
	}

	dataset := *base
	nodes, err := s.fetchCategoryTreeWithFallback(ctx, dataset.Homepage.Settings.GoodsCategorySetting, true)
	if err != nil {
		return nil, err
	}

	sections, products, categoryNotes, err := s.fetchCategoryProducts(ctx, nodes, s.sectionTimeout(), true)
	if err != nil {
		return nil, err
	}

	dataset.Meta.Source = "raw"
	dataset.Meta.Description = "真实源站原始接口标准化数据"
	dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes,
		"raw source 直接请求真实源站分类树、分类商品列表与商品详情链路",
		"当前已覆盖分类树、顶级分类商品列表、价格库存、商品详情、套餐和购物车上下文，checkout 真实链路仍由后续批次接入",
	)
	if strings.TrimSpace(s.cfg.ContactsID) == "" || strings.TrimSpace(s.cfg.CustomerID) == "" {
		dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes, "raw 登录续活当前未显式配置 contactsId/customerId，将使用 fallback 样本并自动尝试 distributor true/false。")
	}
	if strings.TrimSpace(s.cfg.OpenID) == "" {
		dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes, "raw 登录续活当前未配置 openId，预授权状态校验步骤将跳过。")
	}
	dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes, categoryNotes...)
	dataset.CategoryPage.Tree = nodes
	if len(sections) > 0 {
		dataset.CategoryPage.Sections = sections
	}
	if len(products) > 0 {
		dataset.ProductPage.Products = products
	}
	if cartOrder, err := s.fetchRawCartOrder(ctx, dataset.CartOrder); err == nil {
		dataset.CartOrder = cartOrder
		dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes,
			"raw source 已接入购物车列表、购物车详情与结算预览的真实链路",
			"地址管理与提交订单仍保持 fallback，避免后台读取数据时误触发真实地址变更或下单",
		)
	}

	return &dataset, nil
}

func (s *RawSource) FetchTargetSyncDataset(ctx context.Context, entityType string, scopeKey string) (*model.Dataset, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))

	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	if s.fallback == nil {
		return nil, fmt.Errorf("miniapp raw source fallback is nil")
	}

	base, err := s.fallback.FetchDataset(ctx)
	if err != nil {
		return nil, fmt.Errorf("load raw source fallback dataset: %w", err)
	}

	nodes, err := s.fetchCategoryTreeWithFallback(ctx, base.Homepage.Settings.GoodsCategorySetting, true)
	if err != nil {
		return nil, err
	}

	dataset := *base
	dataset.Meta.Source = "raw"
	dataset.CategoryPage.Tree = nodes
	dataset.CategoryPage.Sections = nil
	dataset.ProductPage.Products = nil

	if entityType == "category_tree" {
		return &dataset, nil
	}

	scopedNodes := nodes
	if key := strings.TrimSpace(scopeKey); key != "" {
		node := findCategoryNode(nodes, key)
		if node == nil {
			return nil, fmt.Errorf("category scope not found: %s", key)
		}
		scopedNodes = []model.CategoryNode{*node}
	}

	includePriceStock := entityType != "category_sources"
	sections, products, notes, err := s.fetchCategoryProducts(ctx, scopedNodes, s.targetSyncSectionTimeout(), includePriceStock)
	if err != nil {
		return nil, err
	}
	dataset.CategoryPage.Sections = sections
	if includePriceStock {
		dataset.ProductPage.Products = products
	}
	dataset.Meta.Notes = append(dataset.Meta.Notes, notes...)
	return &dataset, nil
}

func (s *RawSource) FetchTargetSyncProductsFromSections(ctx context.Context, sections []model.CategorySection, scopeKey string) (*model.Dataset, error) {
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}
	if s.fallback == nil {
		return nil, fmt.Errorf("miniapp raw source fallback is nil")
	}

	base, err := s.fallback.FetchDataset(ctx)
	if err != nil {
		return nil, fmt.Errorf("load raw source fallback dataset: %w", err)
	}

	filteredSections := filterCategorySectionsByScope(sections, scopeKey)
	products := buildRawProductsFromSections(filteredSections)
	products = s.enrichRawProducts(ctx, products)

	dataset := *base
	dataset.Meta.Source = "raw_category_sources"
	dataset.Meta.Description = "基于已保存分类商品来源补商品详情"
	dataset.Meta.Notes = appendUniqueStrings(dataset.Meta.Notes,
		"商品规格抓取当前基于已保存分类商品来源补商品详情，不再重新请求分类商品列表。",
	)
	dataset.CategoryPage.Sections = filteredSections
	dataset.ProductPage.Products = products
	return &dataset, nil
}

func (s *RawSource) ResolveProduct(ctx context.Context, spuID string, skuID string) (*model.ProductPage, error) {
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	product := model.ProductPage{
		ID:         strings.TrimSpace(spuID) + "_" + strings.TrimSpace(skuID),
		SpuID:      strings.TrimSpace(spuID),
		SkuID:      strings.TrimSpace(skuID),
		SourceType: "raw_detail",
		Summary: model.HomepageProduct{
			ProductID: strings.TrimSpace(spuID) + "_" + strings.TrimSpace(skuID),
			SpuID:     strings.TrimSpace(spuID),
			SkuID:     strings.TrimSpace(skuID),
		},
	}

	info, err := s.fetchProductInfo(ctx, product.SpuID, product.SkuID)
	if err != nil {
		return nil, fmt.Errorf("fetch raw product info %s/%s: %w", product.SpuID, product.SkuID, err)
	}
	priceList, _ := s.fetchProductPriceList(ctx, product.SpuID, product.SkuID, info.UnitID)
	packageData, _ := s.fetchProductPackage(ctx, product.SpuID, product.SkuID)
	cartSummary, _ := s.fetchCartTotal(ctx)
	chooseData, _ := s.fetchCartChoose(ctx, product.SkuID)
	addCartSummary, _ := s.fetchAddCartTotal(ctx, product.SpuID)

	resolved := mergeRawProductDetail(product, info, priceList, packageData, cartSummary, chooseData, addCartSummary)
	return &resolved, nil
}

func (s *RawSource) fetchRawCartOrder(ctx context.Context, fallback model.CartOrderAggregate) (model.CartOrderAggregate, error) {
	result := fallback

	listBody := map[string]any{
		"isBuyLogo":      true,
		"levelId":        nil,
		"isIntegralList": false,
		"page": map[string]any{
			"pageNum":  1,
			"pageSize": 20,
		},
	}
	listResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/list", nil, listBody)
	if err != nil {
		return fallback, fmt.Errorf("fetch raw cart list: %w", err)
	}
	result.Cart.List = model.OperationSnapshot{
		ContractID:  "raw_cart_list",
		RequestBody: listBody,
		Response:    listResp,
	}

	detailQuery := map[string]any{
		"isFirst":        1,
		"isIntegralList": false,
		"isFromBuyNow":   0,
		"specialAttr":    0,
	}
	detailResp, err := s.fetchRawResponse(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/cart/detail", detailQuery, nil)
	if err != nil {
		return fallback, fmt.Errorf("fetch raw cart detail: %w", err)
	}
	result.Cart.Detail = model.OperationSnapshot{
		ContractID:   "raw_cart_detail",
		RequestQuery: detailQuery,
		Response:     detailResp,
	}

	settleBody := map[string]any{
		"isIntegralList": false,
		"isFromBuyNow":   0,
	}
	settleResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/settle", nil, settleBody)
	if err != nil {
		return fallback, fmt.Errorf("fetch raw cart settle: %w", err)
	}
	result.Cart.Settle = model.OperationSnapshot{
		ContractID:  "raw_cart_settle",
		RequestBody: settleBody,
		Response:    settleResp,
	}

	customerID := rawCustomerIDFromLoginStatus(loginStatusData(s.fetchLoginStatus(ctx, result)))
	if customerID == "" {
		customerID = rawCustomerIDFromFallback(result)
	}

	if customerID != "" {
		defaultDeliveryBody := map[string]any{"businessId": customerID}
		if defaultDeliveryResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/get_default_delivery", nil, defaultDeliveryBody); err == nil {
			result.Order.DefaultDelivery = model.OperationSnapshot{
				ContractID:  "raw_order_default_delivery",
				RequestBody: defaultDeliveryBody,
				Response:    defaultDeliveryResp,
			}
		}

		deliveriesBody := map[string]any{"businessId": customerID, "showNum": false}
		if deliveriesResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/get_deliverys", nil, deliveriesBody); err == nil {
			result.Order.Deliveries = model.OperationSnapshot{
				ContractID:  "raw_order_deliveries",
				RequestBody: deliveriesBody,
				Response:    deliveriesResp,
			}
		}
	}

	analyseBody := rawAnalyseAddressBody(result)
	if len(analyseBody) > 0 {
		if analyseResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/saas-platform-service/api/v1/address/analyse_address", nil, analyseBody); err == nil {
			result.Order.AnalyseAddress = model.OperationSnapshot{
				ContractID:  "raw_order_analyse_address",
				RequestBody: analyseBody,
				Response:    analyseResp,
			}
		}
	}

	if freightPreviewBody := rawFreightPreviewBody(result, customerID); len(freightPreviewBody) > 0 {
		if freightPreviewResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/logisticsservice/api/v1/freight/cost", nil, freightPreviewBody); err == nil {
			upsertRawFreightScenario(&result.Order.FreightCosts, model.ScenarioAction{
				Scenario:    "preview",
				Label:       "未选择配送方式",
				ContractID:  "raw_order_freight_preview",
				RequestBody: freightPreviewBody,
				Response:    freightPreviewResp,
			})
		}
	}

	if freightSelectedBody := rawFreightSelectedBody(result, customerID); len(freightSelectedBody) > 0 {
		if freightSelectedResp, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/logisticsservice/api/v1/freight/cost", nil, freightSelectedBody); err == nil {
			upsertRawFreightScenario(&result.Order.FreightCosts, model.ScenarioAction{
				Scenario:    "selected_delivery",
				Label:       "已选择配送方式",
				ContractID:  "raw_order_freight_selected_delivery",
				RequestBody: freightSelectedBody,
				Response:    freightSelectedResp,
			})
		}
	}

	return result, nil
}

func (s *RawSource) ExecuteCartOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error) {
	fallback, err := s.fallback.FetchDataset(context.WithoutCancel(ctx))
	if err != nil {
		return nil, fmt.Errorf("load fallback cart dataset: %w", err)
	}
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	switch id {
	case "add":
		body := requestBody
		if body == nil {
			return nil, fmt.Errorf("raw add cart requires explicit request body")
		}
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/addCart", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_cart_add", RequestBody: body, Response: response}, nil
	case "change-num":
		body := requestBody
		if body == nil {
			return nil, fmt.Errorf("raw change cart quantity requires explicit request body")
		}
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/change_cart_num", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_cart_change_num", RequestBody: body, Response: response}, nil
	case "list":
		body := firstNonNil(requestBody, fallback.CartOrder.Cart.List.RequestBody)
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/list", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_cart_list", RequestBody: body, Response: response}, nil
	case "detail":
		query := anyMap(fallback.CartOrder.Cart.Detail.RequestQuery)
		response, err := s.fetchRawResponse(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/cart/detail", query, nil)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_cart_detail", RequestQuery: query, Response: response}, nil
	case "settle":
		body := firstNonNil(requestBody, fallback.CartOrder.Cart.Settle.RequestBody)
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/cart/settle", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_cart_settle", RequestBody: body, Response: response}, nil
	default:
		return nil, nil
	}
}

func (s *RawSource) ExecuteOrderOperation(ctx context.Context, id string, requestBody any) (*model.OperationSnapshot, error) {
	fallback, err := s.fallback.FetchDataset(context.WithoutCancel(ctx))
	if err != nil {
		return nil, fmt.Errorf("load fallback order dataset: %w", err)
	}
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	switch id {
	case "default-delivery":
		body := firstNonNil(requestBody, fallback.CartOrder.Order.DefaultDelivery.RequestBody)
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/get_default_delivery", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_order_default_delivery", RequestBody: body, Response: response}, nil
	case "deliveries":
		body := firstNonNil(requestBody, fallback.CartOrder.Order.Deliveries.RequestBody)
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/get_deliverys", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_order_deliveries", RequestBody: body, Response: response}, nil
	case "analyse-address":
		body := firstNonNil(requestBody, fallback.CartOrder.Order.AnalyseAddress.RequestBody)
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/saas-platform-service/api/v1/address/analyse_address", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_order_analyse_address", RequestBody: body, Response: response}, nil
	case "add-delivery":
		body := requestBody
		if body == nil {
			return nil, fmt.Errorf("raw add delivery requires explicit request body")
		}
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/add_delivery", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_order_add_delivery", RequestBody: body, Response: response}, nil
	case "submit":
		body := requestBody
		if body == nil {
			return nil, fmt.Errorf("raw submit order requires explicit request body")
		}
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/billservice/api/v1/wx/sale_bill/save", nil, body)
		if err != nil {
			return nil, err
		}
		return &model.OperationSnapshot{ContractID: "raw_order_submit", RequestBody: body, Response: response}, nil
	default:
		return nil, nil
	}
}

func (s *RawSource) ExecuteFreightScenario(ctx context.Context, scenario string, requestBody any) (*model.ScenarioAction, error) {
	fallback, err := s.fallback.FetchDataset(context.WithoutCancel(ctx))
	if err != nil {
		return nil, fmt.Errorf("load fallback freight dataset: %w", err)
	}
	if err := s.ensureWarmup(ctx, false); err != nil {
		return nil, err
	}

	body := requestBody
	if body == nil {
		switch strings.TrimSpace(scenario) {
		case "", "preview":
			if action := rawFindFreightScenario(fallback.CartOrder.Order.FreightCosts, "preview"); action != nil {
				body = action.RequestBody
				scenario = "preview"
			}
		default:
			if action := rawFindFreightScenario(fallback.CartOrder.Order.FreightCosts, scenario); action != nil {
				body = action.RequestBody
			}
		}
	}
	if body == nil {
		return nil, nil
	}

	response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/logisticsservice/api/v1/freight/cost", nil, body)
	if err != nil {
		return nil, err
	}

	action := &model.ScenarioAction{
		Scenario:    firstNonEmptyRaw(strings.TrimSpace(scenario), "preview"),
		Label:       rawFreightLabel(scenario),
		ContractID:  "raw_order_freight_" + firstNonEmptyRaw(strings.TrimSpace(scenario), "preview"),
		RequestBody: body,
		Response:    response,
	}
	return action, nil
}

func (s *RawSource) fetchLoginStatus(ctx context.Context, fallback model.CartOrderAggregate) (map[string]any, error) {
	fallbackContactsID, fallbackCustomerID := rawLoginFallbackIDs(fallback)
	contactsID := firstNonEmptyRaw(strings.TrimSpace(s.cfg.ContactsID), fallbackContactsID)
	customerID := firstNonEmptyRaw(strings.TrimSpace(s.cfg.CustomerID), fallbackCustomerID)
	distributorCandidates := []bool{s.cfg.IsDistributor}
	if len(distributorCandidates) == 0 || distributorCandidates[0] {
		distributorCandidates = appendUniqueBools(distributorCandidates, false)
	} else {
		distributorCandidates = appendUniqueBools(distributorCandidates, true)
	}

	var lastResponse map[string]any
	for _, isDistributor := range distributorCandidates {
		requestBody := map[string]any{
			"contactsId":    contactsID,
			"customerId":    customerID,
			"isDistributor": isDistributor,
		}
		response, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/app/get_login_status", nil, requestBody)
		if err != nil {
			return nil, err
		}
		lastResponse = response
		if len(loginStatusData(response, nil)) > 0 {
			return response, nil
		}
	}
	return lastResponse, nil
}

func (s *RawSource) ensureWarmup(ctx context.Context, force bool) error {
	if strings.TrimSpace(s.cfg.AuthorizedAccountID) == "" {
		s.setWarmupStatus(func(status *model.RawAuthStatus) {
			status.Enabled = false
			status.Status = "skipped"
			status.Message = "未配置 MINIAPP_AUTH_ACCOUNT_ID，跳过 raw 登录续活。"
			status.OpenID = strings.TrimSpace(s.cfg.OpenID)
		})
		return nil
	}

	s.warmupMu.Lock()
	status := s.warmupStatus
	if !force && (status.Status == "success" || status.Status == "partial") && !s.nextWarmupAt.IsZero() && time.Now().Before(s.nextWarmupAt) {
		s.warmupMu.Unlock()
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	s.warmupStatus.Enabled = true
	s.warmupStatus.Status = "running"
	s.warmupStatus.Message = "正在续活 raw 登录上下文。"
	s.warmupStatus.LastAttemptAt = now
	s.warmupStatus.OpenID = strings.TrimSpace(s.cfg.OpenID)
	s.warmupMu.Unlock()

	err := s.runWarmup(ctx)
	finishedAt := time.Now().UTC().Format(time.RFC3339)

	s.warmupMu.Lock()
	defer s.warmupMu.Unlock()
	s.warmupStatus.Enabled = true
	s.warmupStatus.LastAttemptAt = finishedAt
	s.warmupStatus.OpenID = strings.TrimSpace(s.cfg.OpenID)
	if err != nil {
		s.warmupStatus.Status = "failed"
		s.warmupStatus.Message = err.Error()
		s.warmupStatus.LastErrorAt = finishedAt
		return err
	}
	if strings.TrimSpace(s.warmupStatus.Status) == "" || s.warmupStatus.Status == "running" {
		s.warmupStatus.Status = "success"
		s.warmupStatus.Message = "raw 登录上下文已续活。"
	}
	s.warmupStatus.LastSuccessAt = finishedAt
	s.nextWarmupAt = time.Now().Add(s.nextWarmupTTL())
	return nil
}

func (s *RawSource) nextWarmupTTL() time.Duration {
	if s.warmupMaxTTL <= s.warmupMinTTL {
		return s.warmupMinTTL
	}
	delta := s.warmupMaxTTL - s.warmupMinTTL
	return s.warmupMinTTL + time.Duration(s.rng.Int63n(int64(delta)))
}

func (s *RawSource) runWarmup(ctx context.Context) error {
	fallback, err := s.fallback.FetchDataset(ctx)
	if err != nil {
		return fmt.Errorf("加载续活 fallback 数据失败：%w", err)
	}

	loginStatus, err := s.fetchLoginStatus(ctx, fallback.CartOrder)
	if err != nil {
		return fmt.Errorf("续活登录状态失败：%w", err)
	}
	if len(loginStatusData(loginStatus, nil)) == 0 {
		return fmt.Errorf("续活登录状态失败：返回空登录数据，请检查 MINIAPP_RAW_CONTACTS_ID / MINIAPP_RAW_CUSTOMER_ID / MINIAPP_RAW_IS_DISTRIBUTOR 是否与当前小程序会话一致")
	}

	if _, err := s.fetchRawResponse(ctx, http.MethodPost, "/gateway/customer-service/api/v1/order/app/update/login_time", nil, map[string]any{}); err != nil {
		return fmt.Errorf("刷新登录时间失败：%w", err)
	}

	openID := strings.TrimSpace(s.cfg.OpenID)
	if openID != "" {
		if _, err := s.fetchRawResponse(ctx, http.MethodGet, "/gateway/customer-service/api/v1/order/app/get_bb_auth_status", map[string]any{
			"isPreAuth": false,
			"openId":    openID,
		}, nil); err != nil {
			return fmt.Errorf("校验预授权状态失败：%w", err)
		}
	}

	if _, err := s.fetchRawResponse(ctx, http.MethodGet, "/gateway/marketing-service/api/v1/integral/wx_login_send", nil, nil); err != nil {
		s.warmupMu.Lock()
		s.warmupStatus.Status = "partial"
		s.warmupStatus.Message = "登录上下文已续活，但积分登录通知失败：" + err.Error()
		s.warmupMu.Unlock()
	}

	return nil
}

func (s *RawSource) setWarmupStatus(update func(status *model.RawAuthStatus)) {
	s.warmupMu.Lock()
	defer s.warmupMu.Unlock()
	update(&s.warmupStatus)
}

func (s *RawSource) fetchCategoryTree(ctx context.Context, setting model.GoodsCategorySetting) ([]model.CategoryNode, error) {
	params := map[string]any{
		"layoutType":       defaultInt(setting.LayoutType, 3),
		"maxCategoryLevel": defaultInt(setting.MaxCategoryLevel, 3),
		"categorySortType": defaultInt(setting.CategorySortType, 0),
	}

	var envelope rawEnvelope[[]rawCategoryNode]
	if err := s.requestJSONWithTimeout(ctx, s.categoryTreeTimeout(), http.MethodGet, rawCategoryTreePath, params, nil, &envelope); err != nil {
		return nil, fmt.Errorf("fetch raw category tree: %w", err)
	}

	nodes := make([]model.CategoryNode, 0, len(envelope.Data))
	for _, item := range envelope.Data {
		nodes = append(nodes, convertRawCategoryNode(item))
	}
	return nodes, nil
}

func (s *RawSource) fetchCategoryTreeWithFallback(ctx context.Context, setting model.GoodsCategorySetting, allowCached bool) ([]model.CategoryNode, error) {
	nodes, err := s.fetchCategoryTree(ctx, setting)
	if err == nil {
		s.setCategoryTreeCache(nodes)
		return nodes, nil
	}
	if allowCached {
		if cached, ok := s.cachedCategoryTree(); ok {
			return cached, nil
		}
	}
	return nil, err
}

func (s *RawSource) setCategoryTreeCache(nodes []model.CategoryNode) {
	s.categoryTreeMu.Lock()
	defer s.categoryTreeMu.Unlock()
	s.categoryTreeCache = cloneCategoryNodes(nodes)
	s.categoryTreeCachedAt = time.Now()
}

func (s *RawSource) cachedCategoryTree() ([]model.CategoryNode, bool) {
	s.categoryTreeMu.RLock()
	defer s.categoryTreeMu.RUnlock()
	if len(s.categoryTreeCache) == 0 {
		return nil, false
	}
	return cloneCategoryNodes(s.categoryTreeCache), true
}

func cloneCategoryNodes(nodes []model.CategoryNode) []model.CategoryNode {
	if len(nodes) == 0 {
		return nil
	}
	cloned := make([]model.CategoryNode, 0, len(nodes))
	for _, node := range nodes {
		item := node
		item.Children = cloneCategoryNodes(node.Children)
		cloned = append(cloned, item)
	}
	return cloned
}

func (s *RawSource) categoryTreeTimeout() time.Duration {
	timeout := s.cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	timeout = timeout * 2
	if timeout < 45*time.Second {
		timeout = 45 * time.Second
	}
	if timeout > 90*time.Second {
		timeout = 90 * time.Second
	}
	return timeout
}

func (s *RawSource) fetchCategoryProducts(ctx context.Context, topLevelNodes []model.CategoryNode, perSectionTimeout time.Duration, includePriceStock bool) ([]model.CategorySection, []model.ProductPage, []string, error) {
	requestNodes := flattenCategoryRequestNodes(topLevelNodes)
	pathByKey, _ := buildCategoryTreeMeta(topLevelNodes)
	lineageByKey := buildCategoryLineageKeys(topLevelNodes)

	sections := make([]model.CategorySection, 0, len(requestNodes))
	productMap := make(map[string]model.ProductPage)
	notes := make([]string, 0)

	for _, node := range requestNodes {
		timeout := perSectionTimeout
		if timeout <= 0 {
			timeout = s.sectionTimeout()
		}
		sectionCtx, cancel := context.WithTimeout(ctx, timeout)
		section, err := s.fetchCategorySection(sectionCtx, node, pathByKey[node.Key], lineageByKey[node.Key], includePriceStock)
		cancel()
		if err != nil {
			if cached, ok := s.cachedCategorySection(node.Key); ok {
				section = cached
				notes = append(notes, fmt.Sprintf("raw 分类商品回退 %s（%s）：实时请求失败，已使用最近成功结果：%v", strings.TrimSpace(node.Label), strings.TrimSpace(node.Key), err))
			} else {
				notes = append(notes, fmt.Sprintf("raw 分类商品跳过 %s（%s）：%v", strings.TrimSpace(node.Label), strings.TrimSpace(node.Key), err))
				continue
			}
		} else {
			s.setCategorySectionCache(node.Key, section)
		}
		sections = append(sections, section)

		for _, item := range section.Products {
			productID := normalizedRawProductID(item.SpuID, item.SkuID)
			existing, ok := productMap[productID]
			if !ok {
				productMap[productID] = buildRawProductSkeleton(item, section)
				continue
			}
			existing = mergeObservedCategorySection(existing, section)
			productMap[productID] = existing
		}
	}

	if len(sections) == 0 && len(requestNodes) > 0 {
		if len(notes) > 0 {
			limit := len(notes)
			if limit > 3 {
				limit = 3
			}
			return nil, nil, notes, fmt.Errorf("分类分支没有成功抓到商品列表：%s", strings.Join(notes[:limit], "；"))
		}
		return nil, nil, notes, fmt.Errorf("分类分支没有成功抓到商品列表")
	}

	products := make([]model.ProductPage, 0, len(productMap))
	if includePriceStock {
		for _, item := range productMap {
			products = append(products, item)
		}
		products = s.enrichRawProducts(ctx, products)
		sort.Slice(products, func(i int, j int) bool {
			return products[i].ID < products[j].ID
		})
	}

	return sections, products, notes, nil
}

func (s *RawSource) setCategorySectionCache(key string, section model.CategorySection) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	s.categorySectionMu.Lock()
	defer s.categorySectionMu.Unlock()
	s.categorySectionCache[key] = cloneCategorySection(section)
}

func (s *RawSource) cachedCategorySection(key string) (model.CategorySection, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return model.CategorySection{}, false
	}
	s.categorySectionMu.RLock()
	defer s.categorySectionMu.RUnlock()
	section, ok := s.categorySectionCache[key]
	if !ok {
		return model.CategorySection{}, false
	}
	return cloneCategorySection(section), true
}

func cloneCategorySection(section model.CategorySection) model.CategorySection {
	cloned := section
	if len(section.CategoryKeys) > 0 {
		cloned.CategoryKeys = append([]string(nil), section.CategoryKeys...)
	}
	if len(section.Products) > 0 {
		cloned.Products = append([]model.HomepageProduct(nil), section.Products...)
	}
	return cloned
}

func (s *RawSource) sectionTimeout() time.Duration {
	timeout := s.cfg.Timeout / 3
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	if timeout < 4*time.Second {
		timeout = 4 * time.Second
	}
	if timeout > 12*time.Second {
		timeout = 12 * time.Second
	}
	return timeout
}

func (s *RawSource) targetSyncSectionTimeout() time.Duration {
	timeout := s.cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	timeout = timeout + 10*time.Second
	if timeout < 20*time.Second {
		timeout = 20 * time.Second
	}
	if timeout > 45*time.Second {
		timeout = 45 * time.Second
	}
	return timeout
}

func (s *RawSource) enrichRawProducts(ctx context.Context, products []model.ProductPage) []model.ProductPage {
	if len(products) == 0 {
		return products
	}

	enriched := make([]model.ProductPage, len(products))
	copy(enriched, products)

	sem := make(chan struct{}, s.workerLimit)
	var wg sync.WaitGroup

	for idx := range enriched {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int) {
			defer wg.Done()
			defer func() { <-sem }()

			product := enriched[index]
			info, err := s.fetchProductInfo(ctx, product.SpuID, product.SkuID)
			if err != nil {
				return
			}
			priceList, _ := s.fetchProductPriceList(ctx, product.SpuID, product.SkuID, info.UnitID)
			packageData, _ := s.fetchProductPackage(ctx, product.SpuID, product.SkuID)
			cartSummary, _ := s.fetchCartTotal(ctx)
			chooseData, _ := s.fetchCartChoose(ctx, product.SkuID)
			addCartSummary, _ := s.fetchAddCartTotal(ctx, product.SpuID)
			enriched[index] = mergeRawProductDetail(product, info, priceList, packageData, cartSummary, chooseData, addCartSummary)
		}(idx)
	}

	wg.Wait()
	return enriched
}

func (s *RawSource) fetchCategorySection(ctx context.Context, node model.CategoryNode, categoryPath string, categoryKeys []string, includePriceStock bool) (model.CategorySection, error) {
	subjectPath := firstNonEmptyRaw(strings.TrimSpace(node.PathCode), strings.TrimSpace(node.Key))
	requestBody := map[string]any{
		"checkList":       []any{},
		"includeTagIds":   []any{},
		"includeBrandIds": []any{},
		"sort":            -1,
		"keywords":        "",
		"subjectPath":     subjectPath,
		"pageNum":         1,
		"pageSize":        100,
		"type":            153,
		"isAdaptKeywords": false,
		"isBuyLogo":       true,
	}

	products := make([]model.HomepageProduct, 0)
	pageNum := 1
	pageCount := 1
	for pageNum <= pageCount {
		requestBody["pageNum"] = pageNum

		var envelope rawEnvelope[rawGoodsListData]
		if err := s.requestJSON(ctx, http.MethodPost, rawGoodsListPath, nil, requestBody, &envelope); err != nil {
			return model.CategorySection{}, fmt.Errorf("fetch raw goods list for %s: %w", node.Key, err)
		}

		pageCount = maxInt(1, envelope.Data.Pages)
		pageProducts := convertRawGoodsList(envelope.Data.List)
		if includePriceStock {
			priceInfo, err := s.fetchPriceStock(ctx, envelope.Data)
			if err != nil {
				return model.CategorySection{}, fmt.Errorf("fetch raw price stock for %s page %d: %w", node.Key, pageNum, err)
			}
			for idx := range pageProducts {
				product := pageProducts[idx]
				if enriched, ok := priceInfo[product.SkuID]; ok {
					pageProducts[idx] = mergeRawProductPrice(product, enriched)
				}
			}
		}
		products = append(products, pageProducts...)
		pageNum++
	}

	return model.CategorySection{
		ID:           node.Key,
		Title:        node.Label,
		CategoryKey:  node.Key,
		CategoryPath: firstNonEmptyRaw(strings.TrimSpace(categoryPath), node.PathName, node.Label),
		SubjectPath:  subjectPath,
		CategoryKeys: appendUniqueStrings(nil, categoryKeys...),
		RequestBody:  cloneAnyMap(requestBody),
		Products:     products,
	}, nil
}

func (s *RawSource) fetchPriceStock(ctx context.Context, listData rawGoodsListData) (map[string]rawPriceStockItem, error) {
	if len(listData.List) == 0 {
		return map[string]rawPriceStockItem{}, nil
	}

	itemList := make([]map[string]any, 0, len(listData.List))
	for _, item := range listData.List {
		itemList = append(itemList, map[string]any{
			"queryStock":  true,
			"skuId":       item.SkuID,
			"spuId":       item.SpuID,
			"unitId":      item.UnitID,
			"warehouseId": item.WarehouseID,
		})
	}

	requestBody := map[string]any{
		"goodsShowProducedDate": false,
		"needAllUnit":           true,
		"itemList":              itemList,
		"extReturnVO":           listData.DataMap.ExtReturnVO,
		"levelId":               nil,
	}

	var envelope rawEnvelope[[]rawPriceStockItem]
	if err := s.requestJSON(ctx, http.MethodPost, rawPriceStockPath, nil, requestBody, &envelope); err != nil {
		return nil, err
	}

	lookup := make(map[string]rawPriceStockItem, len(envelope.Data))
	for _, item := range envelope.Data {
		lookup[item.SkuID] = item
	}
	return lookup, nil
}

func (s *RawSource) requestJSON(ctx context.Context, method string, path string, query map[string]any, body any, target any) error {
	return s.requestJSONWithTimeout(ctx, 0, method, path, query, body, target)
}

func (s *RawSource) requestJSONWithTimeout(ctx context.Context, requestTimeout time.Duration, method string, path string, query map[string]any, body any, target any) error {
	fullURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(s.cfg.BaseURL), "/") + path)
	if err != nil {
		return err
	}
	if len(query) > 0 {
		values := fullURL.Query()
		for key, value := range query {
			values.Set(key, fmt.Sprintf("%v", value))
		}
		fullURL.RawQuery = values.Encode()
	}

	var payloadBytes []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payloadBytes = encoded
	}
	attempts := s.retryMax + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := s.waitMinInterval(ctx); err != nil {
			return err
		}
		var payload io.Reader
		if payloadBytes != nil {
			payload = bytes.NewReader(payloadBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), payload)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("xweb_xhr", "1")
		}
		if strings.TrimSpace(s.cfg.UserAgent) != "" {
			req.Header.Set("User-Agent", s.cfg.UserAgent)
		}
		if strings.TrimSpace(s.cfg.AuthorizedAccountID) != "" {
			req.Header.Set("Authorization", "Bearer "+s.cfg.AuthorizedAccountID)
		}
		if strings.TrimSpace(s.cfg.TemplateID) != "" {
			req.Header.Set("xcx-template-id", s.cfg.TemplateID)
		}
		if strings.TrimSpace(s.cfg.Referer) != "" {
			req.Header.Set("Referer", s.cfg.Referer)
		}

		httpClient := s.client
		if requestTimeout > 0 && requestTimeout != s.client.Timeout {
			httpClient = &http.Client{Timeout: requestTimeout}
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request raw source %s %s: %w", method, path, err)
			if attempt+1 < attempts && s.shouldRetryStatus(0) {
				s.sleepBackoff(ctx, attempt)
				continue
			}
			return lastErr
		}

		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("raw source returned status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
			if attempt+1 < attempts && s.shouldRetryStatus(resp.StatusCode) {
				s.sleepBackoff(ctx, attempt)
				continue
			}
			return lastErr
		}

		err = json.NewDecoder(resp.Body).Decode(target)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("decode raw source %s %s: %w", method, path, err)
			if attempt+1 < attempts {
				s.sleepBackoff(ctx, attempt)
				continue
			}
			return lastErr
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("request raw source %s %s failed", method, path)
}

func (s *RawSource) waitMinInterval(ctx context.Context) error {
	if s.cfg.MinInterval <= 0 {
		return nil
	}
	s.requestMu.Lock()
	now := time.Now()
	next := s.lastRequestAt
	if next.Before(now) {
		next = now
	}
	wait := time.Until(next)
	s.lastRequestAt = next.Add(s.cfg.MinInterval)
	s.requestMu.Unlock()
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func (s *RawSource) sleepBackoff(ctx context.Context, attempt int) {
	wait := time.Duration(attempt+1) * 300 * time.Millisecond
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (s *RawSource) shouldRetryStatus(status int) bool {
	if status == 0 {
		return true
	}
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func (s *RawSource) fetchRawResponse(ctx context.Context, method string, path string, query map[string]any, body any) (map[string]any, error) {
	response := map[string]any{}
	if err := s.requestJSON(ctx, method, path, query, body, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func loginStatusData(response map[string]any, err error) map[string]any {
	if err != nil {
		return nil
	}
	data, _ := response["data"].(map[string]any)
	return data
}

func appendUniqueBools(items []bool, value bool) []bool {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

type rawEnvelope[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type rawCategoryNode struct {
	ID                string            `json:"id"`
	CategoryName      string            `json:"categoryName"`
	CategoryPic       string            `json:"categoryPic"`
	PathName          string            `json:"pathName"`
	PathCode          string            `json:"pathCode"`
	Sort              int               `json:"sort"`
	HaveChild         bool              `json:"haveChild"`
	Deep              int               `json:"deep"`
	ChildCategoryList []rawCategoryNode `json:"childCategoryList"`
}

type rawGoodsListData struct {
	PageNum  int            `json:"pageNum"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
	Pages    int            `json:"pages"`
	List     []rawGoodsItem `json:"list"`
	DataMap  struct {
		ExtReturnVO map[string]any `json:"extReturnVO"`
	} `json:"dataMap"`
}

type rawGoodsItem struct {
	ID                string       `json:"id"`
	SkuID             string       `json:"skuId"`
	SpuID             string       `json:"spuId"`
	Cover             string       `json:"cover"`
	Name              string       `json:"name"`
	SkuName           string       `json:"skuName"`
	UnitName          string       `json:"unitName"`
	UnitID            string       `json:"unitId"`
	AllowOrderUnitNum int          `json:"allowOrderUnitNum"`
	CartNum           float64      `json:"cartNum"`
	WarehouseID       int          `json:"warehouseId"`
	TagInfoList       []rawTagInfo `json:"tagInfoList"`
}

type rawTagInfo struct {
	Name string `json:"name"`
}

type rawPriceStockItem struct {
	SkuID              string             `json:"skuId"`
	SpuID              string             `json:"spuId"`
	DiscountAfterPrice float64            `json:"discountAfterPrice"`
	Qty                float64            `json:"qty"`
	DisplayText        string             `json:"displayText"`
	PromotionList      []rawPromotionItem `json:"promotionList"`
	UnitInfoList       []rawUnitInfo      `json:"unitInfoList"`
}

type rawPromotionItem struct {
	Content string `json:"content"`
	Name    string `json:"name"`
}

type rawUnitInfo struct {
	Price              float64 `json:"price"`
	DiscountAfterPrice float64 `json:"discountAfterPrice"`
	Qty                float64 `json:"qty"`
	IsBase             int     `json:"isBase"`
	IsDefault          int     `json:"isDefault"`
	UnitID             string  `json:"unitId"`
	UnitName           string  `json:"unitName"`
	Rate               float64 `json:"rate"`
	BaseUnitName       string  `json:"baseUnitName"`
	DisplayText        string  `json:"displayText"`
	MinOrderQty        float64 `json:"minOrderQty"`
	MaxOrderQty        float64 `json:"maxOrderQty"`
}

type rawGoodsInfoData struct {
	SpuID               string               `json:"spuId"`
	SkuID               string               `json:"skuId"`
	Code                string               `json:"code"`
	BarCode             string               `json:"barCode"`
	Name                string               `json:"name"`
	UnitID              string               `json:"unitId"`
	BaseUnitID          string               `json:"baseUnitId"`
	UnitName            string               `json:"unitName"`
	SkuName             string               `json:"skuName"`
	HasCollect          bool                 `json:"hasCollect"`
	CartNum             float64              `json:"cartNum"`
	CarouselList        []rawGoodsMedia      `json:"carouselList"`
	DetailList          []rawGoodsDetailItem `json:"detailList"`
	ForbidOutStockOrder bool                 `json:"forbidOutStockOrder"`
	FormFieldList       []rawGoodsFormField  `json:"formFieldList"`
	DetailPageShowType  int                  `json:"detailPageShowType"`
	ConfigRecommend     int                  `json:"configRecommend"`
	TagInfoList         []rawTagInfo         `json:"tagInfoList"`
}

type rawGoodsMedia struct {
	CoverURL string `json:"coverUrl"`
	VideoURL string `json:"videoUrl"`
	Type     int    `json:"type"`
}

type rawGoodsDetailItem struct {
	Value    string `json:"value"`
	Type     int    `json:"type"`
	CoverURL string `json:"coverUrl"`
}

type rawGoodsFormField struct {
	ID           string `json:"id"`
	FieldName    string `json:"fieldName"`
	SystemName   string `json:"systemName"`
	Limit        int    `json:"limit"`
	ValueType    int    `json:"valueType"`
	DefaultValue string `json:"defaultValue"`
	Value        string `json:"value"`
}

type rawPriceListItem struct {
	SpuID              string             `json:"spuId"`
	SkuID              string             `json:"skuId"`
	UnitID             string             `json:"unitId"`
	OriginalPrice      float64            `json:"originalPrice"`
	DiscountAfterPrice float64            `json:"discountAfterPrice"`
	PromotionList      []rawPromotionItem `json:"promotionList"`
}

type rawPackageData struct {
	PageNum  int                  `json:"pageNum"`
	PageSize int                  `json:"pageSize"`
	Total    int                  `json:"total"`
	Pages    int                  `json:"pages"`
	List     []rawPackageListItem `json:"list"`
}

type rawPackageListItem struct {
	SpuID       string `json:"spuId"`
	SkuID       string `json:"skuId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rawCartChooseData struct {
	SpuID            string `json:"spuId"`
	DefaultUnitID    string `json:"defaultUnitId"`
	Style            int    `json:"style"`
	GoodsCartCompose struct {
		SkuList []struct {
			SkuRateList []struct {
				Rate        float64 `json:"rate"`
				UnitID      string  `json:"unitId"`
				UnitName    string  `json:"unitName"`
				AllowOrder  int     `json:"allowOrder"`
				MinOrderQty float64 `json:"minOrderQty"`
				MaxOrderQty float64 `json:"maxOrderQty"`
				IsBase      int     `json:"isBase"`
				IsDefault   int     `json:"isDefault"`
			} `json:"skuRateList"`
		} `json:"skuList"`
	} `json:"goodsCartCompose"`
	ExtInfoVO struct {
		ShopConfig struct {
			CurrencySymbolView string `json:"currencySymbolView"`
			EnableMinimumQty   bool   `json:"enableMinimumQty"`
			ShowBookQty        bool   `json:"showBookQty"`
			ShowStockQty       bool   `json:"showStockQty"`
		} `json:"shopConfig"`
		GoodsSetting struct {
			QtyPrecision        int  `json:"qtyPrecision"`
			PricePrecision      int  `json:"pricePrecision"`
			ForbidOutStockOrder bool `json:"forbidOutStockOrder"`
			ShowBookQty         bool `json:"showBookQty"`
			ShowStockQty        bool `json:"showStockQty"`
		} `json:"goodsSetting"`
	} `json:"extInfoVO"`
}

func (s *RawSource) fetchProductInfo(ctx context.Context, spuID string, skuID string) (rawGoodsInfoData, error) {
	params := map[string]any{
		"spuId":           spuID,
		"skuId":           skuID,
		"unitId":          "",
		"isAdaptKeywords": false,
	}
	var envelope rawEnvelope[rawGoodsInfoData]
	if err := s.requestJSON(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/goods/info", params, nil, &envelope); err != nil {
		return rawGoodsInfoData{}, err
	}
	return envelope.Data, nil
}

func (s *RawSource) fetchProductPriceList(ctx context.Context, spuID string, skuID string, unitID string) ([]rawPriceListItem, error) {
	if strings.TrimSpace(unitID) == "" {
		return nil, nil
	}
	requestBody := map[string]any{
		"goodsShowProducedDate": false,
		"levelId":               nil,
		"promotionId":           0,
		"activityType":          7,
		"skuUnitList": []map[string]any{
			{
				"qty":    nil,
				"skuId":  skuID,
				"spuId":  spuID,
				"unitId": unitID,
			},
		},
	}
	var envelope rawEnvelope[[]rawPriceListItem]
	if err := s.requestJSON(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/wx/price/list", nil, requestBody, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *RawSource) fetchProductPackage(ctx context.Context, spuID string, skuID string) (rawPackageData, error) {
	requestBody := map[string]any{
		"skuId":       skuID,
		"spuId":       spuID,
		"wantNum":     1,
		"pageNum":     1,
		"pageSize":    20,
		"ignoreRange": true,
		"statusList":  []int{0, 1},
		"from":        "wx",
	}
	var envelope rawEnvelope[rawPackageData]
	if err := s.requestJSON(ctx, http.MethodPost, "/gateway/goodsservice/api/v1/goods/get_sku_relation_discounts_package", nil, requestBody, &envelope); err != nil {
		return rawPackageData{}, err
	}
	return envelope.Data, nil
}

func (s *RawSource) fetchCartTotal(ctx context.Context) ([]any, error) {
	var envelope rawEnvelope[[]any]
	if err := s.requestJSON(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/cart/get_cart_tot_num", map[string]any{"needAllUnit": true}, nil, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *RawSource) fetchCartChoose(ctx context.Context, skuID string) (rawCartChooseData, error) {
	var envelope rawEnvelope[rawCartChooseData]
	if err := s.requestJSON(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/cart/choose/"+strings.TrimSpace(skuID), nil, nil, &envelope); err != nil {
		return rawCartChooseData{}, err
	}
	return envelope.Data, nil
}

func (s *RawSource) fetchAddCartTotal(ctx context.Context, spuID string) ([]any, error) {
	var envelope rawEnvelope[[]any]
	if err := s.requestJSON(ctx, http.MethodGet, "/gateway/goodsservice/api/v1/wx/cart/get_add_cart_tot", map[string]any{"spuId": spuID}, nil, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func convertRawCategoryNode(item rawCategoryNode) model.CategoryNode {
	children := make([]model.CategoryNode, 0, len(item.ChildCategoryList))
	for _, child := range item.ChildCategoryList {
		children = append(children, convertRawCategoryNode(child))
	}
	return model.CategoryNode{
		Key:         item.ID,
		Label:       item.CategoryName,
		ImageURL:    item.CategoryPic,
		PathName:    item.PathName,
		PathCode:    item.PathCode,
		Depth:       defaultInt(item.Deep, len(strings.Split(strings.TrimSpace(item.PathName), "/"))),
		Sort:        item.Sort,
		HasChildren: item.HaveChild || len(children) > 0,
		Children:    children,
	}
}

func flattenCategoryRequestNodes(nodes []model.CategoryNode) []model.CategoryNode {
	items := make([]model.CategoryNode, 0)
	var walk func([]model.CategoryNode)
	walk = func(list []model.CategoryNode) {
		for _, node := range list {
			items = append(items, node)
			walk(node.Children)
		}
	}
	walk(nodes)
	return items
}

func findCategoryNode(nodes []model.CategoryNode, key string) *model.CategoryNode {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	for idx := range nodes {
		if strings.TrimSpace(nodes[idx].Key) == key {
			return &nodes[idx]
		}
		if child := findCategoryNode(nodes[idx].Children, key); child != nil {
			return child
		}
	}
	return nil
}

func buildCategoryTreeMeta(nodes []model.CategoryNode) (map[string]string, map[string]string) {
	paths := make(map[string]string)
	parents := make(map[string]string)
	var walk func(list []model.CategoryNode, parentPath string, parentKey string)
	walk = func(list []model.CategoryNode, parentPath string, parentKey string) {
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

func buildCategoryLineageKeys(nodes []model.CategoryNode) map[string][]string {
	lineage := make(map[string][]string)
	var walk func(list []model.CategoryNode, ancestors []string)
	walk = func(list []model.CategoryNode, ancestors []string) {
		for _, node := range list {
			current := append(append([]string(nil), ancestors...), strings.TrimSpace(node.Key))
			lineage[node.Key] = appendUniqueStrings(nil, current...)
			if len(node.Children) > 0 {
				walk(node.Children, current)
			}
		}
	}
	walk(nodes, nil)
	return lineage
}

func convertRawGoodsList(items []rawGoodsItem) []model.HomepageProduct {
	products := make([]model.HomepageProduct, 0, len(items))
	for _, item := range items {
		tags := make([]string, 0, len(item.TagInfoList))
		for _, tag := range item.TagInfoList {
			if strings.TrimSpace(tag.Name) != "" {
				tags = append(tags, strings.TrimSpace(tag.Name))
			}
		}
		products = append(products, model.HomepageProduct{
			ProductID:      normalizedRawProductID(item.SpuID, item.SkuID),
			SpuID:          item.SpuID,
			SkuID:          item.SkuID,
			Name:           item.Name,
			Cover:          item.Cover,
			SkuName:        item.SkuName,
			DefaultUnit:    item.UnitName,
			Price:          0,
			StockQty:       0,
			StockText:      "",
			Tags:           tags,
			UnitOptions:    nil,
			PromotionTexts: nil,
		})
	}
	return products
}

func mergeRawProductPrice(product model.HomepageProduct, price rawPriceStockItem) model.HomepageProduct {
	product.Price = defaultFloat(price.DiscountAfterPrice, product.Price)
	product.StockQty = price.Qty
	product.StockText = strings.TrimSpace(price.DisplayText)
	product.PromotionTexts = rawPromotionTexts(price.PromotionList)
	product.UnitOptions = rawUnitOptions(price.UnitInfoList)
	if len(product.UnitOptions) > 0 {
		for _, option := range product.UnitOptions {
			if option.IsDefault {
				product.DefaultUnit = option.UnitName
				product.Price = option.Price
				product.StockQty = option.StockQty
				product.StockText = option.StockText
				break
			}
		}
	}
	return product
}

func rawPromotionTexts(items []rawPromotionItem) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		text := firstNonEmptyRaw(item.Content, item.Name)
		if text != "" {
			values = append(values, text)
		}
	}
	return values
}

func rawUnitOptions(items []rawUnitInfo) []model.UnitOption {
	options := make([]model.UnitOption, 0, len(items))
	for _, item := range items {
		price := defaultFloat(item.DiscountAfterPrice, item.Price)
		options = append(options, model.UnitOption{
			UnitName:  item.UnitName,
			Price:     price,
			BaseUnit:  item.BaseUnitName,
			Rate:      defaultFloat(item.Rate, 1),
			IsDefault: item.IsDefault == 1,
			StockQty:  item.Qty,
			StockText: item.DisplayText,
		})
	}
	sort.SliceStable(options, func(i int, j int) bool {
		if options[i].IsDefault != options[j].IsDefault {
			return options[i].IsDefault
		}
		return options[i].Rate < options[j].Rate
	})
	return options
}

func buildRawProductSkeleton(product model.HomepageProduct, section model.CategorySection) model.ProductPage {
	orderUnits := make([]model.ProductOrderUnit, 0, len(product.UnitOptions))
	for _, option := range product.UnitOptions {
		orderUnits = append(orderUnits, model.ProductOrderUnit{
			UnitID:      "",
			UnitName:    option.UnitName,
			Rate:        option.Rate,
			IsBase:      strings.TrimSpace(option.UnitName) == strings.TrimSpace(option.BaseUnit),
			IsDefault:   option.IsDefault,
			AllowOrder:  true,
			MinOrderQty: 0,
			MaxOrderQty: 0,
		})
	}

	return model.ProductPage{
		ID:                    normalizedRawProductID(product.SpuID, product.SkuID),
		SpuID:                 product.SpuID,
		SkuID:                 product.SkuID,
		SourceType:            "list_skeleton",
		SourceSections:        []string{section.CategoryKey},
		CategoryKey:           section.CategoryKey,
		CategoryPath:          section.CategoryPath,
		CategoryKeys:          appendUniqueStrings(nil, section.CategoryKeys...),
		ObservedCategoryKeys:  []string{section.CategoryKey},
		ObservedCategoryPaths: []string{section.CategoryPath},
		Summary:               product,
		Detail: model.ProductDetail{
			ContractID:   "raw_category_goods_list",
			RequestQuery: map[string]any{"spuId": product.SpuID, "skuId": product.SkuID},
			Name:         product.Name,
			SkuName:      product.SkuName,
			DefaultUnit:  product.DefaultUnit,
			Carousel: []model.ProductMedia{
				{ImageURL: product.Cover, Type: 1},
			},
		},
		Pricing: model.ProductPricing{
			PriceListContractID:       "raw_category_price_stock",
			DefaultStockContractID:    "raw_category_price_stock",
			MultiUnitStockContractID:  "raw_category_price_stock",
			DefaultUnit:               product.DefaultUnit,
			DefaultPrice:              product.Price,
			DefaultStockQty:           product.StockQty,
			DefaultStockText:          product.StockText,
			UnitOptions:               product.UnitOptions,
			PromotionTexts:            product.PromotionTexts,
			PriceListRequestBody:      map[string]any{"subjectPath": section.SubjectPath},
			DefaultStockRequestBody:   map[string]any{"subjectPath": section.SubjectPath},
			MultiUnitStockRequestBody: map[string]any{"subjectPath": section.SubjectPath},
		},
		Package: model.ProductPackage{
			ContractID:  "raw_product_package_pending",
			RequestBody: map[string]any{"spuId": product.SpuID, "skuId": product.SkuID},
			PageNum:     1,
		},
		Context: model.ProductContext{
			CartSummaryContractID: "raw_cart_context_pending",
			CartSummaryQuery:      map[string]any{"needAllUnit": true},
			CartChooseContractID:  "raw_cart_choose_pending",
			CartChoosePath:        "",
			AddCartContractID:     "raw_add_cart_pending",
			AddCartQuery:          map[string]any{"spuId": product.SpuID},
			UnitOptions:           orderUnits,
		},
	}
}

func buildRawProductsFromSections(sections []model.CategorySection) []model.ProductPage {
	productMap := make(map[string]model.ProductPage)
	productOrder := make([]string, 0)
	for _, section := range sections {
		for _, item := range section.Products {
			productID := normalizedRawProductID(item.SpuID, item.SkuID)
			if productID == "" {
				productID = strings.TrimSpace(item.ProductID)
			}
			if productID == "" {
				continue
			}
			product, ok := productMap[productID]
			if !ok {
				product = buildRawProductSkeleton(item, section)
				productOrder = append(productOrder, productID)
			} else {
				product = mergeObservedCategorySection(product, section)
			}
			productMap[productID] = product
		}
	}
	products := make([]model.ProductPage, 0, len(productOrder))
	for _, productID := range productOrder {
		product, ok := productMap[productID]
		if !ok {
			continue
		}
		products = append(products, product)
	}
	return products
}

func filterCategorySectionsByScope(sections []model.CategorySection, scopeKey string) []model.CategorySection {
	scopeKey = strings.TrimSpace(scopeKey)
	if scopeKey == "" {
		return append([]model.CategorySection(nil), sections...)
	}
	filtered := make([]model.CategorySection, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section.CategoryKey) == scopeKey {
			filtered = append(filtered, section)
			continue
		}
		if slices.Contains(appendUniqueStrings(nil, section.CategoryKeys...), scopeKey) {
			filtered = append(filtered, section)
		}
	}
	return filtered
}

func mergeObservedCategorySection(product model.ProductPage, section model.CategorySection) model.ProductPage {
	product.SourceSections = appendUniqueStrings(product.SourceSections, section.CategoryKey)
	product.ObservedCategoryKeys = appendUniqueStrings(product.ObservedCategoryKeys, section.CategoryKey)
	product.ObservedCategoryPaths = appendUniqueStrings(product.ObservedCategoryPaths, section.CategoryPath)
	if strings.TrimSpace(product.CategoryKey) == "" || len(section.CategoryKeys) > len(product.CategoryKeys) {
		product.CategoryKey = section.CategoryKey
		product.CategoryPath = section.CategoryPath
		product.CategoryKeys = appendUniqueStrings(nil, section.CategoryKeys...)
	}
	return product
}

func mergeRawProductDetail(product model.ProductPage, info rawGoodsInfoData, priceList []rawPriceListItem, packageData rawPackageData, cartSummary []any, chooseData rawCartChooseData, addCartSummary []any) model.ProductPage {
	product.SourceType = "raw_detail"
	product.Summary.Name = firstNonEmptyRaw(info.Name, product.Summary.Name)
	product.Summary.SkuName = firstNonEmptyRaw(info.SkuName, product.Summary.SkuName)
	product.Summary.DefaultUnit = firstNonEmptyRaw(info.UnitName, product.Summary.DefaultUnit)
	product.Summary.Tags = rawTagNames(info.TagInfoList, product.Summary.Tags)

	carousel := make([]model.ProductMedia, 0, len(info.CarouselList))
	for _, item := range info.CarouselList {
		carousel = append(carousel, model.ProductMedia{
			ImageURL: item.CoverURL,
			VideoURL: item.VideoURL,
			Type:     item.Type,
		})
	}

	detailAssets := make([]model.ProductMedia, 0, len(info.DetailList))
	detailTexts := make([]string, 0, len(info.DetailList))
	for _, item := range info.DetailList {
		if strings.TrimSpace(item.CoverURL) != "" {
			detailAssets = append(detailAssets, model.ProductMedia{
				ImageURL: item.CoverURL,
				Type:     item.Type,
			})
		}
		if strings.TrimSpace(item.Value) != "" {
			detailTexts = append(detailTexts, strings.TrimSpace(item.Value))
		}
	}

	formFields := make([]model.ProductFormField, 0, len(info.FormFieldList))
	for _, item := range info.FormFieldList {
		formFields = append(formFields, model.ProductFormField{
			ID:           item.ID,
			FieldName:    item.FieldName,
			SystemName:   item.SystemName,
			Limit:        item.Limit,
			ValueType:    item.ValueType,
			DefaultValue: item.DefaultValue,
			Value:        item.Value,
		})
	}

	product.Detail = model.ProductDetail{
		ContractID:          "raw_product_info",
		RequestQuery:        map[string]any{"spuId": product.SpuID, "skuId": product.SkuID, "unitId": "", "isAdaptKeywords": false},
		Code:                info.Code,
		BarCode:             info.BarCode,
		Name:                firstNonEmptyRaw(info.Name, product.Summary.Name),
		SkuName:             firstNonEmptyRaw(info.SkuName, product.Summary.SkuName),
		DefaultUnitID:       info.UnitID,
		BaseUnitID:          info.BaseUnitID,
		DefaultUnit:         firstNonEmptyRaw(info.UnitName, product.Summary.DefaultUnit),
		HasCollect:          info.HasCollect,
		CartNum:             info.CartNum,
		Carousel:            carousel,
		DetailAssets:        detailAssets,
		DetailTexts:         detailTexts,
		ForbidOutStockOrder: info.ForbidOutStockOrder,
		DetailPageShowType:  info.DetailPageShowType,
		ConfigRecommend:     info.ConfigRecommend,
		FormFields:          formFields,
		TagInfo:             rawTagNames(info.TagInfoList, nil),
	}

	if len(carousel) > 0 && strings.TrimSpace(product.Summary.Cover) == "" {
		product.Summary.Cover = carousel[0].ImageURL
	}

	product.Pricing.PriceListContractID = "raw_product_price_list"
	product.Pricing.PriceListRequestBody = map[string]any{
		"goodsShowProducedDate": false,
		"levelId":               nil,
		"promotionId":           0,
		"activityType":          7,
		"skuUnitList": []map[string]any{
			{
				"qty":    nil,
				"skuId":  product.SkuID,
				"spuId":  product.SpuID,
				"unitId": info.UnitID,
			},
		},
	}
	if len(priceList) > 0 {
		product.Pricing.DefaultPrice = defaultFloat(priceList[0].DiscountAfterPrice, product.Pricing.DefaultPrice)
		product.Pricing.PromotionTexts = rawPromotionTexts(priceList[0].PromotionList)
		if len(product.Pricing.PromotionTexts) > 0 {
			product.Summary.PromotionTexts = product.Pricing.PromotionTexts
		}
		product.Summary.Price = product.Pricing.DefaultPrice
	}

	relatedItems := make([]model.ProductPackageItem, 0, len(packageData.List))
	for _, item := range packageData.List {
		relatedItems = append(relatedItems, model.ProductPackageItem{
			SpuID:       item.SpuID,
			SkuID:       item.SkuID,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	product.Package = model.ProductPackage{
		ContractID: "raw_product_package",
		RequestBody: map[string]any{
			"skuId":       product.SkuID,
			"spuId":       product.SpuID,
			"wantNum":     1,
			"pageNum":     1,
			"pageSize":    20,
			"ignoreRange": true,
			"statusList":  []int{0, 1},
			"from":        "wx",
		},
		PageNum:      packageData.PageNum,
		PageSize:     packageData.PageSize,
		Total:        packageData.Total,
		Pages:        packageData.Pages,
		RelatedItems: relatedItems,
	}

	orderUnits := make([]model.ProductOrderUnit, 0)
	for _, sku := range chooseData.GoodsCartCompose.SkuList {
		for _, rate := range sku.SkuRateList {
			orderUnits = append(orderUnits, model.ProductOrderUnit{
				UnitID:      rate.UnitID,
				UnitName:    rate.UnitName,
				Rate:        defaultFloat(rate.Rate, 1),
				IsBase:      rate.IsBase == 1,
				IsDefault:   rate.IsDefault == 1,
				AllowOrder:  rate.AllowOrder == 1,
				MinOrderQty: rate.MinOrderQty,
				MaxOrderQty: rate.MaxOrderQty,
			})
		}
	}
	if len(orderUnits) == 0 {
		orderUnits = product.Context.UnitOptions
	}

	product.Context = model.ProductContext{
		CartSummaryContractID: "raw_cart_total_num",
		CartSummaryQuery:      map[string]any{"needAllUnit": true},
		CartSummary:           cartSummary,
		CartChooseContractID:  "raw_cart_choose",
		CartChoosePath:        "/gateway/goodsservice/api/v1/wx/cart/choose/" + strings.TrimSpace(product.SkuID),
		AddCartContractID:     "raw_add_cart_total",
		AddCartQuery:          map[string]any{"spuId": product.SpuID},
		AddCartSummary:        addCartSummary,
		DefaultUnitID:         firstNonEmptyRaw(chooseData.DefaultUnitID, info.UnitID),
		Style:                 chooseData.Style,
		UnitOptions:           orderUnits,
		Settings: model.ProductContextSettings{
			CurrencySymbol:      firstNonEmptyRaw(chooseData.ExtInfoVO.ShopConfig.CurrencySymbolView, "￥"),
			PricePrecision:      chooseData.ExtInfoVO.GoodsSetting.PricePrecision,
			QtyPrecision:        chooseData.ExtInfoVO.GoodsSetting.QtyPrecision,
			EnableMinimumQty:    chooseData.ExtInfoVO.ShopConfig.EnableMinimumQty,
			ForbidOutStockOrder: chooseData.ExtInfoVO.GoodsSetting.ForbidOutStockOrder,
			ShowBookQty:         chooseData.ExtInfoVO.GoodsSetting.ShowBookQty || chooseData.ExtInfoVO.ShopConfig.ShowBookQty,
			ShowStockQty:        chooseData.ExtInfoVO.GoodsSetting.ShowStockQty || chooseData.ExtInfoVO.ShopConfig.ShowStockQty,
		},
	}
	return product
}

func rawTagNames(tags []rawTagInfo, fallback []string) []string {
	if len(tags) == 0 {
		return fallback
	}
	values := make([]string, 0, len(tags))
	for _, item := range tags {
		if strings.TrimSpace(item.Name) != "" {
			values = append(values, strings.TrimSpace(item.Name))
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func rawLoginFallbackIDs(fallback model.CartOrderAggregate) (string, string) {
	request, _ := fallback.Order.AddDelivery.RequestBody.(map[string]any)
	contactsID := firstNonEmptyRaw(
		stringValueFromAny(request["contactsId"]),
		stringValueFromAny(request["contactId"]),
	)
	customerID := firstNonEmptyRaw(
		stringValueFromAny(request["customerId"]),
		stringValueFromAny(request["businessId"]),
	)
	return contactsID, customerID
}

func rawCustomerIDFromLoginStatus(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
	if related, ok := data["relatedCustomer"].(map[string]any); ok {
		if businessID := stringValueFromAny(related["businessId"]); businessID != "" {
			return businessID
		}
	}
	return firstNonEmptyRaw(
		stringValueFromAny(data["customerId"]),
		stringValueFromAny(data["businessId"]),
	)
}

func rawCustomerIDFromFallback(fallback model.CartOrderAggregate) string {
	request, _ := fallback.Order.AddDelivery.RequestBody.(map[string]any)
	return firstNonEmptyRaw(
		stringValueFromAny(request["customerId"]),
		stringValueFromAny(request["businessId"]),
	)
}

func rawAnalyseAddressBody(fallback model.CartOrderAggregate) map[string]any {
	request, _ := fallback.Order.AnalyseAddress.RequestBody.(map[string]any)
	text := stringValueFromAny(request["text"])
	if text == "" {
		return nil
	}
	return map[string]any{"text": text}
}

func rawFreightPreviewBody(fallback model.CartOrderAggregate, customerID string) map[string]any {
	if customerID == "" {
		return nil
	}
	requestBody := rawFreightBaseBody(fallback.Cart.Detail.Response, customerID)
	if len(requestBody) == 0 {
		return nil
	}
	requestBody["deliveryMethodId"] = 0
	return requestBody
}

func rawFreightSelectedBody(fallback model.CartOrderAggregate, customerID string) map[string]any {
	if customerID == "" {
		return nil
	}
	requestBody := rawFreightBaseBody(fallback.Cart.Detail.Response, customerID)
	if len(requestBody) == 0 {
		return nil
	}
	address := rawPreferredDeliveryAddress(fallback)
	if len(address) == 0 {
		return nil
	}
	requestBody["deliveryMethodId"] = firstNonEmptyRaw(
		stringValueFromAny(address["deliveryId"]),
		stringValueFromAny(address["deliveryMethodId"]),
	)
	requestBody["province"] = address["province"]
	requestBody["city"] = address["city"]
	requestBody["district"] = address["district"]
	return requestBody
}

func rawPreferredDeliveryAddress(fallback model.CartOrderAggregate) map[string]any {
	if response, ok := fallback.Order.DefaultDelivery.Response.(map[string]any); ok {
		if data, ok := response["data"].(map[string]any); ok && len(data) > 0 {
			return data
		}
	}
	if response, ok := fallback.Order.Deliveries.Response.(map[string]any); ok {
		if items, ok := response["data"].([]any); ok {
			for _, raw := range items {
				if item, ok := raw.(map[string]any); ok {
					if boolValueFromAny(item["isDefault"]) {
						return item
					}
				}
			}
			for _, raw := range items {
				if item, ok := raw.(map[string]any); ok {
					return item
				}
			}
		}
	}
	if response, ok := fallback.Order.AddDelivery.Response.(map[string]any); ok {
		if data, ok := response["data"].(map[string]any); ok && len(data) > 0 {
			return data
		}
	}
	return nil
}

func rawFreightBaseBody(detailResponse any, customerID string) map[string]any {
	response, ok := detailResponse.(map[string]any)
	if !ok {
		return nil
	}
	data, _ := response["data"].(map[string]any)
	spuDetail, _ := data["spuDetail"].(map[string]any)
	spuList, _ := spuDetail["spuList"].([]any)
	if len(spuList) == 0 {
		return nil
	}

	skuList := make([]map[string]any, 0, len(spuList))
	var totalQty float64
	var totalAmount float64
	for _, raw := range spuList {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		baseQty := firstNonEmptyFloat(
			floatValueFromAny(item["baseNum"]),
			floatValueFromAny(item["num"]),
			floatValueFromAny(item["qty"]),
		)
		lineAmount := firstNonEmptyFloat(
			floatValueFromAny(item["totPrice"]),
			floatValueFromAny(item["discountAfterTotal"]),
		)
		skuList = append(skuList, map[string]any{
			"skuId":      stringValueFromAny(item["skuId"]),
			"qty":        fmt.Sprintf("%g", baseQty),
			"weight":     "0",
			"giftQty":    0,
			"giftWeight": "0",
			"price":      fmt.Sprintf("%g", lineAmount),
		})
		totalQty += baseQty
		totalAmount += lineAmount
	}
	if len(skuList) == 0 {
		return nil
	}

	return map[string]any{
		"customerId": customerID,
		"qty":        fmt.Sprintf("%g", totalQty),
		"weight":     "0",
		"giftWeight": 0,
		"giftQty":    0,
		"price":      fmt.Sprintf("%g", totalAmount),
		"skuList":    skuList,
	}
}

func upsertRawFreightScenario(items *[]model.ScenarioAction, action model.ScenarioAction) {
	if items == nil {
		return
	}
	for idx := range *items {
		if (*items)[idx].Scenario == action.Scenario {
			(*items)[idx] = action
			return
		}
	}
	*items = append(*items, action)
}

func rawFindFreightScenario(items []model.ScenarioAction, scenario string) *model.ScenarioAction {
	for idx := range items {
		if items[idx].Scenario == scenario {
			return &items[idx]
		}
	}
	return nil
}

func rawFreightLabel(scenario string) string {
	switch strings.TrimSpace(scenario) {
	case "selected_delivery":
		return "已选择配送方式"
	default:
		return "未选择配送方式"
	}
}

func anyMap(value any) map[string]any {
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return nil
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func stringValueFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func floatValueFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		var parsed float64
		fmt.Sscanf(strings.TrimSpace(typed), "%f", &parsed)
		return parsed
	default:
		return 0
	}
}

func boolValueFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func firstNonEmptyFloat(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func normalizedRawProductID(spuID string, skuID string) string {
	if strings.TrimSpace(spuID) == "" || strings.TrimSpace(skuID) == "" {
		return ""
	}
	return strings.TrimSpace(spuID) + "_" + strings.TrimSpace(skuID)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values)+len(additions))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	for _, value := range additions {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func defaultFloat(value float64, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func firstNonEmptyRaw(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

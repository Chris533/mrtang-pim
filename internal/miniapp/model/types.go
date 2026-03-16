package model

type Dataset struct {
	Meta         Meta                  `json:"meta"`
	Contracts    []Contract            `json:"contracts"`
	Homepage     HomepageAggregate     `json:"homepage"`
	CategoryPage CategoryPageAggregate `json:"categoryPage"`
	ProductPage  ProductPageAggregate  `json:"productPage"`
	CartOrder    CartOrderAggregate    `json:"cartOrder"`
}

type Meta struct {
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Notes       []string `json:"notes"`
}

type Contract struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Method       string            `json:"method"`
	OriginalPath string            `json:"originalPath"`
	LocalPath    string            `json:"localPath"`
	Role         string            `json:"role"`
	Request      ContractShape     `json:"request"`
	ResponseKeys []string          `json:"responseKeys"`
	Tags         []string          `json:"tags"`
	Notes        map[string]string `json:"notes,omitempty"`
}

type ContractShape struct {
	QueryKeys []string `json:"queryKeys,omitempty"`
	BodyKeys  []string `json:"bodyKeys,omitempty"`
}

type HomepageAggregate struct {
	Bootstrap    BootstrapState    `json:"bootstrap"`
	Settings     HomepageSettings  `json:"settings"`
	Template     HomepageTemplate  `json:"template"`
	CategoryTabs []CategoryTab     `json:"categoryTabs"`
	Sections     []HomepageSection `json:"sections"`
}

type BootstrapState struct {
	AllowSwitchBranch bool   `json:"allowSwitchBranch"`
	AutoSwitchBranch  bool   `json:"autoSwitchBranch"`
	AppletCount       int    `json:"appletCount"`
	ContactsConfig    []any  `json:"contactsConfig"`
	BBAuthStatus      int    `json:"bbAuthStatus"`
	CartSummary       []any  `json:"cartSummary"`
	LoginStatus       string `json:"loginStatus"`
	CanOrder          bool   `json:"canOrder"`
}

type HomepageSettings struct {
	CorpName             string               `json:"corpName"`
	ShopName             string               `json:"shopName"`
	CurrencySymbol       string               `json:"currencySymbol"`
	ThemeColor           []string             `json:"themeColor"`
	TopBackgroundColor   string               `json:"topBackgroundColor"`
	TopContentColor      string               `json:"topContentColor"`
	PageName             string               `json:"pageName"`
	ShowOnlineService    bool                 `json:"showOnlineService"`
	ShowShoppingCart     bool                 `json:"showShoppingCart"`
	EnableCoupon         bool                 `json:"enableCoupon"`
	EnableIntegral       bool                 `json:"enableIntegral"`
	EnableInvoice        bool                 `json:"enableInvoice"`
	EnablePrePayment     bool                 `json:"enablePrePayment"`
	EnableGoodsFavorite  bool                 `json:"enableGoodsFavorite"`
	PricePrecision       int                  `json:"pricePrecision"`
	QtyPrecision         int                  `json:"qtyPrecision"`
	StockViewRange       int                  `json:"stockViewRange"`
	GoodsListSetting     GoodsListSetting     `json:"goodsListSetting"`
	GoodsCategorySetting GoodsCategorySetting `json:"goodsCategorySetting"`
}

type GoodsListSetting struct {
	LayoutType         int `json:"layoutType"`
	DefaultSort        int `json:"defaultSort"`
	DetailPageShowType int `json:"detailPageShowType"`
	ListPageShowType   int `json:"listPageShowType"`
	ShowSkuSearchBtn   int `json:"showSkuSearchBtn"`
}

type GoodsCategorySetting struct {
	LayoutType         int `json:"layoutType"`
	DefaultSort        int `json:"defaultSort"`
	MaxCategoryLevel   int `json:"maxCategoryLevel"`
	SearchType         int `json:"searchType"`
	CategorySearchType int `json:"categorySearchType"`
	CategorySortType   int `json:"categorySortType"`
}

type HomepageTemplate struct {
	BusinessID        string           `json:"businessId"`
	TemplateName      string           `json:"templateName"`
	PageName          string           `json:"pageName"`
	SharePageURL      string           `json:"sharePageUrl"`
	ShowOnlineService bool             `json:"showOnlineService"`
	ShowShoppingCart  bool             `json:"showShoppingCart"`
	TemplateType      int              `json:"templateType"`
	Modules           []HomepageModule `json:"modules"`
}

type HomepageModule struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Title        string               `json:"title"`
	DisplayStyle int                  `json:"displayStyle,omitempty"`
	DisplayNum   int                  `json:"displayNum,omitempty"`
	Items        []HomepageModuleItem `json:"items,omitempty"`
}

type HomepageModuleItem struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
	CategoryKey string `json:"categoryKey,omitempty"`
}

type CategoryTab struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type HomepageSection struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Sort        string            `json:"sort"`
	PageSize    int               `json:"pageSize"`
	ContractID  string            `json:"contractId"`
	RequestBody map[string]any    `json:"requestBody"`
	Products    []HomepageProduct `json:"products"`
}

type HomepageProduct struct {
	ProductID      string       `json:"productId,omitempty"`
	SpuID          string       `json:"spuId"`
	SkuID          string       `json:"skuId"`
	Name           string       `json:"name"`
	Cover          string       `json:"cover"`
	SkuName        string       `json:"skuName"`
	DefaultUnit    string       `json:"defaultUnit"`
	Price          float64      `json:"price"`
	StockQty       float64      `json:"stockQty"`
	StockText      string       `json:"stockText"`
	Tags           []string     `json:"tags"`
	UnitOptions    []UnitOption `json:"unitOptions"`
	PromotionTexts []string     `json:"promotionTexts"`
}

type UnitOption struct {
	UnitName  string  `json:"unitName"`
	Price     float64 `json:"price"`
	BaseUnit  string  `json:"baseUnit"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"isDefault"`
	StockQty  float64 `json:"stockQty"`
	StockText string  `json:"stockText"`
}

type CategoryPageAggregate struct {
	Context  CategoryPageContext `json:"context"`
	Tree     []CategoryNode      `json:"tree"`
	Sections []CategorySection   `json:"sections"`
}

type CategoryPageContext struct {
	LoginStatus     string               `json:"loginStatus"`
	ContactsName    string               `json:"contactsName"`
	CanOrder        bool                 `json:"canOrder"`
	AuditStatus     int                  `json:"auditStatus"`
	CustomerStatus  int                  `json:"customerStatus"`
	CartItemCount   int                  `json:"cartItemCount"`
	CategorySetting GoodsCategorySetting `json:"categorySetting"`
}

type CategoryNode struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	ImageURL    string         `json:"imageUrl,omitempty"`
	PathName    string         `json:"pathName,omitempty"`
	Depth       int            `json:"depth"`
	Sort        int            `json:"sort"`
	HasChildren bool           `json:"hasChildren"`
	Children    []CategoryNode `json:"children,omitempty"`
}

type CategorySection struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	CategoryKey  string            `json:"categoryKey"`
	CategoryPath string            `json:"categoryPath"`
	RequestBody  map[string]any    `json:"requestBody"`
	Products     []HomepageProduct `json:"products"`
}

type ProductPageAggregate struct {
	Products []ProductPage `json:"products"`
}

type ProductCoverage struct {
	ProductID      string   `json:"productId"`
	SpuID          string   `json:"spuId"`
	SkuID          string   `json:"skuId"`
	Name           string   `json:"name"`
	SourceType     string   `json:"sourceType"`
	SourceSections []string `json:"sourceSections"`
	UnitCount      int      `json:"unitCount"`
	HasMultiUnit   bool     `json:"hasMultiUnit"`
	Priority       string   `json:"priority"`
}

type ProductCoverageBucket struct {
	Priority string            `json:"priority"`
	Count    int               `json:"count"`
	Items    []ProductCoverage `json:"items"`
}

type ProductCoverageSummary struct {
	TotalProducts  int                     `json:"totalProducts"`
	MultiUnitTotal int                     `json:"multiUnitTotal"`
	ByPriority     []ProductCoverageBucket `json:"byPriority"`
	FirstBatch     []ProductCoverage       `json:"firstBatch"`
}

type ProductPage struct {
	ID             string          `json:"id"`
	SpuID          string          `json:"spuId"`
	SkuID          string          `json:"skuId"`
	SourceType     string          `json:"sourceType"`
	SourceSections []string        `json:"sourceSections,omitempty"`
	Summary        HomepageProduct `json:"summary"`
	Detail         ProductDetail   `json:"detail"`
	Pricing        ProductPricing  `json:"pricing"`
	Package        ProductPackage  `json:"package"`
	Context        ProductContext  `json:"context"`
}

type ProductDetail struct {
	ContractID          string             `json:"contractId"`
	RequestQuery        map[string]any     `json:"requestQuery"`
	Code                string             `json:"code"`
	BarCode             string             `json:"barCode"`
	Name                string             `json:"name"`
	SkuName             string             `json:"skuName"`
	DefaultUnitID       string             `json:"defaultUnitId"`
	BaseUnitID          string             `json:"baseUnitId"`
	DefaultUnit         string             `json:"defaultUnit"`
	HasCollect          bool               `json:"hasCollect"`
	CartNum             float64            `json:"cartNum"`
	Carousel            []ProductMedia     `json:"carousel"`
	DetailAssets        []ProductMedia     `json:"detailAssets"`
	DetailTexts         []string           `json:"detailTexts"`
	ForbidOutStockOrder bool               `json:"forbidOutStockOrder"`
	DetailPageShowType  int                `json:"detailPageShowType"`
	ConfigRecommend     int                `json:"configRecommend"`
	FormFields          []ProductFormField `json:"formFields"`
	TagInfo             []string           `json:"tagInfo"`
}

type ProductMedia struct {
	ImageURL string `json:"imageUrl,omitempty"`
	VideoURL string `json:"videoUrl,omitempty"`
	Type     int    `json:"type"`
}

type ProductFormField struct {
	ID           string `json:"id"`
	FieldName    string `json:"fieldName"`
	SystemName   string `json:"systemName"`
	Limit        int    `json:"limit"`
	ValueType    int    `json:"valueType"`
	DefaultValue string `json:"defaultValue"`
	Value        string `json:"value"`
}

type ProductPricing struct {
	PriceListContractID       string         `json:"priceListContractId"`
	PriceListRequestBody      map[string]any `json:"priceListRequestBody"`
	DefaultStockContractID    string         `json:"defaultStockContractId"`
	DefaultStockRequestBody   map[string]any `json:"defaultStockRequestBody"`
	MultiUnitStockContractID  string         `json:"multiUnitStockContractId"`
	MultiUnitStockRequestBody map[string]any `json:"multiUnitStockRequestBody"`
	DefaultUnit               string         `json:"defaultUnit"`
	DefaultPrice              float64        `json:"defaultPrice"`
	DefaultStockQty           float64        `json:"defaultStockQty"`
	DefaultStockText          string         `json:"defaultStockText"`
	UnitOptions               []UnitOption   `json:"unitOptions"`
	PromotionTexts            []string       `json:"promotionTexts"`
}

type ProductPackage struct {
	ContractID   string               `json:"contractId"`
	RequestBody  map[string]any       `json:"requestBody"`
	PageNum      int                  `json:"pageNum"`
	PageSize     int                  `json:"pageSize"`
	Total        int                  `json:"total"`
	Pages        int                  `json:"pages"`
	RelatedItems []ProductPackageItem `json:"relatedItems"`
}

type ProductPackageItem struct {
	SpuID       string `json:"spuId"`
	SkuID       string `json:"skuId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProductContext struct {
	CartSummaryContractID string                 `json:"cartSummaryContractId"`
	CartSummaryQuery      map[string]any         `json:"cartSummaryQuery"`
	CartSummary           []any                  `json:"cartSummary"`
	CartChooseContractID  string                 `json:"cartChooseContractId"`
	CartChoosePath        string                 `json:"cartChoosePath"`
	AddCartContractID     string                 `json:"addCartContractId"`
	AddCartQuery          map[string]any         `json:"addCartQuery"`
	AddCartSummary        []any                  `json:"addCartSummary"`
	DefaultUnitID         string                 `json:"defaultUnitId"`
	Style                 int                    `json:"style"`
	UnitOptions           []ProductOrderUnit     `json:"unitOptions"`
	Settings              ProductContextSettings `json:"settings"`
}

type ProductOrderUnit struct {
	UnitID      string  `json:"unitId"`
	UnitName    string  `json:"unitName"`
	Rate        float64 `json:"rate"`
	IsBase      bool    `json:"isBase"`
	IsDefault   bool    `json:"isDefault"`
	AllowOrder  bool    `json:"allowOrder"`
	MinOrderQty float64 `json:"minOrderQty"`
	MaxOrderQty float64 `json:"maxOrderQty"`
}

type ProductContextSettings struct {
	CurrencySymbol      string `json:"currencySymbol"`
	PricePrecision      int    `json:"pricePrecision"`
	QtyPrecision        int    `json:"qtyPrecision"`
	EnableMinimumQty    bool   `json:"enableMinimumQty"`
	ForbidOutStockOrder bool   `json:"forbidOutStockOrder"`
	ShowBookQty         bool   `json:"showBookQty"`
	ShowStockQty        bool   `json:"showStockQty"`
}

type CartOrderAggregate struct {
	Cart  CartAggregate  `json:"cart"`
	Order OrderAggregate `json:"order"`
}

type CartAggregate struct {
	Add       OperationSnapshot `json:"add"`
	ChangeNum OperationSnapshot `json:"changeNum"`
	List      OperationSnapshot `json:"list"`
	Detail    OperationSnapshot `json:"detail"`
	Settle    OperationSnapshot `json:"settle"`
}

type OrderAggregate struct {
	DefaultDelivery OperationSnapshot `json:"defaultDelivery"`
	Deliveries      OperationSnapshot `json:"deliveries"`
	AnalyseAddress  OperationSnapshot `json:"analyseAddress"`
	AddDelivery     OperationSnapshot `json:"addDelivery"`
	FreightCosts    []ScenarioAction  `json:"freightCosts"`
	Submit          OperationSnapshot `json:"submit"`
}

type OperationSnapshot struct {
	ContractID   string         `json:"contractId"`
	RequestQuery map[string]any `json:"requestQuery,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty"`
	Response     any            `json:"response"`
}

type ScenarioAction struct {
	Scenario     string         `json:"scenario"`
	Label        string         `json:"label"`
	ContractID   string         `json:"contractId"`
	RequestQuery map[string]any `json:"requestQuery,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty"`
	Response     any            `json:"response"`
}

type CartDetailSummary struct {
	VarietyNum       int                     `json:"varietyNum"`
	ItemCount        int                     `json:"itemCount"`
	CartIDs          []string                `json:"cartIds"`
	TotalQty         float64                 `json:"totalQty"`
	BaseUnitTotalQty float64                 `json:"baseUnitTotalQty"`
	TotalAmount      float64                 `json:"totalAmount"`
	TaxRate          float64                 `json:"taxRate"`
	ExemptionFreight float64                 `json:"exemptionFreight"`
	CouponCount      int                     `json:"couponCount"`
	Items            []CartDetailItemSummary `json:"items"`
}

type CartListSummary struct {
	VarietyNum  int                   `json:"varietyNum"`
	ItemCount   int                   `json:"itemCount"`
	TotalQty    float64               `json:"totalQty"`
	TotalAmount float64               `json:"totalAmount"`
	TaxAmount   float64               `json:"taxAmount"`
	Items       []CartListItemSummary `json:"items"`
}

type CartListItemSummary struct {
	CartID         string   `json:"cartId"`
	ProductID      string   `json:"productId"`
	SpuID          string   `json:"spuId"`
	SkuID          string   `json:"skuId"`
	Name           string   `json:"name"`
	SkuName        string   `json:"skuName"`
	UnitName       string   `json:"unitName"`
	Qty            float64  `json:"qty"`
	UnitPrice      float64  `json:"unitPrice"`
	LineAmount     float64  `json:"lineAmount"`
	BaseUnitName   string   `json:"baseUnitName,omitempty"`
	UnitRate       float64  `json:"unitRate"`
	HasMultiUnit   bool     `json:"hasMultiUnit"`
	StockTexts     []string `json:"stockTexts"`
	PromotionTexts []string `json:"promotionTexts"`
}

type CartDetailItemSummary struct {
	CartID         string  `json:"cartId"`
	ProductID      string  `json:"productId"`
	SpuID          string  `json:"spuId"`
	SkuID          string  `json:"skuId"`
	Name           string  `json:"name"`
	SkuName        string  `json:"skuName"`
	UnitName       string  `json:"unitName"`
	Qty            float64 `json:"qty"`
	UnitPrice      float64 `json:"unitPrice"`
	LineAmount     float64 `json:"lineAmount"`
	DefaultUnitID  string  `json:"defaultUnitId,omitempty"`
	BaseUnitID     string  `json:"baseUnitId,omitempty"`
	PromotionCount int     `json:"promotionCount"`
}

type OrderSubmitSummary struct {
	Message          string                     `json:"message"`
	BillID           string                     `json:"billId"`
	CustomerID       string                     `json:"customerId"`
	CustomerName     string                     `json:"customerName"`
	AddressID        string                     `json:"addressId"`
	DeliveryMethodID string                     `json:"deliveryMethodId"`
	CartIDs          []string                   `json:"cartIds"`
	DueAmount        float64                    `json:"dueAmount"`
	FreightAmount    float64                    `json:"freightAmount"`
	RequiresPayment  bool                       `json:"requiresPayment"`
	DeadlineTime     int64                      `json:"deadlineTime"`
	BillType         int                        `json:"billType"`
	PaymentOptions   []OrderPaymentOption       `json:"paymentOptions"`
	ReceiveAddress   OrderReceiveAddressSummary `json:"receiveAddress"`
}

type FreightSummary struct {
	Scenarios []FreightScenarioSummary `json:"scenarios"`
}

type FreightScenarioSummary struct {
	Scenario         string  `json:"scenario"`
	Label            string  `json:"label"`
	DeliveryMethodID string  `json:"deliveryMethodId"`
	CustomerID       string  `json:"customerId"`
	Qty              float64 `json:"qty"`
	TotalAmount      float64 `json:"totalAmount"`
	FreightAmount    float64 `json:"freightAmount"`
	SkuCount         int     `json:"skuCount"`
}

type DefaultDeliverySummary struct {
	Found   bool                    `json:"found"`
	Source  string                  `json:"source"`
	Address *DeliveryAddressSummary `json:"address,omitempty"`
}

type DeliveriesSummary struct {
	Count            int                      `json:"count"`
	DefaultAddressID string                   `json:"defaultAddressId"`
	Items            []DeliveryAddressSummary `json:"items"`
}

type DeliveryAddressSummary struct {
	AddressID     string  `json:"addressId"`
	CustomerID    string  `json:"customerId"`
	CustomerName  string  `json:"customerName"`
	Phone         string  `json:"phone"`
	FullAddress   string  `json:"fullAddress"`
	DetailAddress string  `json:"detailAddress"`
	ProvinceName  string  `json:"provinceName"`
	CityName      string  `json:"cityName"`
	DistrictName  string  `json:"districtName"`
	DeliveryID    string  `json:"deliveryId"`
	DeliveryName  string  `json:"deliveryName"`
	IsDefault     bool    `json:"isDefault"`
	Longitude     float64 `json:"longitude"`
	Latitude      float64 `json:"latitude"`
}

type CheckoutSummary struct {
	CartList        CartListSummary        `json:"cartList"`
	CartDetail      CartDetailSummary      `json:"cartDetail"`
	DefaultDelivery DefaultDeliverySummary `json:"defaultDelivery"`
	Deliveries      DeliveriesSummary      `json:"deliveries"`
	Freight         FreightSummary         `json:"freight"`
	Submit          OrderSubmitSummary     `json:"submit"`
}

type OrderPaymentOption struct {
	Name         string `json:"name"`
	Type         int    `json:"type"`
	PayRecommend int    `json:"payRecommend"`
}

type OrderReceiveAddressSummary struct {
	AddressID    string  `json:"addressId"`
	CustomerName string  `json:"customerName"`
	Phone        string  `json:"phone"`
	FullAddress  string  `json:"fullAddress"`
	DeliveryID   string  `json:"deliveryId"`
	DeliveryName string  `json:"deliveryName"`
	Longitude    float64 `json:"longitude"`
	Latitude     float64 `json:"latitude"`
}

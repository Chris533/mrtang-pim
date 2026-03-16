package model

type Dataset struct {
	Meta      Meta              `json:"meta"`
	Contracts []Contract        `json:"contracts"`
	Homepage  HomepageAggregate `json:"homepage"`
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

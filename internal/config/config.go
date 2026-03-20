package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Admin    AdminConfig
	Security SecurityConfig
	MiniApp  MiniAppConfig
	Supplier SupplierConfig
	Image    ImageConfig
	Workflow WorkflowConfig
	Vendure  VendureConfig
}

type AppConfig struct {
	HTTPAddr  string
	PublicURL string
	DataDir   string
}

type AdminConfig struct {
	SourceAdmins      []string
	ProcurementAdmins []string
}

type SecurityConfig struct {
	APIKey string
}

type MiniAppConfig struct {
	SourceMode            string
	SourceURL             string
	SourceTimeout         time.Duration
	RawTemplateID         string
	RawReferer            string
	RawOpenID             string
	RawContactsID         string
	RawCustomerID         string
	RawIsDistributor      bool
	RawAssetBaseURL       string
	RawConcurrency        int
	RawMinInterval        time.Duration
	RawRetryMax           int
	RawWarmupMinInterval  time.Duration
	RawWarmupMaxInterval  time.Duration
	HomepageSnapshotFile  string
	CategorySnapshotFile  string
	ProductSnapshotFile   string
	CartOrderSnapshotFile string
	AuthorizedAccountID   string
	UserAgent             string
}

type SupplierConfig struct {
	Connector         string
	Code              string
	FilePath          string
	HTTPBaseURL       string
	HTTPSubmitPath    string
	HTTPFetchPath     string
	HTTPToken         string
	HTTPAPIKey        string
	HTTPTimeout       time.Duration
	HTTPSkipTLSVerify bool
}

type ImageConfig struct {
	Processor            string
	WebhookURL           string
	WebhookToken         string
	AllowRemoteDownloads bool
	Timeout              time.Duration
}

type WorkflowConfig struct {
	AutoProcessOnIngest bool
	AutoSyncApproved    bool
	ProcessBatchSize    int
	SyncBatchSize       int
	DefaultStockOnHand  int
	CronHarvest         string
	CronProcess         string
	CronSync            string
}

type VendureConfig struct {
	Endpoint                   string
	ShopEndpoint               string
	Token                      string
	Username                   string
	Password                   string
	LanguageCode               string
	CurrencyCode               string
	ChannelToken               string
	RequestTimeout             time.Duration
	AssetTags                  []string
	ReviewAssetBase            string
	VariantSupplierCodeField   string
	VariantSupplierCostField   string
	VariantConversionRateField string
	VariantSourceProductField  string
	VariantSourceTypeField     string
	ProductTargetAudienceField string
	ProductCEndAssetField      string
}

func Load() Config {
	return Config{
		App: AppConfig{
			HTTPAddr:  getEnv("PIM_HTTP_ADDR", "127.0.0.1:26228"),
			PublicURL: getEnv("PIM_PUBLIC_URL", "http://127.0.0.1:26228"),
			DataDir:   getEnv("PIM_DATA_DIR", "./pb_data"),
		},
		Admin: AdminConfig{
			SourceAdmins:      splitCSV(getEnv("PIM_SOURCE_ADMIN_EMAILS", "")),
			ProcurementAdmins: splitCSV(getEnv("PIM_PROCUREMENT_ADMIN_EMAILS", "")),
		},
		Security: SecurityConfig{
			APIKey: strings.TrimSpace(os.Getenv("PIM_API_KEY")),
		},
		MiniApp: MiniAppConfig{
			SourceMode:            getEnv("MINIAPP_SOURCE_MODE", "snapshot"),
			SourceURL:             strings.TrimSpace(os.Getenv("MINIAPP_SOURCE_URL")),
			SourceTimeout:         getEnvDuration("MINIAPP_SOURCE_TIMEOUT", 20*time.Second),
			RawTemplateID:         getEnv("MINIAPP_RAW_TEMPLATE_ID", "962"),
			RawReferer:            getEnv("MINIAPP_RAW_REFERER", "https://servicewechat.com/wx57f975d225fcd0bf/9/page-frame.html"),
			RawOpenID:             strings.TrimSpace(os.Getenv("MINIAPP_RAW_OPEN_ID")),
			RawContactsID:         strings.TrimSpace(os.Getenv("MINIAPP_RAW_CONTACTS_ID")),
			RawCustomerID:         strings.TrimSpace(os.Getenv("MINIAPP_RAW_CUSTOMER_ID")),
			RawIsDistributor:      getEnvBool("MINIAPP_RAW_IS_DISTRIBUTOR", true),
			RawAssetBaseURL:       strings.TrimSpace(os.Getenv("MINIAPP_RAW_ASSET_BASE_URL")),
			RawConcurrency:        getEnvInt("MINIAPP_RAW_CONCURRENCY", 4),
			RawMinInterval:        getEnvDuration("MINIAPP_RAW_MIN_INTERVAL", 300*time.Millisecond),
			RawRetryMax:           getEnvInt("MINIAPP_RAW_RETRY_MAX", 2),
			RawWarmupMinInterval:  getEnvDuration("MINIAPP_RAW_WARMUP_MIN_INTERVAL", 30*time.Minute),
			RawWarmupMaxInterval:  getEnvDuration("MINIAPP_RAW_WARMUP_MAX_INTERVAL", 60*time.Minute),
			HomepageSnapshotFile:  getEnv("MINIAPP_HOMEPAGE_SNAPSHOT", "./datasets/miniapp/homepage"),
			CategorySnapshotFile:  getEnv("MINIAPP_CATEGORY_SNAPSHOT", "./datasets/miniapp/category-page"),
			ProductSnapshotFile:   getEnv("MINIAPP_PRODUCT_SNAPSHOT", "./datasets/miniapp/product-page"),
			CartOrderSnapshotFile: getEnv("MINIAPP_CART_ORDER_SNAPSHOT", "./datasets/miniapp/cart-order"),
			AuthorizedAccountID:   strings.TrimSpace(os.Getenv("MINIAPP_AUTH_ACCOUNT_ID")),
			UserAgent:             getEnv("MINIAPP_USER_AGENT", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.53(0x18003537) NetType/WIFI Language/zh_CN miniProgram"),
		},
		Supplier: SupplierConfig{
			Connector:         getEnv("SUPPLIER_CONNECTOR", "file"),
			Code:              getEnv("SUPPLIER_CODE", "SUP_A"),
			FilePath:          getEnv("SUPPLIER_FILE", "./datasets/mock_supplier_products.json"),
			HTTPBaseURL:       strings.TrimSpace(os.Getenv("SUPPLIER_HTTP_BASE_URL")),
			HTTPSubmitPath:    getEnv("SUPPLIER_HTTP_SUBMIT_PATH", "/purchase-orders"),
			HTTPFetchPath:     strings.TrimSpace(os.Getenv("SUPPLIER_HTTP_FETCH_PATH")),
			HTTPToken:         strings.TrimSpace(os.Getenv("SUPPLIER_HTTP_TOKEN")),
			HTTPAPIKey:        strings.TrimSpace(os.Getenv("SUPPLIER_HTTP_API_KEY")),
			HTTPTimeout:       getEnvDuration("SUPPLIER_HTTP_TIMEOUT", 15*time.Second),
			HTTPSkipTLSVerify: getEnvBool("SUPPLIER_HTTP_SKIP_TLS_VERIFY", false),
		},
		Image: ImageConfig{
			Processor:            getEnv("IMAGE_PROCESSOR", "mock"),
			WebhookURL:           strings.TrimSpace(os.Getenv("IMAGE_WEBHOOK_URL")),
			WebhookToken:         strings.TrimSpace(os.Getenv("IMAGE_WEBHOOK_TOKEN")),
			AllowRemoteDownloads: getEnvBool("ALLOW_REMOTE_IMAGE_DOWNLOADS", true),
			Timeout:              getEnvDuration("IMAGE_TIMEOUT", 45*time.Second),
		},
		Workflow: WorkflowConfig{
			AutoProcessOnIngest: getEnvBool("AUTO_PROCESS_ON_INGEST", true),
			AutoSyncApproved:    getEnvBool("AUTO_SYNC_APPROVED", false),
			ProcessBatchSize:    getEnvInt("PROCESS_BATCH_SIZE", 20),
			SyncBatchSize:       getEnvInt("SYNC_BATCH_SIZE", 20),
			DefaultStockOnHand:  getEnvInt("DEFAULT_STOCK_ON_HAND", 100),
			CronHarvest:         getEnv("CRON_HARVEST", "0 */6 * * *"),
			CronProcess:         getEnv("CRON_PROCESS", "*/10 * * * *"),
			CronSync:            getEnv("CRON_SYNC", "*/15 * * * *"),
		},
		Vendure: VendureConfig{
			Endpoint:                   getEnv("VENDURE_ADMIN_API", "http://127.0.0.1:26227/admin-api"),
			ShopEndpoint:               getEnv("VENDURE_SHOP_API", defaultVendureShopEndpoint(getEnv("VENDURE_ADMIN_API", "http://127.0.0.1:26227/admin-api"))),
			Token:                      strings.TrimSpace(os.Getenv("VENDURE_ADMIN_TOKEN")),
			Username:                   strings.TrimSpace(os.Getenv("VENDURE_ADMIN_USERNAME")),
			Password:                   strings.TrimSpace(os.Getenv("VENDURE_ADMIN_PASSWORD")),
			LanguageCode:               getEnv("VENDURE_LANGUAGE_CODE", "zh_Hans"),
			CurrencyCode:               getEnv("VENDURE_CURRENCY_CODE", "CNY"),
			ChannelToken:               strings.TrimSpace(os.Getenv("VENDURE_CHANNEL_TOKEN")),
			RequestTimeout:             getEnvDuration("VENDURE_TIMEOUT", 30*time.Second),
			AssetTags:                  splitCSV(getEnv("VENDURE_ASSET_TAGS", "pim,supplier")),
			ReviewAssetBase:            getEnv("PIM_PUBLIC_URL", "http://127.0.0.1:26228"),
			VariantSupplierCodeField:   strings.TrimSpace(os.Getenv("VENDURE_CF_VARIANT_SUPPLIER_CODE")),
			VariantSupplierCostField:   strings.TrimSpace(os.Getenv("VENDURE_CF_VARIANT_SUPPLIER_COST_PRICE")),
			VariantConversionRateField: strings.TrimSpace(os.Getenv("VENDURE_CF_VARIANT_CONVERSION_RATE")),
			VariantSourceProductField:  strings.TrimSpace(os.Getenv("VENDURE_CF_VARIANT_SOURCE_PRODUCT_ID")),
			VariantSourceTypeField:     strings.TrimSpace(os.Getenv("VENDURE_CF_VARIANT_SOURCE_TYPE")),
			ProductTargetAudienceField: strings.TrimSpace(os.Getenv("VENDURE_CF_PRODUCT_TARGET_AUDIENCE")),
			ProductCEndAssetField:      strings.TrimSpace(os.Getenv("VENDURE_CF_PRODUCT_C_END_FEATURED_ASSET")),
		},
	}
}

func ValidateRuntime(cfg Config) error {
	if !isProductionEnv() {
		return nil
	}

	var problems []string

	if strings.TrimSpace(os.Getenv("MRTANG_PIM_ENCRYPTION_KEY")) == "" {
		problems = append(problems, "MRTANG_PIM_ENCRYPTION_KEY is required in production")
	}
	if strings.TrimSpace(cfg.Security.APIKey) == "" {
		problems = append(problems, "PIM_API_KEY is required in production")
	}
	if strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID) == "" {
		problems = append(problems, "MINIAPP_AUTH_ACCOUNT_ID is required in production")
	}
	publicURL := strings.ToLower(strings.TrimSpace(cfg.App.PublicURL))
	if strings.HasPrefix(publicURL, "http://127.0.0.1") || strings.HasPrefix(publicURL, "http://localhost") {
		problems = append(problems, "PIM_PUBLIC_URL must be a public HTTPS origin in production")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.MiniApp.SourceMode)) {
	case "", "snapshot":
		for _, path := range []string{
			cfg.MiniApp.HomepageSnapshotFile,
			cfg.MiniApp.CategorySnapshotFile,
			cfg.MiniApp.ProductSnapshotFile,
			cfg.MiniApp.CartOrderSnapshotFile,
		} {
			if strings.TrimSpace(path) == "" {
				problems = append(problems, "miniapp snapshot paths must not be empty in production")
				continue
			}
			if _, err := os.Stat(path); err != nil {
				problems = append(problems, fmt.Sprintf("snapshot path not found: %s", path))
			}
		}
	case "raw":
		if strings.TrimSpace(cfg.MiniApp.SourceURL) == "" {
			problems = append(problems, "MINIAPP_SOURCE_URL is required when MINIAPP_SOURCE_MODE=raw")
		}
		if strings.TrimSpace(cfg.MiniApp.RawOpenID) == "" {
			problems = append(problems, "MINIAPP_RAW_OPEN_ID is required when MINIAPP_SOURCE_MODE=raw")
		}
		if strings.TrimSpace(cfg.MiniApp.RawCustomerID) == "" {
			problems = append(problems, "MINIAPP_RAW_CUSTOMER_ID is required when MINIAPP_SOURCE_MODE=raw")
		}
	default:
		problems = append(problems, fmt.Sprintf("unsupported MINIAPP_SOURCE_MODE in production: %s", cfg.MiniApp.SourceMode))
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Supplier.Connector), "file") {
		if strings.TrimSpace(cfg.Supplier.FilePath) == "" {
			problems = append(problems, "SUPPLIER_FILE is required when SUPPLIER_CONNECTOR=file")
		} else if _, err := os.Stat(cfg.Supplier.FilePath); err != nil {
			problems = append(problems, fmt.Sprintf("supplier file not found: %s", cfg.Supplier.FilePath))
		}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Supplier.Connector), "http") {
		if strings.TrimSpace(cfg.Supplier.HTTPBaseURL) == "" {
			problems = append(problems, "SUPPLIER_HTTP_BASE_URL is required when SUPPLIER_CONNECTOR=http")
		}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Supplier.Connector), "miniapp_cart_order") {
		if !strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "raw") {
			problems = append(problems, "MINIAPP_SOURCE_MODE=raw is required when SUPPLIER_CONNECTOR=miniapp_cart_order")
		}
		if strings.TrimSpace(cfg.MiniApp.SourceURL) == "" {
			problems = append(problems, "MINIAPP_SOURCE_URL is required when SUPPLIER_CONNECTOR=miniapp_cart_order")
		}
		if strings.TrimSpace(cfg.MiniApp.RawCustomerID) == "" {
			problems = append(problems, "MINIAPP_RAW_CUSTOMER_ID is required when SUPPLIER_CONNECTOR=miniapp_cart_order")
		}
		if strings.TrimSpace(cfg.MiniApp.RawOpenID) == "" {
			problems = append(problems, "MINIAPP_RAW_OPEN_ID is required when SUPPLIER_CONNECTOR=miniapp_cart_order")
		}
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Image.Processor), "webhook") &&
		strings.TrimSpace(cfg.Image.WebhookURL) == "" {
		problems = append(problems, "IMAGE_WEBHOOK_URL is required when IMAGE_PROCESSOR=webhook")
	}

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("invalid production configuration:\n- %s", strings.Join(problems, "\n- "))
}

func isProductionEnv() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return value == "prod" || value == "production"
}

func defaultVendureShopEndpoint(adminEndpoint string) string {
	value := strings.TrimSpace(adminEndpoint)
	if value == "" {
		return "http://127.0.0.1:26227/shop-api"
	}
	if strings.Contains(value, "/admin-api") {
		return strings.Replace(value, "/admin-api", "/shop-api", 1)
	}
	return value
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}

	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

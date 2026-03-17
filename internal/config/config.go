package config

import (
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
	RawConcurrency        int
	RawMinInterval        time.Duration
	RawRetryMax           int
	HomepageSnapshotFile  string
	CategorySnapshotFile  string
	ProductSnapshotFile   string
	CartOrderSnapshotFile string
	AuthorizedAccountID   string
	UserAgent             string
}

type SupplierConfig struct {
	Connector string
	Code      string
	FilePath  string
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
	Endpoint        string
	Token           string
	Username        string
	Password        string
	LanguageCode    string
	CurrencyCode    string
	ChannelToken    string
	RequestTimeout  time.Duration
	AssetTags       []string
	ReviewAssetBase string
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
			RawConcurrency:        getEnvInt("MINIAPP_RAW_CONCURRENCY", 4),
			RawMinInterval:        getEnvDuration("MINIAPP_RAW_MIN_INTERVAL", 300*time.Millisecond),
			RawRetryMax:           getEnvInt("MINIAPP_RAW_RETRY_MAX", 2),
			HomepageSnapshotFile:  getEnv("MINIAPP_HOMEPAGE_SNAPSHOT", "./datasets/miniapp/homepage"),
			CategorySnapshotFile:  getEnv("MINIAPP_CATEGORY_SNAPSHOT", "./datasets/miniapp/category-page"),
			ProductSnapshotFile:   getEnv("MINIAPP_PRODUCT_SNAPSHOT", "./datasets/miniapp/product-page"),
			CartOrderSnapshotFile: getEnv("MINIAPP_CART_ORDER_SNAPSHOT", "./datasets/miniapp/cart-order"),
			AuthorizedAccountID:   strings.TrimSpace(os.Getenv("MINIAPP_AUTH_ACCOUNT_ID")),
			UserAgent:             getEnv("MINIAPP_USER_AGENT", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.53(0x18003537) NetType/WIFI Language/zh_CN miniProgram"),
		},
		Supplier: SupplierConfig{
			Connector: getEnv("SUPPLIER_CONNECTOR", "file"),
			Code:      getEnv("SUPPLIER_CODE", "SUP_A"),
			FilePath:  getEnv("SUPPLIER_FILE", "./datasets/mock_supplier_products.json"),
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
			Endpoint:        getEnv("VENDURE_ADMIN_API", "http://127.0.0.1:26227/admin-api"),
			Token:           strings.TrimSpace(os.Getenv("VENDURE_ADMIN_TOKEN")),
			Username:        strings.TrimSpace(os.Getenv("VENDURE_ADMIN_USERNAME")),
			Password:        strings.TrimSpace(os.Getenv("VENDURE_ADMIN_PASSWORD")),
			LanguageCode:    getEnv("VENDURE_LANGUAGE_CODE", "zh_Hans"),
			CurrencyCode:    getEnv("VENDURE_CURRENCY_CODE", "CNY"),
			ChannelToken:    strings.TrimSpace(os.Getenv("VENDURE_CHANNEL_TOKEN")),
			RequestTimeout:  getEnvDuration("VENDURE_TIMEOUT", 30*time.Second),
			AssetTags:       splitCSV(getEnv("VENDURE_ASSET_TAGS", "pim,supplier")),
			ReviewAssetBase: getEnv("PIM_PUBLIC_URL", "http://127.0.0.1:26228"),
		},
	}
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

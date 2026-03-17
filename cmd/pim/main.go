package main

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"mrtang-pim/internal/config"
	miniappapi "mrtang-pim/internal/miniapp/api"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
	"mrtang-pim/internal/server"
	_ "mrtang-pim/migrations"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	applyDefaultServeHTTPAddr(cfg.App.HTTPAddr)
	applyDefaultServeDataDir(cfg.App.DataDir)

	app := pocketbase.New()
	service := pim.NewService(cfg)
	miniappService := miniappservice.New(newMiniAppSource(cfg), nil)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	server.RegisterHooks(app, cfg, service)
	server.RegisterCrons(app, cfg, service)
	server.RegisterRoutes(app, cfg, service, miniappService)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func applyDefaultServeHTTPAddr(defaultAddr string) {
	defaultAddr = strings.TrimSpace(defaultAddr)
	if defaultAddr == "" || len(os.Args) < 2 || os.Args[1] != "serve" {
		return
	}

	for _, arg := range os.Args[2:] {
		if arg == "--http" || strings.HasPrefix(arg, "--http=") {
			return
		}
	}

	os.Args = append(os.Args, "--http="+defaultAddr)
}

func applyDefaultServeDataDir(defaultDir string) {
	defaultDir = strings.TrimSpace(defaultDir)
	if defaultDir == "" || len(os.Args) < 2 || os.Args[1] != "serve" {
		return
	}

	for _, arg := range os.Args[2:] {
		if arg == "--dir" || strings.HasPrefix(arg, "--dir=") {
			return
		}
	}

	os.Args = append(os.Args, "--dir="+defaultDir)
}

func newMiniAppSource(cfg config.Config) miniappapi.Source {
	snapshot := miniappapi.NewSnapshotSource(
		cfg.MiniApp.HomepageSnapshotFile,
		cfg.MiniApp.CategorySnapshotFile,
		cfg.MiniApp.ProductSnapshotFile,
		cfg.MiniApp.CartOrderSnapshotFile,
	)

	var base miniappapi.Source
	switch strings.ToLower(strings.TrimSpace(cfg.MiniApp.SourceMode)) {
	case "raw":
		base = miniappapi.NewRawSource(miniappapi.RawSourceConfig{
			BaseURL:             cfg.MiniApp.SourceURL,
			AuthorizedAccountID: cfg.MiniApp.AuthorizedAccountID,
			UserAgent:           cfg.MiniApp.UserAgent,
			TemplateID:          cfg.MiniApp.RawTemplateID,
			Referer:             cfg.MiniApp.RawReferer,
			Timeout:             cfg.MiniApp.SourceTimeout,
			Concurrency:         cfg.MiniApp.RawConcurrency,
			MinInterval:         cfg.MiniApp.RawMinInterval,
			RetryMax:            cfg.MiniApp.RawRetryMax,
		}, snapshot)
	default:
		base = snapshot
	}

	return miniappapi.NewOverlaySource(base)
}

func init() {
	if os.Getenv("PB_ENCRYPTION_ENV") == "" {
		_ = os.Setenv("PB_ENCRYPTION_ENV", "MRTANG_PIM_ENCRYPTION_KEY")
	}
}

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

func newMiniAppSource(cfg config.Config) miniappapi.Source {
	var base miniappapi.Source
	if strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "http") {
		base = miniappapi.NewHTTPSource(miniappapi.HTTPSourceConfig{
			URL:                 cfg.MiniApp.SourceURL,
			AuthorizedAccountID: cfg.MiniApp.AuthorizedAccountID,
			UserAgent:           cfg.MiniApp.UserAgent,
			Timeout:             cfg.MiniApp.SourceTimeout,
		})
	} else {
		base = miniappapi.NewSnapshotSource(
			cfg.MiniApp.HomepageSnapshotFile,
			cfg.MiniApp.CategorySnapshotFile,
			cfg.MiniApp.ProductSnapshotFile,
			cfg.MiniApp.CartOrderSnapshotFile,
		)
	}

	return miniappapi.NewOverlaySource(base)
}

func init() {
	if os.Getenv("PB_ENCRYPTION_ENV") == "" {
		_ = os.Setenv("PB_ENCRYPTION_ENV", "MRTANG_PIM_ENCRYPTION_KEY")
	}
}

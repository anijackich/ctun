package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	ctunapp "ctun/internal/app"
	"ctun/internal/profile"
	"ctun/internal/routing"
	"ctun/internal/settings"
)

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	paths, err := ctunapp.NewPaths()
	if err != nil {
		log.Fatal(err)
	}
	store, err := settings.Open(paths.Settings)
	if err != nil {
		log.Fatal(err)
	}
	profiles, err := profile.OpenStore(paths.Profiles)
	if err != nil {
		log.Fatal(err)
	}
	routes, err := routing.Open(paths.Routing)
	if err != nil {
		log.Fatal(err)
	}
	coreService := ctunapp.NewCoreService(paths, store, profiles, routes)
	handlers := []slog.Handler{slog.NewTextHandler(os.Stderr, nil), ctunapp.LogHandler(coreService)}
	logFile, logErr := ctunapp.OpenLogFile(paths.Logs)
	if logErr == nil {
		handlers = append(handlers, slog.NewTextHandler(logFile, nil))
	}
	slog.SetDefault(slog.New(slog.NewMultiHandler(handlers...)))
	if logErr != nil {
		slog.Warn("log file unavailable", "err", logErr)
	}
	slog.Info("ctun starting", "version", version)
	deps := ctunapp.Services{
		Core:      coreService,
		Settings:  ctunapp.NewSettingsService(store, coreService),
		Profiles:  ctunapp.NewProfileService(profiles, coreService),
		Privilege: ctunapp.NewPrivilegeService(coreService),
		Routing:   ctunapp.NewRoutingService(coreService, profiles, routes),
	}

	var window *application.WebviewWindow
	app := application.New(application.Options{
		Name:        "ctun",
		Description: "mihomo desktop client",

		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.ctun.app",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if window != nil {
					window.Show().Focus()
				}
			},
		},
		Services: []application.Service{
			application.NewService(ctunapp.NewAppService(version)),
			application.NewService(deps.Core),
			application.NewService(deps.Settings),
			application.NewService(deps.Profiles),
			application.NewService(deps.Privilege),
			application.NewService(deps.Routing),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	window = newMainWindow(app)
	ctunapp.NewTray(app, window, deps)

	showWindow := func(*application.ApplicationEvent) { window.Show().Focus() }
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(e *application.ApplicationEvent) {
		applySidebarMaterial(window)
		applyTheme(window, store.Get().Theme)
		showWindow(e)
	})
	app.Event.On(ctunapp.EventSettings, func(*application.CustomEvent) { applyTheme(window, store.Get().Theme) })
	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, showWindow)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func newMainWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "ctun",
		Width:            1220,
		Height:           760,
		MinWidth:         1040,
		MinHeight:        620,
		BackgroundType:   application.BackgroundTypeTranslucent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/",
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
			InvisibleTitleBarHeight: 52,
		},
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

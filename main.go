package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"vinyl/internal/app"
	"vinyl/internal/auth"
	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[player.Snapshot]("nowplaying")
	application.RegisterEvent[settings.Settings]("settings")
}

func main() {
	settingsPath, err := settings.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store := settings.NewStore(settingsPath)
	store.Load()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}
	artworkDir := filepath.Join(cacheDir, "Vinyl", "artwork")

	runner := player.OSAScript{}
	players := []player.Player{player.NewAppleMusic(runner, artworkDir), player.NewSpotify(runner)}

	var wailsApp *application.App
	watcher := player.NewWatcher(players,
		func() string { return store.Get().Source },
		func(s player.Snapshot) { wailsApp.Event.Emit("nowplaying", s) },
		time.Now)
	music := catalog.NewMusicLibrary(runner, catalog.NewITunes(), artworkDir)
	service := app.New(store, watcher, music)
	spAuth := auth.NewSpotifyAuth(func() string { return store.Get().SpotifyClientID }, auth.KeychainStore{})
	service.SetSpotifyAuth(spAuth)
	service.SetCatalog("spotify", catalog.NewSpotifyWeb(spAuth.AccessToken))

	wailsApp = application.New(application.Options{
		Name:        "Vinyl",
		Description: "Platine vinyle pour Apple Music et Spotify",
		Services:    []application.Service{application.NewService(service)},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: app.ArtworkMiddleware(artworkDir),
		},
		Mac: application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: false},
	})

	cfg := store.Get()
	widgetOpts := application.WebviewWindowOptions{
		Name:           "widget",
		Title:          "Vinyl",
		URL:            "/widget.html",
		Width:          220,
		Height:         220,
		Hidden:         true, // le widget de bureau le remplace ; ⌘⇧M l'affiche
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		BackgroundType: application.BackgroundTypeTransparent,
		Mac:            application.MacWindow{Backdrop: application.MacBackdropTransparent},
	}
	if cfg.WidgetPlaced {
		widgetOpts.InitialPosition = application.WindowXY
		widgetOpts.X, widgetOpts.Y = cfg.WidgetX, cfg.WidgetY
	}
	widget := wailsApp.Window.NewWithOptions(widgetOpts)

	mainWin := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Vinyl",
		URL:              "/",
		Width:            1100,
		Height:           720,
		MinWidth:         760,
		MinHeight:        520,
		BackgroundColour: application.NewRGB(20, 20, 20),
		Mac: application.MacWindow{
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
			InvisibleTitleBarHeight: 50,
		},
	})
	// Fermer la fenêtre principale la cache ; ⌘Q quitte.
	mainWin.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		mainWin.Hide()
		e.Cancel()
	})
	// Clic sur l'icône du Dock → fenêtre principale.
	wailsApp.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		service.OpenMainWindow()
	})

	// Menu de l'app : ⌘⇧M affiche ou masque le disque flottant.
	menu := application.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.EditMenu)
	fenetre := menu.AddSubmenu("Fenêtre")
	fenetre.Add("Mini-platine").SetAccelerator("CmdOrCtrl+Shift+M").OnClick(func(*application.Context) {
		if widget.IsVisible() {
			widget.Hide()
		} else {
			widget.Show()
		}
	})
	wailsApp.Menu.Set(menu)

	wailsApp.OnShutdown(func() {
		x, y := widget.Position()
		v := store.Get()
		v.WidgetX, v.WidgetY, v.WidgetPlaced = x, y, true
		store.Save(v)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Run(ctx, 500*time.Millisecond)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

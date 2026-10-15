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
	"vinyl/internal/fluid"
	"vinyl/internal/history"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/stats"
	"vinyl/internal/widgetupdate"
)

//go:embed all:frontend/dist
var assets embed.FS

// projectDir est la racine du dépôt, inscrite à la compilation (-ldflags "-X 'main.projectDir=…'").
var projectDir string

func init() {
	application.RegisterEvent[player.Snapshot]("nowplaying")
	application.RegisterEvent[settings.Settings]("settings")
	application.RegisterEvent[widgetupdate.Status]("widgetupdate")
	application.RegisterEvent[stats.Snapshot]("stats")
	application.RegisterEvent[[]history.Entry]("history")
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
	var observe func(player.Snapshot) // réécoute depuis la pile : reprise de la playlist à la fin du morceau
	watcher := player.NewWatcher(players,
		func() string { return store.Get().Source },
		func(s player.Snapshot) {
			wailsApp.Event.Emit("nowplaying", s)
			observe(s) // changements d'état (arrêt, morceau suivant…)
		},
		time.Now)
	music := catalog.NewMusicLibrary(runner, catalog.NewITunes(), artworkDir)
	service := app.New(store, watcher, music)
	spAuth := auth.NewSpotifyAuth(func() string { return store.Get().SpotifyClientID }, auth.KeychainStore{})
	service.SetSpotifyAuth(spAuth)
	service.SetCatalog("spotify", catalog.NewSpotifyWeb(spAuth.AccessToken))

	statsPath, err := stats.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	statStore := stats.NewStore(statsPath, time.Now)
	statStore.Load()
	service.SetStats(statStore)
	histPath, err := history.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	histStore := history.NewStore(histPath)
	histStore.Load()
	service.SetHistory(histStore, func(l []history.Entry) { wailsApp.Event.Emit("history", l) })
	observe = service.ObservePlayback
	watcher.SetOnPlaying(func(np player.NowPlaying) {
		service.ObservePlayback(player.Snapshot{NowPlaying: np}) // position, à chaque sondage
		if np.TrackID == "" {
			return // radio ou flux sans identifiant : pas compté
		}
		// Pile des derniers morceaux écoutés, à droite du disque.
		if pile, changed := histStore.Playing(history.Entry{Source: np.Source, TrackID: np.TrackID, Title: np.Title,
			Artist: np.Artist, Album: np.Album, ArtworkURL: np.ArtworkURL}); changed {
			wailsApp.Event.Emit("history", pile)
		}
		if snap, changed := statStore.Played(np.Source + ":" + np.TrackID); changed {
			wailsApp.Event.Emit("stats", snap)
		}
	})

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
	widgetDir := cfg.WidgetProjectDir
	if widgetDir == "" {
		widgetDir = projectDir
	}
	widgetMgr := widgetupdate.NewManager(&widgetupdate.Updater{
		ProjectDir:    widgetDir,
		AppPath:       "/Applications/Vinyl Widget.app",
		BuildDir:      filepath.Join(cacheDir, "Vinyl", "widget-build"),
		RecordPath:    filepath.Join(filepath.Dir(settingsPath), "widget-install.json"),
		XcodeGenPaths: []string{"/opt/homebrew/bin/xcodegen", "/usr/local/bin/xcodegen"},
		Run:           widgetupdate.ExecRunner{},
		Now:           time.Now,
	}, func(s widgetupdate.Status) { wailsApp.Event.Emit("widgetupdate", s) })
	service.SetWidgetUpdates(widgetMgr)
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
	// Écrans ProMotion (120 Hz) : WebKit bride la page à 60 images/s, ce qui rend l'étagère saccadée
	// au trackpad à côté du reste de macOS. On lève ce bridage dès que la page est prête.
	mainWin.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.InvokeSync(func() {
			if !fluid.Unlock120Hz(mainWin.NativeWindow()) {
				log.Print("affichage : bridage à 60 images/s non levé (réglage WebKit indisponible)")
			}
		})
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

	// Menus macOS par défaut (⌘W, ⌘M, plein écran…) ; le menu Fenêtre gagne « Mini-platine » (⌘⇧M).
	menu := application.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.FileMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.ViewMenu)
	fenetre := menu.AddSubmenu("Fenêtre")
	fenetre.Add("Mini-platine").SetAccelerator("CmdOrCtrl+Shift+M").OnClick(func(*application.Context) {
		if widget.IsVisible() {
			widget.Hide()
		} else {
			widget.Show()
		}
	})
	fenetre.AddSeparator()
	fenetre.AddRole(application.Minimise)
	fenetre.AddRole(application.Zoom)
	fenetre.AddSeparator()
	fenetre.AddRole(application.Front)
	menu.AddRole(application.HelpMenu)
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
	go widgetMgr.Run(ctx, 6*time.Hour)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

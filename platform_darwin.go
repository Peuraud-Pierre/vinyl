//go:build darwin

package main

import (
	"path/filepath"
	"time"

	"vinyl/internal/apps"
	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/widgetupdate"
)

// platform : pièces propres au système. Mac : Musique et Spotify par AppleScript, bibliothèque Apple Music.
type platform struct {
	players []player.Player
	music   *catalog.MusicLibrary
}

func newPlatform(artworkDir string) platform {
	player.IsRunning = apps.Running // une app non lancée n'est pas interrogée (pas d'osascript)
	runner := player.OSAScript{}
	music := catalog.NewMusicLibrary(runner, catalog.NewITunes(), artworkDir)
	am := player.NewAppleMusic(runner, artworkDir)
	am.HD = music.FetchHD // pochette HD (iTunes) du morceau en cours
	return platform{
		players: []player.Player{am, player.NewSpotify(runner)},
		music:   music,
	}
}

// newWidgetManager : mise à jour du widget de bureau (WidgetKit), Mac seulement.
func newWidgetManager(cfg settings.Settings, cacheDir, settingsPath string, emit func(widgetupdate.Status)) *widgetupdate.Manager {
	widgetDir := cfg.WidgetProjectDir
	if widgetDir == "" {
		widgetDir = projectDir
	}
	return widgetupdate.NewManager(&widgetupdate.Updater{
		ProjectDir:    widgetDir,
		AppPath:       "/Applications/Vinyl Widget.app",
		BuildDir:      filepath.Join(cacheDir, "Vinyl", "widget-build"),
		RecordPath:    filepath.Join(filepath.Dir(settingsPath), "widget-install.json"),
		XcodeGenPaths: []string{"/opt/homebrew/bin/xcodegen", "/usr/local/bin/xcodegen"},
		Run:           widgetupdate.ExecRunner{},
		Now:           time.Now,
	}, emit)
}

//go:build windows

package main

import (
	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/widgetupdate"
)

// platform : pièces propres au système. Windows : contrôles multimédia, pas de bibliothèque Apple Music.
type platform struct {
	players []player.Player
	music   *catalog.MusicLibrary
}

func newPlatform(artworkDir string) platform { return platform{} }

// newWidgetManager : pas de widget de bureau sous Windows.
func newWidgetManager(settings.Settings, string, string, func(widgetupdate.Status)) *widgetupdate.Manager {
	return nil
}

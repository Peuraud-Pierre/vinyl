//go:build windows

package main

import (
	"log"
	"time"

	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/widgetupdate"
	"vinyl/internal/winmedia"
)

// platform : pièces propres au système. Windows : Spotify et Apple Music par les contrôles multimédia
// du système ; pas de bibliothèque Apple Music.
type platform struct {
	players []player.Player
	music   *catalog.MusicLibrary
}

func newPlatform(artworkDir string) platform {
	sessions, err := winmedia.NewSystemSessions()
	if err != nil {
		log.Printf("contrôles multimédia de Windows indisponibles : %v", err)
		return platform{}
	}
	return platform{players: []player.Player{
		winmedia.NewPlayer(winmedia.AppleMusicSource, sessions, time.Now, artworkDir),
		winmedia.NewPlayer(winmedia.SpotifySource, sessions, time.Now, artworkDir),
	}}
}

// newWidgetManager : pas de widget de bureau sous Windows.
func newWidgetManager(settings.Settings, string, string, func(widgetupdate.Status)) *widgetupdate.Manager {
	return nil
}

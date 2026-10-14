# Vinyl

Platine vinyle pour Mac qui affiche et pilote Apple Music et Spotify. Inspirée de MD Vinyl (iPhone).

- Widget rond flottant, toujours au premier plan, et fenêtre principale avec platine et bras de lecture.
- Le disque tourne à 33⅓ tours/min avec la pochette au centre ; glisser le bras avance dans le morceau.
- Textures (noir, picture disc, yolk, split, couleur unie), couleur de thème, craquement à la pose de l'aiguille.
- Bibliothèque (playlists, albums, titres likés, artistes), page artiste, carte de partage PNG.

## Prérequis

- macOS, Go 1.26, Node 24, Xcode (outils en ligne de commande).
- CGO doit être actif : si `go env CGO_ENABLED` renvoie `0`, préfixer les commandes Go par `CGO_ENABLED=1`.

## Lancer

    go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
    cd frontend && npm install && cd ..
    CGO_ENABLED=1 wails3 dev          # développement
    CGO_ENABLED=1 wails3 package      # crée bin/vinyl.app

Au premier lancement, macOS demande l'autorisation de piloter Musique et Spotify : accepte.
Pour la modifier : Réglages Système → Confidentialité et sécurité → Automatisation → Vinyl.

## Spotify

1. Crée une app sur https://developer.spotify.com/dashboard.
2. Ajoute l'URI de redirection `http://127.0.0.1:43821/callback`.
3. Colle le Client ID dans Vinyl → Réglages, puis « Se connecter ».

Le jeton est stocké dans le Trousseau macOS (entrée « Vinyl.Spotify »).
La lecture et la platine passent par l'app Spotify et fonctionnent sans connexion ; la connexion sert à la bibliothèque et aux pages artiste.

## Tests

    CGO_ENABLED=1 go test . ./internal/...
    cd frontend && npm test

Tests d'intégration avec les vraies apps et l'API iTunes (désactivés par défaut) :

    VINYL_LIVE=1 go test ./internal/player/ ./internal/catalog/ -run Live -v

Les commandes Go ciblent `. ./internal/...` : `./...` inclut les dossiers mobiles générés par Wails (`build/ios`, `build/android`), qui ne compilent pas sur macOS.

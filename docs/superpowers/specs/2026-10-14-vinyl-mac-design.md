# Vinyl pour Mac — Spec de conception

Date : 2026-10-14
Statut : en relecture

## 1. Objectif

Recréer sur macOS une copie fidèle de l'app iPhone **MD Vinyl** (MD Studio) : une platine vinyle virtuelle qui affiche et pilote la musique en cours d'Apple Music ou de Spotify.

- **Usage** : personnel. Pas de distribution sur l'App Store, pas de publicités ni d'achats intégrés.
- **Stack** : Go 1.26, **Wails v3** (v3.0.0-beta.29 ou plus récente), frontend Vite + TypeScript + **Tailwind CSS v4**.
- **Écran principal** : la platine. L'utilisateur ouvre surtout l'app pour regarder le vinyle tourner.
- **Ce qu'on ne reprend pas de MD Vinyl** : son nom, ses icônes, ses images et ses textures. Tout est recréé.

### Critères de réussite

1. Quand un morceau joue dans Musique ou Spotify, le disque tourne avec la pochette en moins d'une seconde.
2. Cliquer sur la pochette met en pause ou relance la lecture. Le disque ralentit, s'arrête et le bras se lève.
3. Glisser le bras sur le disque déplace la lecture dans le morceau.
4. Le widget flottant et la fenêtre principale restent synchronisés.
5. La bibliothèque (playlists, albums, titres likés) et la page artiste fonctionnent pour les deux sources.

## 2. Architecture

```
vinyl/
├── main.go                    démarrage Wails v3, création des 2 fenêtres, menu barre d'app
├── internal/
│   ├── player/
│   │   ├── player.go          interface Player + types NowPlaying, State
│   │   ├── osascript.go       exécution d'AppleScript (wrapper sur /usr/bin/osascript, injectable)
│   │   ├── applemusic.go      implémentation Player pour l'app Musique
│   │   ├── spotify.go         implémentation Player pour l'app Spotify
│   │   └── watcher.go         sondage toutes les 500 ms, détection de changements, sélection de la source
│   ├── catalog/
│   │   ├── catalog.go         interface Catalog (Artist, TopTracks, Albums, Library…)
│   │   ├── itunes.go          API iTunes Search (sans clé)
│   │   ├── musiclibrary.go    playlists et albums de la bibliothèque Musique (AppleScript)
│   │   └── spotifyweb.go      Spotify Web API + OAuth PKCE
│   ├── auth/keychain.go       stockage du jeton Spotify dans le Trousseau macOS
│   ├── settings/settings.go   préférences JSON dans ~/Library/Application Support/Vinyl/settings.json
│   └── app/service.go         service Wails exposé au frontend (bindings + événements)
└── frontend/
    ├── widget.html / src/widget.ts
    ├── main.html   / src/main.ts
    └── src/
        ├── turntable/         rendu disque, bras, textures, sons (partagé par les 2 fenêtres)
        ├── views/             platine, bibliothèque, artiste, personnaliser, réglages
        └── styles.css         Tailwind v4 + variables de thème (--accent)
```

### Interface Player (Go)

```go
type State string // "playing" | "paused" | "stopped"

type NowPlaying struct {
    Source     string        // "applemusic" | "spotify"
    TrackID    string
    Title, Artist, Album string
    Duration   time.Duration
    Position   time.Duration
    State      State
    ArtworkURL string        // data: URL (Musique) ou https (Spotify)
}

type Player interface {
    Name() string
    Running() bool           // l'app est-elle lancée ? (ne la lance jamais)
    NowPlaying(ctx context.Context) (NowPlaying, error)
    PlayPause(ctx context.Context) error
    Next(ctx context.Context) error
    Previous(ctx context.Context) error
    Seek(ctx context.Context, pos time.Duration) error
    PlayItem(ctx context.Context, id string) error // playlist, album, titre
}
```

- **Pochette dans Musique** : extraite via AppleScript (`raw data of artwork 1`), écrite dans un cache `~/Library/Caches/Vinyl/` indexé par TrackID, puis servie au frontend par l'asset server de Wails.
- **Pochette dans Spotify** : `artwork url of current track`.

### Flux de données

1. **Sondage** : le `watcher` interroge les Players toutes les 500 ms.
2. **Choix de la source** : en mode *Auto*, la source active est celle qui joue. Si les deux jouent, la dernière à avoir démarré l'emporte. Un réglage permet de forcer *Apple Music* ou *Spotify*.
3. **Émission de `nowplaying`** : l'événement Wails part vers les deux fenêtres quand le morceau ou l'état change. Sinon, il part toutes les 2 s pour recaler la position.
4. **Position affichée** : entre deux événements, le frontend fait avancer la position localement.
5. **Actions** : le frontend appelle `Service.PlayPause`, `Seek`, `Next`, `Previous` et `PlayItem`. Le watcher est relancé immédiatement après chaque action.

### Fenêtres

- **Widget** : fenêtre 220×220, sans bordure, fond transparent, toujours au premier plan, déplaçable.
  - Position mémorisée dans les réglages.
  - Commandes (⏮ ⏯ ⏭ ⤢) au survol. ⤢ ouvre ou met au premier plan la fenêtre principale.
- **Fenêtre principale** : 1100×720, redimensionnable, barre de titre transparente façon macOS.
- **Au démarrage** : le widget s'ouvre. La fenêtre principale s'ouvre via ⤢, le Dock ou le menu.
- **Fermeture** : fermer la fenêtre principale ne quitte pas l'app. On quitte avec ⌘Q.

## 3. Interface

Maquette validée : `.superpowers/brainstorm/…/ui-design.html`.

### Platine (vue par défaut de la fenêtre principale)

- **Composition** : socle sombre, plateau, disque avec sillons et reflet, pochette en étiquette centrale, bras de lecture à droite.
- **Infos et commandes** : sous la platine, titre, artiste — album, barre de progression et ⏮ ⏯ ⏭.
- **Fond** : dégradé radial teinté par la couleur du thème.
- **Rotation** : 33⅓ tours/min (1,8 s par tour).
  - Accélération de 0,6 s au démarrage, décélération de 1,2 s à l'arrêt.
  - Implémentée avec `requestAnimationFrame` et un angle accumulé, pour que la vitesse varie de façon continue.

### Bras de lecture

- **Angles** : bras levé = 4° (repos). Aiguille sur le bord extérieur = 18° (début du morceau). Aiguille près de l'étiquette = 34° (fin du morceau).
- **Pendant la lecture** : `angle = 18 + (34 − 18) × position/durée`.
- **Glisser** : on peut faire glisser le bras.
  - Relâché entre 18° et 34° : `Seek` à la position correspondante, puis lecture si la musique était en pause.
  - Relâché avant 12° : pause, et le bras retourne au repos.
- **Clic sur le bras levé** : il se pose et la lecture reprend.
- **Craquement** : un court bruit de vinyle (fichier son libre de droits, généré ou CC0) joue quand l'aiguille se pose. On le désactive dans les réglages.

### Gestes

- **Clic sur la pochette** (fenêtre ou widget) : lecture/pause.
- **Barre d'espace** (fenêtre principale) : lecture/pause.
- **Flèches ← / →** : morceau précédent / suivant.

### Textures du disque (Personnaliser)

| Texture | Rendu |
|---|---|
| Noir classique | disque noir, sillons, reflet |
| Picture disc | la pochette couvre tout le disque, avec des sillons légers par-dessus |
| Yolk | un disque coloré au centre (le « jaune ») entouré d'une couleur claire, avec 2 couleurs réglables |
| Split | une moitié dans une couleur, l'autre moitié dans une autre, avec 2 couleurs réglables |
| Couleur unie | disque translucide d'une couleur réglable |

### Thème

- **Couleur d'accent** : 4 préréglages et un sélecteur libre. Elle s'applique à la sélection du menu, la barre de progression, le contrepoids du bras et la teinte du fond.
- **Mode** : sombre uniquement, comme MD Vinyl.

### Fenêtre principale : menu latéral

- **Lecture** : Platine
- **Bibliothèque** : Playlists, Albums, Titres likés (Spotify uniquement), Artistes
- **App** : Personnaliser, Réglages
- **Pied du menu** : sélecteur de source (Auto / Apple Music / Spotify) et état de la connexion Spotify.

## 4. Bibliothèque et page artiste

### Bibliothèque

| Élément | Apple Music | Spotify |
|---|---|---|
| Playlists | AppleScript (`user playlists`) | `GET /me/playlists` |
| Albums | AppleScript (albums distincts de la bibliothèque, pochette du 1er titre) | `GET /me/albums` |
| Titres likés | — (masqué) | `GET /me/tracks` |
| Artistes | artistes distincts de la bibliothèque | `GET /me/following?type=artist` |

- **Affichage** : grille de pochettes. Un clic lance la lecture (`PlayItem`) et ramène sur la Platine.
- **Pagination** : par 50, avec chargement au défilement.

### Page artiste

- **Accès** : clic sur le nom de l'artiste depuis la platine, ou depuis la vue Artistes.
- **Contenu** : grande image d'en-tête (pochette ou photo d'artiste en flou), meilleurs titres (10), albums (grille) et bouton « Radio de l'artiste ».
- **Source Apple Music** : API iTunes Search (`/lookup?id=…&entity=song|album`).
  - Un titre présent dans la bibliothèque est lu directement.
  - Sinon, il s'ouvre dans Musique via son URL `music://`.
  - La radio de l'artiste ouvre la station de l'artiste dans Musique.
- **Source Spotify** : `GET /artists/{id}`, `/artists/{id}/top-tracks` et `/artists/{id}/albums`.
  - La lecture passe par AppleScript : `play track "spotify:track:…"`.
  - La radio est approximée en lançant les meilleurs titres de l'artiste. L'API de recommandations n'est pas garantie, donc on ne s'en sert pas.

### Carte de partage

- **Rendu** : une image PNG 1080×1350 générée côté frontend (canvas), avec le vinyle (texture actuelle), la pochette, le titre et l'artiste sur le fond du thème.
- **Actions** : « Copier » (presse-papier) ou « Enregistrer… » (boîte de dialogue native).

## 5. Connexion Spotify

- **Méthode** : OAuth 2.0 **Authorization Code + PKCE**, sans secret client.
  - L'utilisateur crée une app gratuite sur developer.spotify.com et colle son **Client ID** dans Réglages.
  - Redirect URI : `http://127.0.0.1:<port>/callback`, avec un serveur HTTP local éphémère.
- **Scopes** : `user-library-read playlist-read-private user-follow-read user-read-playback-state`.
- **Jetons** : access et refresh token stockés dans le Trousseau (service `Vinyl.Spotify`). L'access token est rafraîchi automatiquement.

## 6. Gestion des erreurs

| Situation | Comportement |
|---|---|
| Aucun lecteur lancé | Platine à l'arrêt, bras levé, message « Lance Musique ou Spotify ». L'app ne lance jamais les lecteurs d'elle-même. |
| Permission d'automatisation refusée (erreur AppleScript −1743) | Écran d'aide : Réglages Système → Confidentialité et sécurité → Automatisation → Vinyl. Bouton qui ouvre ce panneau. |
| Spotify non connecté ou jeton invalide | Vues Spotify avec un bouton « Se connecter ». La platine continue de marcher, car elle passe par AppleScript. |
| Erreur réseau ou HTTP (iTunes, Spotify) | Message dans la vue concernée et bouton « Réessayer ». La platine n'est pas affectée. Délai d'attente de 10 s par requête. |
| Erreur 429 de Spotify | Respect de `Retry-After`. |
| Pochette indisponible | Étiquette générique (couleur du thème et note de musique). |
| AppleScript lent (> 2 s) | Sondage annulé et reprise au tick suivant. Pas d'empilement d'appels. |

## 7. Tests

- **Go (`go test ./...`)**
  - Analyse des sorties AppleScript : séparateur fixe entre les champs, gestion des titres contenant des guillemets.
  - Watcher : choix de la source en mode Auto, émission seulement si quelque chose change, recalage toutes les 2 s. Testé avec de faux Players et une horloge injectée.
  - Clients iTunes et Spotify : testés avec `httptest` sur des réponses JSON enregistrées.
  - PKCE : verifier et challenge.
- **Frontend (Vitest)**
  - Conversion position ↔ angle du bras, seuils de glisser.
  - Accélération et décélération de la rotation.
  - Génération des styles de texture.
- **Manuel** : scénarios avec Musique, avec Spotify, sans lecteur, permission refusée, et hors ligne.

## 8. Hors périmètre

- Widgets du Bureau ou du Centre de notifications : WidgetKit exige Swift. Le widget flottant les remplace.
- Siri, Spotlight, paroles, publicités, achats intégrés.
- Éditeur de vinyle avancé (dessin, stickers) : seuls la texture et ses couleurs sont réglables.
- API MusicKit (compte Apple Developer payant). On pourra l'ajouter plus tard comme nouvelle implémentation de `Catalog`.
- Signature et notarisation de l'app : build local uniquement.

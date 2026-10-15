# Vinyl sur Windows — conception

Date : 2026-10-15 · Branche : `feat/windows`, créée depuis `feat/refonte-graphique`

## But

Que Vinyl fonctionne sur **macOS et Windows**, distribué par la release GitHub : les utilisateurs Windows
installent `Vinyl-Setup.exe`, voient la platine avec le morceau en cours et pilotent la lecture de
**Spotify** et d'**Apple Music pour Windows**. La version Mac ne perd aucune fonction.

Réussi quand, sur le PC Windows de l'utilisateur (Windows 10/11, WebView2) :

- avec Spotify puis avec Apple Music : morceau, pochette, position affichés ; disque qui tourne, bras ;
  lecture/pause, suivant, précédent, glissé du bras (avance) ; aléatoire / répétition si l'app les accepte ;
- la pile des derniers écoutés se remplit, et la réécoute d'un disque revient ensuite au morceau
  interrompu sans toucher à la file du lecteur ;
- l'étagère et les pages d'albums / playlists fonctionnent avec Spotify ;
- l'installateur installe (menu Démarrer) et la désinstallation est propre ;
- une release est produite automatiquement (Mac + Windows) à partir d'un tag.

Hors périmètre : version Mac Intel (l'utilisateur ne la demande pas) ; tout autre lecteur Windows
(navigateurs, VLC…) ; signature de code Windows (avertissement SmartScreen accepté, documenté).

## Décisions (validées avec l'utilisateur)

- Lecteurs Windows : Spotify et Apple Music.
- Accès aux lecteurs : **directement depuis Go** via les contrôles multimédia de Windows
  (`Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager`, « GSMTC »), par une
  bibliothèque de liaison WinRT pour Go — pas de programme d'aide C#, pas de PowerShell (coût de
  démarrage incompatible avec l'effort d'économie d'énergie).
- Installation Windows : installateur `.exe` (modèle NSIS déjà présent dans `build/windows/nsis`).
- Release : workflow GitHub Actions déclenché par un tag `v*`, Mac (arm64) + Windows (amd64).

## Architecture

Un seul code source. Les parties propres à une plateforme sont isolées par contraintes de
compilation (`//go:build darwin` / `windows`), comme déjà pour `internal/fluid` et `internal/apps`.

**Commun (inchangé)** : tout le frontend, `internal/app` (service), `player.Watcher` et son sondage
adaptatif, `history`, `stats`, `settings`, `catalog.SpotifyWeb`, `auth` (jeton Spotify via
`go-keyring`, qui utilise déjà le Gestionnaire d'identification sur Windows).

**Mac uniquement (comportement inchangé)** : lecteurs AppleScript (`player.AppleMusic`,
`player.Spotify`, `OSAScript`), bibliothèque Apple Music (`catalog.MusicLibrary`, mosaïques),
`widgetupdate` (widget WidgetKit), `fluid` (120 i/s), `apps` (NSRunningApplication), ouverture des
réglages d'Automatisation.

**Windows uniquement (nouveau)** : paquet `internal/winmedia` (voir ci-dessous).

**Assemblage au démarrage** : `main.go` appelle une fonction par plateforme,
`platformPlayers(...)` / `platformCatalogs(...)` / `platformWidget(...)`, définie dans
`platform_darwin.go` et `platform_windows.go`. Le reste du code ne voit que les interfaces existantes
(`player.Player`, `player.Modes`, `player.BackReplayer`, `catalog.Catalog`).

**Frontend** : une méthode `Platform()` du service (« darwin » / « windows ») permet de masquer ce qui
n'existe pas : bandeau de mise à jour du widget, bouton d'autorisation d'Automatisation, étagère
Apple Music (remplacée par « La bibliothèque Apple Music n'est disponible que sur Mac »). La
mini-platine flottante (fenêtre Wails) reste disponible sur les deux plateformes.

**Ouverture de liens** (`OpenURL`) : `open` sur Mac, ouverture via Wails (`Browser.OpenURL`) ou le
gestionnaire de protocole Windows sur Windows (liens `spotify:` et `https:`).

## `internal/winmedia` — lecteurs Windows

Deux couches, pour tester la logique sans Windows :

1. **`Sessions`** (interface) : liste les sessions multimédia et agit sur l'une d'elles.
   Implémentation réelle (`sessions_windows.go`) sur GSMTC : `TryGetMediaPropertiesAsync` (titre,
   artiste, album, miniature), `GetTimelineProperties` (position, durée), `GetPlaybackInfo` (état,
   aléatoire, répétition), et `TryTogglePlayPauseAsync`, `TrySkipNextAsync`, `TrySkipPreviousAsync`,
   `TryChangePlaybackPositionAsync`, `TryChangeShuffleActiveAsync`, `TryChangeAutoRepeatModeAsync`.
   Implémentation factice pour les tests.
2. **`Player`** (logique, testée partout) : un lecteur par source, reconnu par l'identifiant
   d'application de la session (`SourceAppUserModelId`) : `Spotify.exe` → « spotify » ; Apple Music
   pour Windows (`AppleInc.AppleMusicWin…`) → « applemusic ». Implémente :
   - `player.Player` : `NowPlaying` (pas de session → `ErrNotRunning`), `PlayPause`, `Next`,
     `Previous`, `Seek` ; `PlayItem` : pour Spotify, voir « Étagère » ; Apple Music → erreur
     « non disponible sur Windows ».
   - `player.Modes` : aléatoire et répétition via GSMTC (erreur claire si l'app ne les gère pas).
   - `player.BackReplayer` : `ReplayBack` = « précédent » N fois puis attente (≤ 1,5 s) que le titre
     visé apparaisse, un ou deux crans de plus si la pile est décalée, sinon retour par « suivant » et
     repositionnement ; `JumpForward` = « suivant » N fois puis position retrouvée.
   - Pas de `ReplayResumer` (pas de lecture « une fois » sous Windows) : si le morceau n'est plus
     dans la file, message « Ce morceau n'est plus dans la file », comme sur Mac.

**Identifiant de morceau** : GSMTC n'en fournit pas ; `TrackID` = empreinte stable de
« titre + artiste + album » (compteur, pile, réécoute s'appuient dessus).

**Pochette** : la miniature GSMTC est lue une fois par morceau et écrite dans le dossier de pochettes
(`/artwork/<empreinte>.img`), servie par le middleware existant ; la teinte et la pile l'utilisent
comme sur Mac.

**Coût** : appels WinRT dans le processus, au rythme du sondage adaptatif existant (1 s en lecture,
3 s à l'arrêt) ; pas de processus lancé.

## Étagère et lecture d'albums (Spotify sur Windows)

- Bibliothèque, albums, playlists, artistes, titres likés : `catalog.SpotifyWeb` (inchangé).
- Lancer un album / une playlist / un titre (`PlayFrom`, `PlayItem`) : via l'API web Spotify
  (`PUT /me/player/play` avec `context_uri` et `offset`, aléatoire via `PUT /me/player/shuffle`).
  Nécessite **Spotify Premium** et un appareil Spotify actif. En cas de refus (pas Premium, aucun
  appareil) : ouverture du lien `spotify:` (l'app s'ouvre sur l'album, l'utilisateur lance la lecture).
  Le scope OAuth `user-modify-playback-state` est ajouté (une reconnexion Spotify est demandée une
  fois).
- File d'attente : Spotify via l'API web (existant) ; Apple Music sur Windows : « File indisponible ».
- Apple Music sur Windows : pas d'étagère (aucun accès à la bibliothèque).

## Fabrication et release

- Windows : `wails3 package` (Windows amd64, WebView2) produit `Vinyl-Setup.exe` (NSIS) ; la
  compilation Windows n'utilise pas cgo pour nos paquets (`fluid`, `apps` ont leurs versions
  « autre plateforme »).
- `.github/workflows/release.yml`, déclenché par `push` d'un tag `v*` :
  - job **macos-latest** : Go, Node, `wails3`, `wails3 package`, `ditto` → `Vinyl-mac.zip` ;
  - job **windows-latest** : Go, Node, `wails3`, NSIS, `wails3 package` → `Vinyl-Setup.exe` ;
  - job **release** : crée la release GitHub du tag et y joint les deux fichiers (notes : app non
    signée → clic droit « Ouvrir » sur Mac, « Informations complémentaires → Exécuter quand même »
    sur Windows).
- `.github/workflows/ci.yml` : tests Go et frontend sur macOS et Windows à chaque push (vérifie la
  compilation Windows sans PC).

## Données

`os.UserConfigDir()` / `os.UserCacheDir()` déjà utilisés : `%AppData%\Vinyl` et
`%LocalAppData%\Vinyl` sur Windows. Aucun changement de format.

## Erreurs

- Aucune session (app fermée) → « Lance Musique ou Spotify » (existant).
- Commande refusée par l'app (ex. aléatoire non géré) → erreur affichée par l'appelant, sans plantage.
- Échec WinRT ponctuel → traité comme une erreur passagère par le `Watcher` (dernier état conservé).
- Lecture d'album refusée par l'API Spotify → repli sur l'ouverture du lien `spotify:`.

## Tests

- `winmedia.Player` testé avec des sessions factices (reconnaissance des apps, empreinte de morceau,
  états, commandes, réécoute : recul, crans supplémentaires, retour, introuvable).
- Assemblage par plateforme : compilation vérifiée par la CI Windows ; tests existants inchangés.
- Lecture d'album Spotify via l'API web : client HTTP factice (succès, 403 pas Premium, 404 aucun
  appareil → repli).
- Vérification manuelle sur le PC de l'utilisateur, liste fournie : Spotify puis Apple Music —
  affichage, commandes, glissé du bras, pile et réécoute, étagère Spotify, installation /
  désinstallation.

## Risques

- **Liaison WinRT en Go** : bibliothèque moins répandue ; la mise au point des appels asynchrones
  (`IAsyncOperation`) se fera avec le PC de l'utilisateur. Repli possible si bloquant : petit
  programme d'aide (approche B), à décider avec l'utilisateur.
- **Identifiant de session d'Apple Music pour Windows** : à confirmer sur le PC (reconnaissance par
  préfixe `AppleInc.AppleMusic`).
- **Aléatoire / répétition** : certaines apps ne les exposent pas à GSMTC ; les touches restent alors
  sans effet visible (message).

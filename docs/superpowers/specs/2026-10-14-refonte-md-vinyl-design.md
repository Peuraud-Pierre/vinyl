# Refonte « comme MD Vinyl » — Spec de conception

Date : 2026-10-14
Statut : en relecture
Branche de départ : `feat/refonte-graphique`
Références visuelles : `screen/` (captures et enregistrement d'écran de MD Vinyl sur iPhone)

## 1. Objectif

Rapprocher l'app Mac Vinyl de MD Vinyl, telle que montrée dans `screen/`. On part de la refonte déjà présente sur `feat/refonte-graphique` : fond teinté par la pochette, commandes en verre, dock vertical, pochette + vinyle + bras.

### Critères de réussite

1. **Platine** : on y retrouve, comme sur `vinylvisual(withoutprice).png` :
   - un vinyle noir fumé et translucide ;
   - une étiquette noire, avec le titre et l'artiste en blanc ;
   - un bras droit dans son socle en verre au repos.
2. **Compteur** : en défilant sous la platine, une bande sombre affiche « Morceaux lus depuis AAAA.MM.JJ » avec un compteur à rouleaux. Le compteur augmente à chaque nouveau morceau lu.
3. **Étagère** : elle présente les albums et playlists en coverflow 3D, avec les tranches nommées visibles et un reflet. Elle se parcourt à la molette et au clavier.
4. **Page album** : un clic sur un album ou une playlist ouvre sa page. On y trouve la pochette, les infos, « Lecture », « Lecture aléatoire », et la liste des titres avec leurs durées et les cœurs.
5. **File d'attente** : le panneau montre les titres à venir. On y active l'aléatoire et la répétition, sur Musique comme sur Spotify.
6. **Partage** : il produit une image verticale 1080×1920 façon story.

## 2. Ce qui existe et qu'on garde

- **Backend** : `internal/player`, `internal/catalog`, `internal/settings`, `internal/auth`, `internal/app`, `internal/widgetupdate`.
- **Frontend** :
  - `shell/shell.ts` : fond teinté, boutons en verre, dock ;
  - `ui/theme.ts` : teinte extraite de la pochette ;
  - `turntable/*` ;
  - `views/*` ;
  - les animations `vy-*` de `styles.css`.
- **Conventions** :
  - valeurs utilisateur passées à AppleScript uniquement via argv ;
  - `osascript -e script -- args` ;
  - commandes Go avec `CGO_ENABLED=1` sur `. ./internal/...` ;
  - bindings générés avec `wails3 generate bindings -clean=true -ts -i`.

## 3. Platine

### Vinyle

- **Texture « noir »** : rendu fumé translucide.
  - Dégradés coniques et radiaux superposés pour un marbrage gris sombre.
  - Sillons concentriques et deux reflets diagonaux.
  - Bord légèrement plus clair.
- **Étiquette** (diamètre ≈ 46 % du disque) :
  - fond **noir** et liseré blanc fin ;
  - titre en gras blanc et artiste en blanc atténué, centrés ;
  - deux inscriptions courtes en arc de cercle sur le pourtour intérieur : « VINYL · 33⅓ RPM » et le nom de l'album ;
  - trou central argenté.
- **Texture « picture disc »** : l'étiquette prend la teinte claire extraite de la pochette, avec un texte foncé (cf. aperçu In Utero).
- **Autres textures** (Yolk, Split, Couleur unie) : inchangées.

### Bras

- **Au repos** : bras droit et vertical. Contrepoids rectangulaire noir en haut, tête de lecture noire coudée en bas.
- **Socle** : rectangle arrondi en verre (fond translucide et liseré clair) autour du pivot.
- **Comportement** : rotations, glisser et clic inchangés (`arm.ts`, `ARM_GEOMETRY`). Seul le dessin change.

### Bande « statistiques »

- **Place** : sous la platine, dans la même vue qui défile. Fond `#1b1b1b` sur toute la largeur.
- **Compteur à rouleaux** : 8 chiffres, chacun dans une case noire arrondie, avec le chiffre précédent et le suivant à demi visibles en haut et en bas.
- **Animation** : quand le total change, les rouleaux tournent vers la nouvelle valeur en 0,6 s.
- **Texte** : « Morceaux lus depuis 2026.10.14 », suivi d'une icône de partage qui ouvre le partage story du compteur.

## 4. Étagère coverflow

- **Route** : `#/etagere/albums` ou `#/etagere/playlists`. L'onglet actif est mémorisé.
- **En-tête** :
  - en haut à gauche, un bouton menu qui ouvre le menu existant (Personnaliser, Réglages, Artistes, Likés) ;
  - au centre, une commande segmentée en verre « Albums | Playlists » ;
  - en haut à droite, un bouton recherche qui filtre l'étagère par titre ou artiste.
- **Disposition**, pour l'élément sélectionné d'indice `f` :
  - élément `i = f` : de face au centre, `translateZ(+120px)`, légère ombre ;
  - éléments `i < f` : rotateY(+70°), décalés vers la gauche ;
  - éléments `i > f` : rotateY(−70°), décalés vers la droite ;
  - écart de 64 px entre deux tranches, perspective 1300 px ;
  - les tranches visibles portent « **Titre** Artiste » écrit verticalement (bas vers haut), sur un fond teinté par la pochette ;
  - reflet sous les boîtiers (`-webkit-box-reflect`, fondu vers le bas).
- **Navigation** :
  - molette ou trackpad horizontal et vertical, accumulé pour avancer d'un cran tous les 60 px ;
  - flèches ← et → ;
  - un clic sur un boîtier latéral le sélectionne ;
  - un clic sur le boîtier central, ou la touche Entrée, ouvre la page album.
- **Chargement** :
  - pages de 50 via `Library`, chargées quand la sélection approche de la fin ;
  - pochettes Musique paresseuses via `ItemArtwork` (2 exports à la fois au plus, déjà en place).
- **Performance** : seuls les éléments à ±12 de la sélection sont dans le DOM.

## 5. Page album ou playlist

- **Route** : `#/album/<source>/<id encodé>`. Le même écran sert pour une playlist (`kind` dans la réponse).
- **Disposition** :
  - **haut** : bouton retour (rond, en verre) à gauche, bouton partager (rond) à droite ;
  - **gauche** (≈ 40 %) : grande pochette carrée avec son reflet ;
  - **droite** : titre en gras, artiste, « Genre · Année » ;
  - **sous les infos** : deux pilules contour, « Lecture » et « Lecture aléatoire » ;
  - **en dessous, la liste des titres** :
    - numéro sur 2 chiffres (`01`) ;
    - ♥ devant le numéro si le titre est aimé ;
    - titre ;
    - durée au format `3′37″` ;
    - séparateurs fins ;
  - quand on fait défiler la liste, l'en-tête se réduit à une ligne de titre centrée (cf. `playlistalbum.png`).
- **Fond** : teinte extraite de cette pochette, via `ui/theme.ts`.
- **Actions** :
  - clic sur un titre : lance l'album ou la playlist à partir de ce titre ;
  - « Lecture » : lance depuis le début ;
  - « Lecture aléatoire » : active l'aléatoire puis lance ;
  - après chaque action, retour à la platine.

### Backend : `Tracks`

```go
type Track struct {
    ID         string `json:"id"`          // persistent ID (Musique) ou URI spotify:track:…
    Number     int    `json:"number"`
    Title      string `json:"title"`
    Artist     string `json:"artist"`
    DurationMs int64  `json:"durationMs"`
    Loved      bool   `json:"loved"`
}
type Collection struct {
    ID, Kind, Title, Artist, Genre, Year, ArtworkURL, Source string // Kind : album | playlist
    Tracks []Track
}
// Catalog gagne une méthode :
Tracks(ctx context.Context, id string) (Collection, error)
```

- **Musique**
  - **Album** (`album:<nom>\x1f<artiste>`) : titres de `library playlist 1` dont l'album (et l'artiste d'album, avec repli sur l'album seul) correspond. Ils sont triés par `disc number` puis `track number`.
  - **Champs lus** : `persistent ID`, `name`, `artist`, `duration`, `track number`, `genre`, `year` et `favorited` (avec repli sur `loved` selon la version de macOS, dans un `try`).
  - **Playlist** (`playlist:<id>`) : titres dans l'ordre de la playlist.
  - **Format de sortie** : enregistrements séparés par `\x1e`, champs séparés par `\x1f`. Les durées en secondes sont lues avec `parseSeconds` (virgule acceptée).
- **Spotify**
  - **Album** : `GET /albums/{id}` (genre souvent absent : on laisse vide), plus `GET /me/tracks/contains?ids=…` pour les cœurs, par lots de 50.
  - **Playlist** : `GET /playlists/{id}` et `GET /playlists/{id}/tracks`, paginés jusqu'à 500 titres.
- **Service** : `Tracks(source, id string) (catalog.Collection, error)`.

### Backend : lancer à partir d'un titre

`PlayFrom(source, collectionID, trackID string, shuffle bool) error` :

- **Musique**
  - **Playlist** : `play` du titre `trackID` *dans* la playlist, pour que la suite continue dans la playlist.
  - **Album** : remplissage de la playlist temporaire « Vinyl — Album » (comportement existant), puis `play` du titre dans cette playlist.
  - **Aléatoire** : `set shuffle enabled to shuffle` avant la lecture.
- **Spotify** : `set shuffling to shuffle`, puis `play track "<uri du titre>" in context "<uri de l'album ou de la playlist>"`.
  - Si cette forme n'est pas acceptée par la version de Spotify installée : `play track "<uri du contexte>"`, puis on accepte de démarrer au début.

## 6. File d'attente

- **Ouverture** : panneau latéral droit (≈ 380 px, en verre sombre), depuis l'icône file d'attente du dock. On le ferme avec Échap, un clic en dehors, ou l'icône à nouveau.
- **Contenu** :
  - pochette du morceau en cours (inclinée de −4°, avec ombre) ;
  - « File d'attente » ;
  - morceau en cours en gras ;
  - liste des suivants (titre + artiste), cliquable.
- **Boutons ronds en bas** :
  - **Aléatoire** : bascule ;
  - **Répéter** : cycle aucun → tout → un seul. L'état actif est affiché en plein (fond sombre, icône claire).

### Backend : `Queue`

```go
type RepeatMode string // "off" | "all" | "one"
type QueueState struct {
    Shuffle  bool       `json:"shuffle"`
    Repeat   RepeatMode `json:"repeat"`
    Upcoming []Track    `json:"upcoming"` // au plus 30
    Note     string     `json:"note"`     // ex. "Ordre aléatoire" quand la suite n'est pas connue
}
```

- **Interface `Player`**, nouvelles méthodes :
  - `PlaybackModes(ctx) (shuffle bool, repeat RepeatMode, error)` ;
  - `SetShuffle(ctx, bool) error` ;
  - `SetRepeat(ctx, RepeatMode) error`.
- **Musique**
  - **Modes** : `shuffle enabled` et `song repeat` (`off`, `all`, `one`).
  - **Titres à venir** : titres de `current playlist` après l'index du morceau en cours. On s'arrête à 30 titres, et on repart au début de la playlist si la répétition est sur « tout ».
  - **Aléatoire actif** : `Upcoming` est vide et `Note` vaut « Ordre aléatoire ».
- **Spotify**
  - **Modes** : `shuffling` et `repeating` (booléen : `true` donne « all »). Le mode « one » n'existe pas en AppleScript : `SetRepeat("one")` est accepté, mais traité comme « all ».
  - **Titres à venir** : `GET /me/player/queue` quand le compte est connecté. Sinon, `Note` vaut « Connecte Spotify pour voir la suite ».
- **Service** : `Queue() (QueueState, error)`, `SetShuffle(bool) error`, `SetRepeat(string) error`.
- **Rafraîchissement** : à l'ouverture du panneau, puis à chaque événement `nowplaying` qui change de morceau.

## 7. Compteur d'écoutes

- **Nouveau paquet** `internal/stats` : `Store{ Count int64; Since string /* AAAA.MM.JJ */ }`, enregistré dans `~/Library/Application Support/Vinyl/stats.json`.
- **Première création** : `Since` prend la date du jour.
- **Fichier absent ou corrompu** : on repart de zéro, avec la date du jour.
- **Ce qui compte comme une écoute** : le watcher appelle `stats.Played(trackID)` quand une source passe à l'état `playing` sur un `TrackID` différent du dernier compté.
  - Un même morceau mis en pause puis relancé ne compte pas deux fois.
  - Un `TrackID` vide ne compte pas.
- **Diffusion** : événement Wails `stats` à chaque changement, et méthode `Stats() stats.Snapshot`.
- **Écriture** : à chaque incrément, via un fichier temporaire puis un renommage.

## 8. Partage story

- **Image** : canvas 1080×1920.
  - Fond teinté par la pochette.
  - Pochette en haut, inclinée de −8°, débordant légèrement du bord.
  - Vinyle (texture et étiquette de la platine) en bas à droite, rogné par le bord.
  - « VINYL » en petites capitales, en bas à gauche.
- **Variantes** : morceau en cours, album (depuis sa page), compteur (« N morceaux lus depuis … »).
- **Aperçu** : fenêtre superposée, avec les boutons Copier et Enregistrer… (mécanisme existant, `SaveShareImage`).

## 9. Gestion des erreurs

| Situation | Comportement |
|---|---|
| `Tracks` en échec (Musique fermée, autorisation, réseau) | La page album affiche le message et « Réessayer ». Le bouton retour reste actif. |
| Album sans genre ou sans année | La ligne « Genre · Année » n'affiche que ce qui est connu, ou disparaît. |
| File Spotify sans connexion | `Note` explicite, et les boutons aléatoire et répéter restent actifs (AppleScript). |
| Mode de répétition non pris en charge | Accepté, puis remplacé par le mode le plus proche ; l'interface relit l'état réel ensuite. |
| `stats.json` illisible | Repart de zéro avec la date du jour. Pas d'erreur visible. |
| Étagère vide | Message « Rien ici pour l'instant ». |

## 10. Tests

- **Go**
  - Analyse des sorties `Tracks`, `PlaybackModes` et `Upcoming` (faux Runner), dont les cas suivants :
    - virgule décimale ;
    - « missing value » ;
    - `favorited` absent ;
    - index en fin de playlist.
  - Spotify `Tracks` et `Queue` via `httptest`.
  - `stats` : premier lancement, incrément, même morceau, ID vide, fichier corrompu, accès concurrents.
  - Watcher : appelle `Played` au bon moment.
  - Tests live (`VINYL_LIVE=1`) : les nouveaux scripts compilent et répondent.
- **Vitest**
  - Format des durées `3′37″` (0, 59 s, plus d'une heure).
  - Positions du coverflow (angle, décalage, z) pour `i < f`, `i = f` et `i > f`.
  - Fenêtre ±12.
  - Accumulateur de molette.
  - Chiffres et décalages des rouleaux du compteur.
- **Visuel** : captures Chrome headless avec le faux backend (`.superpowers/mock`), comparées à `screen/`.

## 11. Hors périmètre

- Mise en page portrait (boutons « PLAY » en pilules).
- Carte « Playlist du jour ».
- Écrans d'achat.
- Vraie file « À suivre » de Musique (non exposée en AppleScript).
- Enregistrement vidéo du partage (icône caméra de MD Vinyl).

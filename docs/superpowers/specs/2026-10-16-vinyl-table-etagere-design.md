# Platine : grande table et étagère de playlists — conception

Date : 2026-10-16 · Branche : `feat/table-etagere`, créée depuis `main`

## But

Refaire la scène de la platine comme une pièce : une grande table en bois occupe tout le bas de la
fenêtre, la platine et la pile des derniers disques sont posées dessus (plus petites qu'aujourd'hui), et
une petite étagère murale, au-dessus de la pile, montre les playlists de l'utilisateur. Un clic sur
l'étagère ouvre la vue Étagère des playlists.

Réussi quand :

- la table couvre toute la largeur de la fenêtre, en bas, quelle que soit sa taille ;
- la platine (pochette, disque, bras, touches) et la pile 3D reposent visiblement sur la table, plus petites
  qu'aujourd'hui, et fonctionnent comme avant (clics, glisser du bras, animations, réécoute) ;
- l'étagère montre jusqu'à 12 tranches aux couleurs des playlists de la source active et une pochette
  posée en biais (playlist en cours si elle en fait partie, sinon la première) ;
- un clic n'importe où sur l'étagère ouvre la vue Étagère des playlists, au début, en mode étagère ;
- sans playlists ou en cas d'erreur, la planche reste visible, vide et cliquable, sans message d'erreur.

## Décisions (validées avec l'utilisateur)

- Disposition B de la maquette (`.superpowers/brainstorm/…/disposition-v3.html`) : petite étagère à droite,
  tranches serrées + une pochette en biais ; la pile garde son rendu 3D.
- Contenu : « mes playlists » (bibliothèque de la source active, ordre de la vue Étagère).
- Clic : vue Étagère au début (pas de centrage sur la playlist cliquée, pas de lecture directe).
- Technique : éléments HTML/CSS dans la scène existante (approche 1), pas de canvas ni d'image peinte.

## Scène

La scène actuelle (`views/platine.ts`) garde son principe : une scène de taille fixe mise à l'échelle dans
la place disponible (`fit`), la pile placée par-dessus avec la même échelle. Changements :

- **Table** : bande pleine largeur en bas de la page, y compris sous la colonne de commandes (élément
  `#pl-table` dans la section de la page, pas dans `#pl-wrap` qui s'arrête avant la colonne ; positionné en
  `left:0; right:0; bottom:0`, derrière la platine et la pile), hauteur =
  le tiers bas de la fenêtre (le dessus de la table est aligné sur la ligne où reposent platine et pile,
  recalculée dans `fit`). Bois : même palette que la table actuelle (noyer, veinage léger), chant avant
  plus clair, légère ombre portée des objets posés.
- **Platine** : mise à l'échelle supplémentaire `TT_SCALE = 0.75` (pochette, disque, bras et touches
  ensemble), posée sur la ligne de la table, à gauche.
- **Pile** : `PILE_SCALE = 0.7` appliqué en plus de l'échelle de scène, posée sur la ligne de la table, à
  droite ; la petite table à pieds (`vy-table*`) est supprimée.
- **Étagère** : `#pl-shelf`, au-dessus de la pile et à gauche de la colonne de commandes, mise à l'échelle
  avec la scène.
- Inchangés : colonne de commandes, boutons Partager / Réglages, mini-platine (par-dessus la table),
  compteur sous la scène, message au-dessus de la pile.

Les positions sont calculées par une fonction pure `layoutScene(width, height)` → positions et échelles de
la platine, de la pile, de l'étagère et hauteur de la table (testable sans navigateur).

## Étagère (`ui/wallshelf.ts`)

`mountWallShelf(el, deps)` avec `deps = { load(): Promise<Item[]>; current(): Promise<string>; open(): void }`
et une méthode `refresh()`.

- Planche murale (bois) + deux équerres ; jusqu'à `SHELF_MAX = 12` tranches debout, serrées, couleur de la
  pochette (`spineColors` + teinte de l'image, comme la vue Étagère), hauteurs légèrement variables
  (déterministes, d'après l'identifiant) ; une pochette posée en biais devant les tranches.
- Pochette en biais : la playlist dont l'identifiant vaut `current()` si elle fait partie des 12, sinon la
  première ; elle n'est pas répétée en tranche.
- Chargement : planche seule, puis tranches d'un coup ; la liste est gardée en mémoire tant que l'app
  tourne (rechargée au changement de source, `refresh()`).
- Clic (planche, tranches ou pochette) ou Entrée/Espace : `open()`. Survol : léger éclaircissement,
  infobulle « Ouvrir l'étagère » ; un seul élément focusable (`role="button"`, libellé « Ouvrir l'étagère des
  playlists »).
- Erreur de `load()` ou liste vide : planche vide, cliquable, aucun message.

Branchement dans la platine :
- `load` : `api.Library(source, "playlists", 0)` (source = source active, comme la vue Étagère) ;
- `current` : nouvelle méthode du service `CurrentPlaylist(source) (string, error)` ;
- `open` : forcer le mode étagère (`vinyl.shelf` = `side`), oublier la position mémorisée de
  `source:playlists` (nouvelle fonction `forget(key)` de `views/shelfcache.ts`), puis `go("bibliotheque", "playlists")`.

## Playlist en cours (`VinylService.CurrentPlaylist`)

Renvoie l'identifiant au format des éléments de la bibliothèque, "" si inconnu :

- Apple Music (Mac) : `SaveContext` → `"playlist:" + PlaylistID` (vide si aucune playlist en cours).
- Spotify : API web, `GET /me/player` → `context.uri` si `context.type == "playlist"` ; non connecté ou erreur
  → "". (Le pilotage AppleScript de Spotify ne connaît pas la playlist.)
- Autres cas (Windows Apple Music, aucun lecteur) : "".

## Erreurs

- Bibliothèque indisponible, Spotify non connecté ou refus (403, Premium…) : étagère vide, cliquable ; la
  vue Étagère affiche déjà le message adapté.
- `CurrentPlaylist` en erreur : pochette en biais = première playlist.

## Tests

- `layoutScene` : table toujours pleine largeur ; platine et pile reposent sur la ligne de la table ;
  l'étagère ne chevauche ni la colonne de commandes ni la pile ; petites et grandes fenêtres.
- `wallshelf` : 12 tranches au plus ; pochette en biais = playlist en cours si présente, sinon première, jamais
  répétée en tranche ; clic et clavier appellent `open` ; liste vide et erreur → planche vide cliquable.
- `open` de la platine : mode étagère forcé, position oubliée, route `bibliotheque/playlists`.
- Service : `CurrentPlaylist` Apple Music (format `playlist:…`, vide sans playlist) ; Spotify (contexte
  playlist, autre contexte, non connecté) avec un faux serveur.
- Contrôle visuel : faux backend (`.superpowers/mock/shoot.sh`), 1440×900, 1920×1080, 1100×700.

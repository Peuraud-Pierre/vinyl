# Vinyl Widget (widget de bureau macOS) — Spec de conception

Date : 2026-10-14
Statut : en relecture

## 1. Objectif

Afficher sur le bureau macOS un **vrai widget** (WidgetKit) qui montre et pilote le morceau en cours dans **Apple Music**, dans le style de la platine Vinyl.

- **Usage** : personnel, sur le Mac de l'utilisateur. Pas d'App Store.
- **Signature** : compte Apple **gratuit** (Personal Team). La signature expire au bout d'environ 7 jours. Il faut alors recompiler depuis Xcode.
- **Source** : Apple Music uniquement. Spotify est hors périmètre.
- **Indépendance** : le widget fonctionne sans que l'app Vinyl (Go/Wails) soit lancée, et même quand Musique est fermée.
- **L'app Vinyl existante n'est pas modifiée.**

### Critères de réussite

1. Le widget existe en trois tailles (petit, moyen, grand) dans la galerie de widgets de macOS.
2. Quand un morceau démarre ou change dans Musique, le widget se met à jour en quelques secondes. Le délai exact dépend de macOS.
3. Pendant la lecture, le bras avance avec la position du morceau sans intervention de l'assistant.
4. Les boutons ⏮ ⏯ ⏭ du widget pilotent Musique. Si Musique est fermée, ⏯ la lance et reprend la lecture.
5. Musique fermée : le widget montre le dernier morceau joué. L'assistant ne lance **jamais** Musique de lui-même, il le fait seulement sur un appui de bouton.
6. Après un redémarrage du Mac, le widget fonctionne sans rien ouvrir.

### Limites acceptées

- Le disque **ne tourne pas** : un widget n'affiche que des images figées.
- macOS décide du moment exact des rafraîchissements, qui ne sont pas instantanés.
- Une seule texture de disque : noir classique.

## 2. Architecture

Nouveau dossier `widget/` dans le dépôt, indépendant du code Go :

```
widget/
├── project.yml                     description XcodeGen (source de vérité du projet)
├── VinylWidget.xcodeproj           généré par `xcodegen generate` (non versionné)
├── Shared/                         compilé dans les deux cibles
│   ├── NowPlaying.swift            modèle d'état + encodage JSON
│   ├── StateStore.swift            lecture/écriture dans le conteneur App Group
│   ├── ArmMath.swift               angle du bras, progression
│   ├── Commands.swift              noms des signaux ⏮ ⏯ ⏭ (notifications Darwin)
│   └── AppGroup.swift              identifiant du groupe, chemins
├── App/                            cible « Vinyl Widget » (assistant)
│   ├── VinylWidgetApp.swift        point d'entrée, sans fenêtre (LSUIElement)
│   ├── MusicBridge.swift           AppleScript vers Musique (lecture d'état, commandes)
│   ├── MusicParser.swift           analyse des réponses AppleScript
│   ├── NowPlayingMonitor.swift     écoute de Musique, mise à jour de l'état
│   ├── CommandListener.swift       réception des signaux du widget
│   ├── AccentReader.swift          couleur d'accent depuis les réglages de Vinyl
│   └── LoginItem.swift             démarrage à l'ouverture de session (SMAppService)
├── Extension/                      cible « Vinyl Widget Extension »
│   ├── VinylWidget.swift           déclaration du widget (3 tailles)
│   ├── Provider.swift              TimelineProvider : génération des entrées
│   ├── Intents.swift               AppIntents PlayPause / Next / Previous
│   └── Views/                      Disc, Tonearm, SmallView, MediumView, LargeView
└── Tests/                          tests unitaires (logique pure)
```

### Identifiants

| Élément | Valeur |
|---|---|
| App | `com.peuraud.vinylwidget` |
| Extension | `com.peuraud.vinylwidget.extension` |
| App Group | `$(TeamIdentifierPrefix)com.peuraud.vinylwidget` (forme macOS préfixée par l'équipe) |
| macOS minimum | 14.0 (widgets interactifs) |

### Cible 1 — l'assistant « Vinyl Widget »

- **Forme** : app sans fenêtre ni icône dans le Dock (`LSUIElement = YES`), non sandboxée, Hardened Runtime avec l'entitlement `com.apple.security.automation.apple-events` et un `NSAppleEventsUsageDescription` en français.
- **Démarrage** : au premier lancement, `SMAppService.mainApp.register()` l'inscrit pour démarrer à chaque ouverture de session.
- **Écoute de Musique** :
  - Elle s'abonne à la notification distribuée `com.apple.Music.playerInfo`, que Musique émet à chaque changement de morceau ou d'état.
  - Elle s'abonne aussi au lancement et à la fermeture de Musique (`NSWorkspace`, identifiant `com.apple.Music`).
  - Pendant la lecture, elle relit la position toutes les 10 s pour détecter un déplacement manuel dans le morceau. Elle ne demande un rafraîchissement que si l'écart dépasse 3 s.
- **Lecture d'état** : elle vérifie d'abord avec `NSRunningApplication` que Musique tourne. Sinon, elle ne fait **aucun** appel AppleScript, car `tell application "Music"` lancerait l'app. Elle lit ensuite titre, artiste, album, durée, position, état, identifiant persistant du morceau et pochette (`raw data of artwork 1`).
- **Pochette** : réduite à 600 px de côté au maximum, puis écrite de façon atomique dans le conteneur partagé (`artwork-<id>.jpg`). Les anciennes pochettes sont supprimées.
- **État** : écrit de façon atomique dans `state.json` du conteneur, puis `WidgetCenter.shared.reloadAllTimelines()`. Le rafraîchissement n'est demandé que sur un vrai changement.
- **Commandes** : elle écoute les notifications Darwin `com.peuraud.vinylwidget.playpause`, `.next` et `.previous`.
  - ⏯ : `playpause` si Musique tourne, sinon lancement de Musique puis `play`.
  - ⏮ / ⏭ : sans effet si Musique est fermée.
- **Couleur d'accent** : lue dans `~/Library/Application Support/Vinyl/settings.json`, champ `accent` au format `#rrggbb`. À défaut, `#f5c542`, le défaut de Vinyl. Elle est relue à chaque mise à jour de l'état.
- **Appels AppleScript** : exécutés hors du fil principal, abandonnés au bout de 2 s, jamais empilés. Un seul appel est en cours à la fois.

### Cible 2 — l'extension « Vinyl Widget Extension »

- **Forme** : sandboxée, avec accès au même App Group en lecture.
- **Provider** :
  - Il lit `state.json` et la pochette.
  - En lecture, il produit une entrée toutes les 15 s, de maintenant jusqu'à la fin du morceau. La position de chaque entrée est calculée à partir de la position relevée et de son horodatage. Politique : `.atEnd`.
  - En pause, à l'arrêt ou avec Musique fermée, il produit une seule entrée. Politique : `.never`, car l'assistant demandera le rafraîchissement.
- **Boutons** : trois `AppIntent` (PlayPause, Next, Previous). Chacun publie la notification Darwin correspondante. L'extension ne parle jamais à Musique.

### Modèle partagé

```swift
struct NowPlaying: Codable, Equatable {
    enum PlayerState: String, Codable { case playing, paused, stopped }
    enum Availability: String, Codable { case running, closed, permissionDenied, neverPlayed }

    var availability: Availability
    var state: PlayerState
    var trackID: String?          // identifiant persistant Musique
    var title: String?
    var artist: String?
    var album: String?
    var duration: TimeInterval    // secondes
    var position: TimeInterval    // secondes, au moment `capturedAt`
    var capturedAt: Date
    var artworkFile: String?      // nom du fichier dans le conteneur
    var accentHex: String         // "#rrggbb"
}
```

### Flux

1. Musique change de morceau → notification `playerInfo` → l'assistant lit l'état → écrit `state.json` et la pochette → recharge les timelines.
2. macOS demande des entrées au Provider → lecture de `state.json` → entrées toutes les 15 s.
3. Appui sur ⏯ dans le widget → AppIntent → notification Darwin → l'assistant pilote Musique → Musique émet `playerInfo` → retour à l'étape 1.

## 3. Apparence

Style repris de Vinyl : fond sombre, disque noir avec sillons et reflet, pochette en étiquette centrale, bras à droite.

### Bras (mêmes valeurs que `frontend/src/turntable/arm.ts`)

- Levé (pause, arrêt, Musique fermée) : 4°.
- En lecture : `18 + (34 − 18) × position / durée`, avec la progression bornée entre 0 et 1. Si la durée est nulle ou absente : 18°.

### Tailles

| Taille | Contenu |
|---|---|
| Petit | Vinyle plein cadre avec le bras. La pochette est un bouton ⏯. |
| Moyen | Vinyle à gauche. À droite : titre, artiste, barre de progression, ⏮ ⏯ ⏭. |
| Grand | Grand vinyle en haut. En dessous : titre, artiste, barre de progression, ⏮ ⏯ ⏭. |

- **Clic hors des boutons** : ouvre Musique (`widgetURL` vers `music://`).
- **Couleur d'accent** : barre de progression et contrepoids du bras.
- **Mode teinté de macOS** : la pochette et le disque suivent le rendu imposé par le système. Les boutons et la barre sont marqués `widgetAccentable`.

### États

| État | Rendu |
|---|---|
| En lecture | Bras posé qui avance. |
| En pause | Bras levé. |
| Musique fermée | Dernier morceau, bras levé, rendu légèrement estompé, mention « Musique fermée ». ⏯ relance. |
| Jamais rien joué | Disque vide, message « Lance un morceau dans Musique ». |
| Autorisation refusée | Message « Autorise Vinyl Widget à piloter Musique ». Le clic ouvre `x-apple.systempreferences:com.apple.preference.security?Privacy_Automation`. |
| Pas de pochette | Étiquette générique couleur d'accent avec une note de musique (SF Symbol `music.note`). |

## 4. Gestion des erreurs

| Situation | Comportement |
|---|---|
| Autorisation d'automatisation refusée (erreur AppleScript −1743) | `availability = permissionDenied`, état affiché par le widget (voir §3). Nouvel essai à chaque notification de Musique. |
| Musique se ferme | Notification `NSWorkspace` → `availability = closed`, avec conservation du dernier morceau et de sa pochette. |
| AppleScript lent (> 2 s) ou en erreur | Appel abandonné, état précédent conservé, nouvel essai au prochain événement. |
| `state.json` absent ou illisible côté widget | Affichage « Jamais rien joué ». |
| Pochette illisible | Étiquette générique. |
| Assistant arrêté | Le widget garde le dernier état et les boutons sont sans effet. Il revient à la prochaine ouverture de session. |
| Signature gratuite expirée | Documenté dans le README : recompiler depuis Xcode (⌘R). |

## 5. Tests

- **Tests unitaires** (cible de tests, Swift Testing), sans Musique ni WidgetKit :
  - `MusicParser` : réponses AppleScript normales, champs vides, durée absente, caractères spéciaux.
  - `ArmMath` : angle levé, début, fin, bornage, durée nulle.
  - Génération des entrées du Provider : nombre, positions, fin de morceau, état non joué. La logique est isolée dans une fonction pure.
  - `StateStore` : aller-retour JSON, écriture atomique, fichier absent.
  - `AccentReader` : fichier absent, couleur invalide, couleur valide.
- **Compilation** : `xcodebuild -scheme "Vinyl Widget" build test` en ligne de commande.
- **Recette manuelle** (liste dans le README) :
  - ajout des trois tailles ;
  - lecture, pause, suivant et précédent depuis le widget ;
  - changement de morceau dans Musique ;
  - déplacement dans le morceau ;
  - fermeture de Musique, puis ⏯ depuis le widget ;
  - refus puis accord de l'autorisation ;
  - redémarrage du Mac.

## 6. Installation (README `widget/README.md`)

1. `brew install xcodegen`, puis `cd widget && xcodegen generate`.
2. Xcode → Réglages → Comptes : ajouter l'identifiant Apple. Dans le projet, choisir l'équipe « Personal Team » pour les deux cibles.
3. Lancer la cible « Vinyl Widget » (⌘R) et accepter l'accès à Musique.
4. Clic droit sur le bureau → Modifier les widgets → Vinyl → choisir une taille.
5. Tous les ~7 jours (expiration de la signature gratuite) : relancer ⌘R depuis Xcode.

`widget/VinylWidget.xcodeproj` est ajouté au `.gitignore` : seul `project.yml` est versionné.

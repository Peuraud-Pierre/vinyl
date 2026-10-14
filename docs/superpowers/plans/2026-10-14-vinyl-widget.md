# Vinyl Widget — Plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un vrai widget de bureau macOS (WidgetKit, 3 tailles) qui affiche et pilote Apple Music sans que Vinyl soit lancé, plus une icône ☁️↓ dans Vinyl pour le recompiler et le réinstaller, et un démarrage de Vinyl sans le disque flottant.

**Architecture:** Un projet Xcode généré par XcodeGen dans `widget/` contient l'assistant « Vinyl Widget » (app sans fenêtre, non sandboxée, qui écoute Musique via AppleScript et écrit l'état dans un conteneur App Group) et l'extension WidgetKit (sandboxée, qui lit cet état et renvoie les commandes ⏮ ⏯ ⏭ par notifications Darwin). Côté Vinyl (Go/Wails), un package `internal/widgetupdate` détecte le besoin de mise à jour, puis recompile et réinstalle l'assistant. Le frontend affiche l'icône.

**Tech Stack:** Swift 5 (mode langage), SwiftUI, WidgetKit, AppIntents, ServiceManagement, Swift Testing, XcodeGen, Xcode 27 ; Go 1.26, Wails v3 beta.29 ; TypeScript, Vitest.

**Spec:** `docs/superpowers/specs/2026-10-14-vinyl-widget-design.md`

## Global Constraints

- **macOS** : macOS minimum 14.0 pour les deux cibles Swift. `SWIFT_VERSION = 5.0`.
- **Identifiants** :
  - app `com.peuraud.vinylwidget` ;
  - extension `com.peuraud.vinylwidget.extension` ;
  - App Group `$(TeamIdentifierPrefix)com.peuraud.vinylwidget`, lu à l'exécution via la clé Info.plist `VinylAppGroup`.
- **Noms des produits** :
  - app : `Vinyl Widget` (installée dans `/Applications/Vinyl Widget.app`) ;
  - widget : nom affiché « Vinyl », `kind` `VinylWidget`.
- **Notifications Darwin** : `com.peuraud.vinylwidget.playpause`, `com.peuraud.vinylwidget.next`, `com.peuraud.vinylwidget.previous`.
- **Schéma d'URL de l'assistant** : `vinylwidget://music` (ouvre Musique), `vinylwidget://automation` (ouvre Réglages → Automatisation).
- **Bras** : levé 4°, début 18°, fin 34°, `18 + (34 − 18) × position / durée` borné. Durée nulle → 18°.
- **Timeline** :
  - une entrée toutes les 15 s, 80 entrées au maximum ;
  - en lecture, politique `.atEnd` ; sinon, une seule entrée et `.never`.
- **Assistant** :
  - relit la position toutes les 10 s, seulement en lecture ;
  - publie si l'écart dépasse 3 s ;
  - délai AppleScript de 2 s (15 s seulement pour ⏯ quand Musique est fermée) ;
  - n'appelle jamais AppleScript si Musique n'est pas lancée.
- **Pochette** : JPEG de 600 px de côté au maximum, écrite de façon atomique, seule celle du morceau courant est conservée.
- **Accent** : `accent` de `~/Library/Application Support/Vinyl/settings.json` si `^#[0-9a-fA-F]{6}$`, sinon `#f5c542`.
- **Mise à jour depuis Vinyl** :
  - vérification au démarrage puis toutes les 6 h ;
  - seuil d'expiration de 48 h ;
  - délai total de 10 min ;
  - empreinte enregistrée dans `~/Library/Application Support/Vinyl/widget-install.json` ;
  - build dans `~/Library/Caches/Vinyl/widget-build` ;
  - outils par chemin absolu : `/opt/homebrew/bin/xcodegen`, `/usr/local/bin/xcodegen`, `/usr/bin/xcodebuild`, `/usr/bin/git`, `/usr/bin/security`, `/usr/bin/ditto`, `/usr/bin/killall`, `/usr/bin/open`.
- **Vinyl au démarrage** : fenêtre principale visible, disque flottant masqué, menu « Fenêtre → Mini-platine » (⌘⇧M).
- **Textes** : tous les textes visibles sont en français.
- **Commandes Go** : `CGO_ENABLED=1 go test . ./internal/...` (jamais `./...`).
- **Commits** : style du dépôt (`feat(widget): …`, en français). Chaque message se termine par `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- **Étapes « 👤 Pierre »** : elles demandent son compte Apple ou son bureau. L'agent les signale et ne les simule pas.

## Review Focus

1. **Morceau sans durée** (radio, flux) : le bras reste à 18° et le widget ne boucle pas sur des rafraîchissements. Test dans la Task 3, « durée nulle ».
2. **Titres avec retour à la ligne, emoji, champs vides ou `missing value`** : l'analyse ne casse pas et ne décale pas les champs. Tests dans la Task 4.
3. **Chemin du dépôt avec des espaces** (`/Users/pierre/Document on Mac/…`) : la mise à jour et le chemin inscrit à la compilation fonctionnent. Test dans la Task 9 (chemin avec espace) et vérification dans la Task 10.
4. **Plusieurs certificats « Apple Development », dont un expiré** : l'équipe choisie est celle du certificat valide. Test dans la Task 8.
5. **Musique fermée pendant la lecture, ou morceau terminé sans nouvelle notification** : le widget passe à « Musique fermée » sans relancer Musique, et une timeline dépassée ne génère qu'une entrée. Tests dans la Task 2 (`closed`) et la Task 3 (« lecture au-delà de la fin »).

---

## Structure des fichiers

```
main.go                                   Task 1 (fenêtres + menu), Task 10 (branchement widgetupdate)
build/darwin/Taskfile.yml                 Task 10 (ldflags main.projectDir)
internal/settings/settings.go             Task 10 (widgetProjectDir)
internal/app/service.go                   Task 10 (WidgetUpdateStatus, UpdateWidget)
internal/widgetupdate/
  detect.go        Detect, Message, ParseExpiry, TeamFromPEM           Task 8
  updater.go       Runner, ExecRunner, Updater, UserError, ErrorMessage Task 9
  manager.go       Status, Manager                                       Task 10
frontend/index.html                       Task 11 (emplacement de l'icône)
frontend/src/widgetupdate.ts              Task 11 (rendu pur de l'icône)
frontend/src/api.ts, main.ts              Task 11
widget/
  project.yml                             Tasks 2, 5, 6
  README.md                               Task 7
  Shared/   AppGroup, NowPlaying, StateStore, Command (T2) ; ArmMath, TimelineBuilder, StateDiff (T3)
  App/      MusicParser, AccentReader (T4) ; MusicBridge, Artwork, NowPlayingMonitor, CommandListener,
            LoginItem, URLRoute, VinylWidgetApp (T5)
  Extension/ VinylWidget, Provider, Intents, Views/* (T6)
  Tests/    NowPlayingTests, StateStoreTests, CommandTests (T2) ; ArmMathTests, TimelineBuilderTests,
            StateDiffTests (T3) ; MusicParserTests, AccentReaderTests (T4)
.gitignore, README.md                     Task 7
```

**Commande de test Swift (utilisée partout) :**

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl/widget" && xcodegen generate && \
xcodebuild test -project VinylWidget.xcodeproj -scheme VinylWidgetTests -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO 2>&1 | tail -30
```

**Prérequis outil (une fois) :** `brew install xcodegen`.

---

### Task 1: Vinyl démarre sans le disque flottant

**Files:**
- Modify: `main.go` (options de la fenêtre `widget`, fenêtre `main`, menu)

**Interfaces:**
- Consumes: rien.
- Produces: rien d'utilisé par les autres tâches.

- [ ] **Step 1: Masquer le disque flottant au lancement**

Dans `main.go`, ajouter `Hidden: true` aux options du widget :

```go
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
```

- [ ] **Step 2: Afficher la fenêtre principale au lancement**

Dans les options de `mainWin`, supprimer la ligne `Hidden:           true,`.

- [ ] **Step 3: Ajouter le menu « Fenêtre → Mini-platine »**

Juste après le hook `ApplicationShouldHandleReopen`, ajouter :

```go
	// Menu de l'app : ⌘⇧M affiche ou masque le disque flottant.
	menu := application.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.EditMenu)
	fenetre := menu.AddSubmenu("Fenêtre")
	fenetre.Add("Mini-platine").SetAccelerator("CmdOrCtrl+Shift+M").OnClick(func(*application.Context) {
		if widget.IsVisible() {
			widget.Hide()
		} else {
			widget.Show()
		}
	})
	wailsApp.Menu.Set(menu)
```

- [ ] **Step 4: Compiler**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go vet . && CGO_ENABLED=1 go test . ./internal/...`
Expected: aucune erreur, tous les tests PASS.

- [ ] **Step 5: Vérifier dans l'app**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 wails3 dev`
Expected :
- au lancement, la fenêtre principale s'ouvre et aucun disque flottant n'apparaît ;
- ⌘⇧M affiche le disque, puis un nouveau ⌘⇧M le masque ;
- le menu « Édition » permet toujours copier/coller dans les champs de Réglages.

Arrêter avec ⌘Q.

- [ ] **Step 6: Commit**

```bash
git add main.go
git commit -m "feat(fenêtres): Vinyl s'ouvre sur la fenêtre principale, mini-platine via ⌘⇧M

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Projet XcodeGen et modèle partagé

**Files:**
- Create: `widget/project.yml`
- Create: `widget/Shared/AppGroup.swift`, `widget/Shared/NowPlaying.swift`, `widget/Shared/StateStore.swift`, `widget/Shared/Command.swift`
- Create: `widget/Tests/NowPlayingTests.swift`, `widget/Tests/StateStoreTests.swift`, `widget/Tests/CommandTests.swift`
- Modify: `.gitignore`

**Interfaces:**
- Produces :
  - `struct NowPlaying: Codable, Equatable` :
    - champs `availability`, `state`, `trackID`, `title`, `artist`, `album`, `duration`, `position`, `capturedAt`, `artworkFile`, `accentHex` ;
    - `static let defaultAccent = "#f5c542"` ;
    - `static func neverPlayed(accentHex: String = defaultAccent, now: Date = Date()) -> NowPlaying` ;
    - `func position(at: Date) -> TimeInterval` ;
    - `func closed(accentHex: String, now: Date) -> NowPlaying`.
  - `struct StateStore` :
    - `init(directory: URL)` ;
    - `func load() -> NowPlaying?` ;
    - `func save(_: NowPlaying) throws` ;
    - `func artworkURL(_ name: String) -> URL` ;
    - `func saveArtwork(_ jpeg: Data, trackID: String) throws -> String`.
  - `enum Command: String, CaseIterable { playpause, next, previous }` :
    - `var notificationName: String` ;
    - `init?(notificationName:)` ;
    - `func post()`.
  - `enum AppGroup { static var identifier: String; static var containerURL: URL? }`.

- [ ] **Step 1: Installer XcodeGen et écrire `widget/project.yml` (cible de tests seule)**

Run: `brew list xcodegen >/dev/null 2>&1 || brew install xcodegen`

```yaml
name: VinylWidget
options:
  bundleIdPrefix: com.peuraud
  deploymentTarget:
    macOS: "14.0"
  createIntermediateGroups: true
settings:
  base:
    SWIFT_VERSION: "5.0"
    MACOSX_DEPLOYMENT_TARGET: "14.0"
    MARKETING_VERSION: "1.0"
    CURRENT_PROJECT_VERSION: "1"
    CODE_SIGN_STYLE: Automatic
    ENABLE_HARDENED_RUNTIME: YES
targets:
  VinylWidgetTests:
    type: bundle.unit-test
    platform: macOS
    sources:
      - path: Tests
      - path: Shared
    settings:
      base:
        PRODUCT_BUNDLE_IDENTIFIER: com.peuraud.vinylwidget.tests
        GENERATE_INFOPLIST_FILE: YES
schemes:
  VinylWidgetTests:
    build:
      targets:
        VinylWidgetTests: [test]
    test:
      targets: [VinylWidgetTests]
```

- [ ] **Step 2: Ignorer les fichiers générés**

Ajouter à la fin de `.gitignore` :

```
# Widget : projet et plists générés par XcodeGen
widget/*.xcodeproj
widget/App/Info.plist
widget/App/*.entitlements
widget/Extension/Info.plist
widget/Extension/*.entitlements
```

- [ ] **Step 3: Écrire les tests**

`widget/Tests/NowPlayingTests.swift` :

```swift
import Foundation
import Testing

struct NowPlayingTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func playing(position: TimeInterval = 10, duration: TimeInterval = 200) -> NowPlaying {
        NowPlaying(availability: .running, state: .playing, trackID: "A1", title: "Titre", artist: "Artiste",
                   album: "Album", duration: duration, position: position, capturedAt: t0,
                   artworkFile: "artwork-A1.jpg", accentHex: "#f5c542")
    }

    @Test func positionAvanceEnLecture() {
        #expect(playing().position(at: t0.addingTimeInterval(5)) == 15)
    }

    @Test func positionBorneeALaDuree() {
        #expect(playing(position: 198).position(at: t0.addingTimeInterval(10)) == 200)
    }

    @Test func positionFigeeEnPause() {
        var np = playing()
        np.state = .paused
        #expect(np.position(at: t0.addingTimeInterval(30)) == 10)
    }

    @Test func positionSansDureeNonBornee() {
        #expect(playing(position: 10, duration: 0).position(at: t0.addingTimeInterval(5)) == 15)
    }

    @Test func fermeGardeLeDernierMorceau() {
        let c = playing().closed(accentHex: "#112233", now: t0.addingTimeInterval(4))
        #expect(c.availability == .closed)
        #expect(c.state == .paused)
        #expect(c.title == "Titre")
        #expect(c.artworkFile == "artwork-A1.jpg")
        #expect(c.position == 14)
        #expect(c.capturedAt == t0.addingTimeInterval(4))
        #expect(c.accentHex == "#112233")
    }

    @Test func fermeSansMorceauDonneJamaisJoue() {
        let c = NowPlaying.neverPlayed(now: t0).closed(accentHex: "#112233", now: t0)
        #expect(c.availability == .neverPlayed)
        #expect(c.accentHex == "#112233")
    }
}
```

`widget/Tests/StateStoreTests.swift` :

```swift
import Foundation
import Testing

struct StateStoreTests {
    let dir = FileManager.default.temporaryDirectory
        .appendingPathComponent("vinylwidget-tests-\(UUID().uuidString)")

    @Test func fichierAbsent() {
        #expect(StateStore(directory: dir).load() == nil)
    }

    @Test func allerRetour() throws {
        let store = StateStore(directory: dir)
        let np = NowPlaying(availability: .running, state: .paused, trackID: "X", title: "Été 🎵", artist: "A",
                            album: "B", duration: 213.5, position: 12.25,
                            capturedAt: Date(timeIntervalSince1970: 1_234_567.891),
                            artworkFile: nil, accentHex: "#ff6b6b")
        try store.save(np)
        #expect(store.load() == np)
    }

    @Test func fichierIllisible() throws {
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        try Data("pas du json".utf8).write(to: dir.appendingPathComponent("state.json"))
        #expect(StateStore(directory: dir).load() == nil)
    }

    @Test func pochetteRemplaceLAncienne() throws {
        let store = StateStore(directory: dir)
        let a = try store.saveArtwork(Data([1]), trackID: "AAA")
        let b = try store.saveArtwork(Data([2]), trackID: "BBB")
        #expect(a == "artwork-AAA.jpg")
        #expect(b == "artwork-BBB.jpg")
        #expect(!FileManager.default.fileExists(atPath: store.artworkURL(a).path))
        #expect(try Data(contentsOf: store.artworkURL(b)) == Data([2]))
    }

    @Test func identifiantNettoye() throws {
        #expect(try StateStore(directory: dir).saveArtwork(Data([1]), trackID: "../x/../y") == "artwork-xy.jpg")
        #expect(try StateStore(directory: dir).saveArtwork(Data([1]), trackID: "") == "artwork-x.jpg")
    }
}
```

`widget/Tests/CommandTests.swift` :

```swift
import Testing

struct CommandTests {
    @Test func nomsDesNotifications() {
        #expect(Command.playpause.notificationName == "com.peuraud.vinylwidget.playpause")
        #expect(Command.next.notificationName == "com.peuraud.vinylwidget.next")
        #expect(Command.previous.notificationName == "com.peuraud.vinylwidget.previous")
    }

    @Test func allerRetour() {
        for c in Command.allCases {
            #expect(Command(notificationName: c.notificationName) == c)
        }
        #expect(Command(notificationName: "com.apple.Music.playerInfo") == nil)
        #expect(Command(notificationName: "com.peuraud.vinylwidget.stop") == nil)
    }
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils échouent**

Run : la commande de test Swift (en tête du plan).
Expected : échec de compilation (`cannot find 'NowPlaying' in scope`).

- [ ] **Step 5: Écrire le code partagé**

`widget/Shared/AppGroup.swift` :

```swift
import Foundation

/// Conteneur partagé entre l'assistant et l'extension.
enum AppGroup {
    /// Valeur de la clé Info.plist `VinylAppGroup` (`$(TeamIdentifierPrefix)com.peuraud.vinylwidget`).
    static var identifier: String {
        Bundle.main.object(forInfoDictionaryKey: "VinylAppGroup") as? String ?? ""
    }

    static var containerURL: URL? {
        guard !identifier.isEmpty else { return nil }
        return FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: identifier)
    }
}
```

`widget/Shared/NowPlaying.swift` :

```swift
import Foundation

/// État publié par l'assistant et affiché par le widget.
struct NowPlaying: Codable, Equatable {
    enum PlayerState: String, Codable { case playing, paused, stopped }
    enum Availability: String, Codable { case running, closed, permissionDenied, neverPlayed }

    var availability: Availability
    var state: PlayerState
    var trackID: String?          // identifiant persistant Musique
    var title: String?
    var artist: String?
    var album: String?
    var duration: TimeInterval    // secondes, 0 si inconnue
    var position: TimeInterval    // secondes, au moment `capturedAt`
    var capturedAt: Date
    var artworkFile: String?      // nom du fichier dans le conteneur
    var accentHex: String         // "#rrggbb"

    static let defaultAccent = "#f5c542"

    static func neverPlayed(accentHex: String = defaultAccent, now: Date = Date()) -> NowPlaying {
        NowPlaying(availability: .neverPlayed, state: .stopped, trackID: nil, title: nil, artist: nil, album: nil,
                   duration: 0, position: 0, capturedAt: now, artworkFile: nil, accentHex: accentHex)
    }

    /// Position estimée à `date` : elle n'avance qu'en lecture, bornée à [0, durée] si la durée est connue.
    func position(at date: Date) -> TimeInterval {
        var p = position
        if availability == .running && state == .playing {
            p += date.timeIntervalSince(capturedAt)
        }
        p = max(0, p)
        return duration > 0 ? min(p, duration) : p
    }

    /// État « Musique fermée » : garde le dernier morceau, ou « jamais rien joué » s'il n'y en a pas.
    func closed(accentHex: String, now: Date) -> NowPlaying {
        guard trackID != nil else { return .neverPlayed(accentHex: accentHex, now: now) }
        var n = self
        n.availability = .closed
        n.state = .paused
        n.position = position(at: now)
        n.capturedAt = now
        n.accentHex = accentHex
        return n
    }
}
```

`widget/Shared/StateStore.swift` :

```swift
import Foundation

/// Lecture/écriture de l'état et de la pochette dans le conteneur partagé.
struct StateStore {
    let directory: URL

    var stateURL: URL { directory.appendingPathComponent("state.json") }

    func load() -> NowPlaying? {
        guard let data = try? Data(contentsOf: stateURL) else { return nil }
        return try? JSONDecoder().decode(NowPlaying.self, from: data)
    }

    func save(_ np: NowPlaying) throws {
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        try JSONEncoder().encode(np).write(to: stateURL, options: .atomic)
    }

    func artworkURL(_ name: String) -> URL { directory.appendingPathComponent(name) }

    /// Écrit la pochette de façon atomique, supprime les autres et renvoie le nom du fichier.
    func saveArtwork(_ jpeg: Data, trackID: String) throws -> String {
        let safe = String(trackID.filter { $0.isLetter || $0.isNumber })
        let name = "artwork-\(safe.isEmpty ? "x" : safe).jpg"
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        try jpeg.write(to: artworkURL(name), options: .atomic)
        let files = (try? FileManager.default.contentsOfDirectory(atPath: directory.path)) ?? []
        for f in files where f.hasPrefix("artwork-") && f != name {
            try? FileManager.default.removeItem(at: artworkURL(f))
        }
        return name
    }
}
```

`widget/Shared/Command.swift` :

```swift
import Foundation

/// Ordres envoyés par le widget à l'assistant, par notification Darwin (autorisée dans le bac à sable).
enum Command: String, CaseIterable {
    case playpause, next, previous

    static let prefix = "com.peuraud.vinylwidget."

    var notificationName: String { Self.prefix + rawValue }

    init?(notificationName: String) {
        guard notificationName.hasPrefix(Self.prefix) else { return nil }
        self.init(rawValue: String(notificationName.dropFirst(Self.prefix.count)))
    }

    func post() {
        CFNotificationCenterPostNotification(CFNotificationCenterGetDarwinNotifyCenter(),
                                             CFNotificationName(notificationName as CFString), nil, nil, true)
    }
}
```

- [ ] **Step 6: Lancer les tests pour vérifier qu'ils passent**

Run : la commande de test Swift.
Expected : `** TEST SUCCEEDED **`, 13 tests.

- [ ] **Step 7: Commit**

```bash
git add .gitignore widget/project.yml widget/Shared widget/Tests
git commit -m "feat(widget): projet XcodeGen, état partagé, stockage et commandes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Bras, timeline et détection des changements (logique pure)

**Files:**
- Create: `widget/Shared/ArmMath.swift`, `widget/Shared/TimelineBuilder.swift`, `widget/Shared/StateDiff.swift`
- Create: `widget/Tests/ArmMathTests.swift`, `widget/Tests/TimelineBuilderTests.swift`, `widget/Tests/StateDiffTests.swift`

**Interfaces:**
- Consumes : `NowPlaying` (Task 2).
- Produces :
  - `enum ArmMath` :
    - `rest = 4.0`, `start = 18.0`, `end = 34.0` ;
    - `static func progress(position:duration:) -> Double` ;
    - `static func angle(isPlaying:position:duration:) -> Double`.
  - `struct VinylEntry` :
    - champs `date: Date`, `nowPlaying: NowPlaying`, `position: TimeInterval` ;
    - propriétés `isPlaying`, `armAngle`, `progress`.
  - `enum TimelineBuilder` :
    - `step = 15`, `maxEntries = 80` ;
    - `static func entries(for: NowPlaying, now: Date) -> (entries: [VinylEntry], refreshAtEnd: Bool)`.
  - `enum StateDiff` :
    - `driftTolerance = 3` ;
    - `static func shouldPublish(old: NowPlaying, new: NowPlaying) -> Bool`.

- [ ] **Step 1: Écrire les tests**

`widget/Tests/ArmMathTests.swift` :

```swift
import Testing

struct ArmMathTests {
    @Test func debutMilieuFin() {
        #expect(ArmMath.angle(isPlaying: true, position: 0, duration: 200) == 18)
        #expect(ArmMath.angle(isPlaying: true, position: 100, duration: 200) == 26)
        #expect(ArmMath.angle(isPlaying: true, position: 200, duration: 200) == 34)
    }

    @Test func leveHorsLecture() {
        #expect(ArmMath.angle(isPlaying: false, position: 100, duration: 200) == 4)
    }

    @Test func bornes() {
        #expect(ArmMath.angle(isPlaying: true, position: -5, duration: 200) == 18)
        #expect(ArmMath.angle(isPlaying: true, position: 500, duration: 200) == 34)
    }

    @Test func dureeNulleOuInvalide() {
        #expect(ArmMath.angle(isPlaying: true, position: 50, duration: 0) == 18)
        #expect(ArmMath.progress(position: .nan, duration: 200) == 0)
        #expect(ArmMath.progress(position: 50, duration: .nan) == 0)
    }
}
```

`widget/Tests/TimelineBuilderTests.swift` :

```swift
import Foundation
import Testing

struct TimelineBuilderTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func np(_ state: NowPlaying.PlayerState = .playing, availability: NowPlaying.Availability = .running,
            position: TimeInterval = 0, duration: TimeInterval = 60) -> NowPlaying {
        NowPlaying(availability: availability, state: state, trackID: "A", title: "T", artist: "A", album: "B",
                   duration: duration, position: position, capturedAt: t0, artworkFile: nil, accentHex: "#f5c542")
    }

    @Test func lectureUneEntreeToutesLes15s() {
        let r = TimelineBuilder.entries(for: np(), now: t0)
        #expect(r.entries.map(\.position) == [0, 15, 30, 45, 60])
        #expect(r.entries.map(\.date) == [0, 15, 30, 45, 60].map { t0.addingTimeInterval($0) })
        #expect(r.refreshAtEnd)
    }

    @Test func positionCalculeeDepuisLaCapture() {
        let r = TimelineBuilder.entries(for: np(position: 10), now: t0.addingTimeInterval(5))
        #expect(r.entries.first?.position == 15)
    }

    @Test func pauseUneSeuleEntree() {
        let r = TimelineBuilder.entries(for: np(.paused, position: 20), now: t0)
        #expect(r.entries.count == 1)
        #expect(r.entries[0].armAngle == ArmMath.rest)
        #expect(!r.refreshAtEnd)
    }

    @Test func musiqueFermeeUneSeuleEntree() {
        let r = TimelineBuilder.entries(for: np(availability: .closed), now: t0)
        #expect(r.entries.count == 1)
        #expect(!r.refreshAtEnd)
    }

    @Test func dureeNulle() {
        let r = TimelineBuilder.entries(for: np(duration: 0), now: t0)
        #expect(r.entries.count == 1)
        #expect(r.entries[0].armAngle == ArmMath.start)
        #expect(!r.refreshAtEnd)
    }

    @Test func lectureAuDelaDeLaFin() {
        let r = TimelineBuilder.entries(for: np(position: 50), now: t0.addingTimeInterval(120))
        #expect(r.entries.count == 1)
        #expect(!r.refreshAtEnd)
    }

    @Test func morceauLongPlafonne() {
        let r = TimelineBuilder.entries(for: np(duration: 3600), now: t0)
        #expect(r.entries.count == TimelineBuilder.maxEntries)
        #expect(r.refreshAtEnd)
    }
}
```

`widget/Tests/StateDiffTests.swift` :

```swift
import Foundation
import Testing

struct StateDiffTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func np(_ state: NowPlaying.PlayerState = .playing, track: String = "A", position: TimeInterval,
            at offset: TimeInterval = 0, accent: String = "#f5c542") -> NowPlaying {
        NowPlaying(availability: .running, state: state, trackID: track, title: "T", artist: "A", album: "B",
                   duration: 200, position: position, capturedAt: t0.addingTimeInterval(offset),
                   artworkFile: nil, accentHex: accent)
    }

    @Test func lectureNormaleNePublieRien() {
        #expect(!StateDiff.shouldPublish(old: np(position: 10), new: np(position: 15, at: 5)))
        #expect(!StateDiff.shouldPublish(old: np(position: 10), new: np(position: 17.5, at: 5)))
    }

    @Test func deplacementDansLeMorceau() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(position: 60, at: 5)))
    }

    @Test func changementDeMorceau() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(track: "B", position: 0, at: 5)))
    }

    @Test func pauseVersLecture() {
        #expect(StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(position: 10, at: 5)))
    }

    @Test func couleurDAccent() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(position: 10, accent: "#112233")))
    }

    @Test func deplacementEnPause() {
        #expect(!StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(.paused, position: 10, at: 30)))
        #expect(StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(.paused, position: 50, at: 30)))
    }
}
```

- [ ] **Step 2: Lancer les tests pour vérifier qu'ils échouent**

Run : la commande de test Swift.
Expected : échec de compilation (`cannot find 'ArmMath' in scope`).

- [ ] **Step 3: Écrire le code**

`widget/Shared/ArmMath.swift` :

```swift
import Foundation

/// Angles du bras, identiques à frontend/src/turntable/arm.ts.
enum ArmMath {
    static let rest = 4.0   // bras levé
    static let start = 18.0 // aiguille au bord du disque (début du morceau)
    static let end = 34.0   // aiguille près de l'étiquette (fin du morceau)

    static func progress(position: TimeInterval, duration: TimeInterval) -> Double {
        guard duration > 0, duration.isFinite, position.isFinite else { return 0 }
        return min(1, max(0, position / duration))
    }

    static func angle(isPlaying: Bool, position: TimeInterval, duration: TimeInterval) -> Double {
        guard isPlaying else { return rest }
        return start + (end - start) * progress(position: position, duration: duration)
    }
}
```

`widget/Shared/TimelineBuilder.swift` :

```swift
import Foundation

/// Une image du widget à un instant donné.
struct VinylEntry {
    let date: Date
    let nowPlaying: NowPlaying
    let position: TimeInterval

    var isPlaying: Bool { nowPlaying.availability == .running && nowPlaying.state == .playing }
    var armAngle: Double { ArmMath.angle(isPlaying: isPlaying, position: position, duration: nowPlaying.duration) }
    var progress: Double { ArmMath.progress(position: position, duration: nowPlaying.duration) }
}

enum TimelineBuilder {
    static let step: TimeInterval = 15
    static let maxEntries = 80

    /// En lecture : une entrée toutes les 15 s jusqu'à la fin du morceau (refreshAtEnd = true).
    /// Sinon : une seule entrée, et c'est l'assistant qui demandera le prochain rafraîchissement.
    static func entries(for np: NowPlaying, now: Date) -> (entries: [VinylEntry], refreshAtEnd: Bool) {
        let first = VinylEntry(date: now, nowPlaying: np, position: np.position(at: now))
        guard first.isPlaying, np.duration > 0, first.position < np.duration else { return ([first], false) }
        var list = [first]
        var t = now.addingTimeInterval(step)
        while list.count < maxEntries {
            let p = np.position(at: t)
            list.append(VinylEntry(date: t, nowPlaying: np, position: p))
            if p >= np.duration { break }
            t = t.addingTimeInterval(step)
        }
        return (list, true)
    }
}
```

`widget/Shared/StateDiff.swift` :

```swift
import Foundation

/// Décide si un nouvel état mérite un rafraîchissement du widget (budget de rechargements limité).
enum StateDiff {
    static let driftTolerance: TimeInterval = 3

    static func shouldPublish(old: NowPlaying, new: NowPlaying) -> Bool {
        var a = old, b = new
        a.position = 0; a.capturedAt = .distantPast
        b.position = 0; b.capturedAt = .distantPast
        if a != b { return true }
        return abs(old.position(at: new.capturedAt) - new.position) > driftTolerance
    }
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils passent**

Run : la commande de test Swift.
Expected : `** TEST SUCCEEDED **`.

- [ ] **Step 5: Commit**

```bash
git add widget/Shared widget/Tests
git commit -m "feat(widget): angle du bras, timeline et détection des changements

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Lecture des réponses de Musique et de la couleur d'accent

**Files:**
- Create: `widget/App/MusicParser.swift`, `widget/App/AccentReader.swift`
- Create: `widget/Tests/MusicParserTests.swift`, `widget/Tests/AccentReaderTests.swift`
- Modify: `widget/project.yml` (sources de la cible de tests)

**Interfaces:**
- Consumes : `NowPlaying` (Task 2).
- Produces :
  - `struct TrackInfo: Equatable` : `state`, `trackID`, `title`, `artist`, `album`, `duration`, `position`.
  - `enum MusicReading: Equatable { case closed, stopped, track(TrackInfo) }`.
  - `enum MusicParser` :
    - `static let separator = "\u{1F}"` ;
    - `static func parse(_ output: String) -> MusicReading?` ;
    - `static func seconds(_ s: String) -> TimeInterval`.
  - `enum AccentReader` :
    - `static var vinylSettingsURL: URL` ;
    - `static func read(from url: URL = vinylSettingsURL) -> String`.

- [ ] **Step 1: Ajouter les fichiers à la cible de tests**

Dans `widget/project.yml`, cible `VinylWidgetTests`, remplacer `sources` par :

```yaml
    sources:
      - path: Tests
      - path: Shared
      - path: App/MusicParser.swift
      - path: App/AccentReader.swift
```

- [ ] **Step 2: Écrire les tests**

`widget/Tests/MusicParserTests.swift` :

```swift
import Testing

struct MusicParserTests {
    let us = MusicParser.separator

    @Test func fermeeEtArretee() {
        #expect(MusicParser.parse("closed\n") == .closed)
        #expect(MusicParser.parse("stopped") == .stopped)
    }

    @Test func morceauAvecVirgulesDecimales() {
        let out = ["playing", "ABC123", "Titre", "Artiste", "Album", "213,5", "12,25"].joined(separator: us) + "\n"
        #expect(MusicParser.parse(out) == .track(TrackInfo(state: .playing, trackID: "ABC123", title: "Titre",
                                                           artist: "Artiste", album: "Album",
                                                           duration: 213.5, position: 12.25)))
    }

    @Test func champsVidesEtMissingValue() {
        let out = ["paused", "ID", "", "", "", "missing value", "0"].joined(separator: us)
        #expect(MusicParser.parse(out) == .track(TrackInfo(state: .paused, trackID: "ID", title: "", artist: "",
                                                           album: "", duration: 0, position: 0)))
    }

    @Test func retourALaLigneEtEmoji() {
        let out = ["playing", "ID", "Ligne 1\nLigne 2 🎵", "A", "B", "100", "1"].joined(separator: us) + "\n"
        guard case .track(let t)? = MusicParser.parse(out) else { Issue.record("pas un morceau"); return }
        #expect(t.title == "Ligne 1\nLigne 2 🎵")
        #expect(t.album == "B")
        #expect(t.position == 1)
    }

    @Test func reponsesInattendues() {
        #expect(MusicParser.parse("fast forwarding" + us + "ID" + us + "T" + us + "A" + us + "B" + us + "1" + us + "1") == nil)
        #expect(MusicParser.parse(["playing", "ID", "T"].joined(separator: us)) == nil)
        #expect(MusicParser.parse("") == nil)
    }
}
```

`widget/Tests/AccentReaderTests.swift` :

```swift
import Foundation
import Testing

struct AccentReaderTests {
    let dir = FileManager.default.temporaryDirectory.appendingPathComponent("vinylwidget-accent-\(UUID().uuidString)")

    func write(_ json: String) throws -> URL {
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let url = dir.appendingPathComponent("settings.json")
        try Data(json.utf8).write(to: url)
        return url
    }

    @Test func fichierAbsent() {
        #expect(AccentReader.read(from: dir.appendingPathComponent("absent.json")) == "#f5c542")
    }

    @Test func couleurValide() throws {
        #expect(AccentReader.read(from: try write(##"{"accent":"#FF6b6b","texture":"black"}"##)) == "#FF6b6b")
    }

    @Test func couleurInvalide() throws {
        #expect(AccentReader.read(from: try write(##"{"accent":"#zzz"}"##)) == "#f5c542")
        #expect(AccentReader.read(from: try write(#"{"accent":42}"#)) == "#f5c542")
        #expect(AccentReader.read(from: try write("[1,2]")) == "#f5c542")
    }
}
```

- [ ] **Step 3: Lancer les tests pour vérifier qu'ils échouent**

Run : la commande de test Swift.
Expected : échec. `xcodegen` signale le chemin source manquant `App/MusicParser.swift`, ou bien la compilation ne trouve pas `MusicParser` / `AccentReader`.

- [ ] **Step 4: Écrire le code**

`widget/App/MusicParser.swift` :

```swift
import Foundation

struct TrackInfo: Equatable {
    var state: NowPlaying.PlayerState
    var trackID: String
    var title: String
    var artist: String
    var album: String
    var duration: TimeInterval
    var position: TimeInterval
}

enum MusicReading: Equatable {
    case closed
    case stopped
    case track(TrackInfo)
}

/// Analyse la sortie du script d'état de MusicBridge (champs séparés par ASCII 31).
enum MusicParser {
    static let separator = "\u{1F}"

    static func parse(_ output: String) -> MusicReading? {
        let text = output.hasSuffix("\n") ? String(output.dropLast()) : output
        if text == "closed" { return .closed }
        if text == "stopped" { return .stopped }
        let f = text.components(separatedBy: separator)
        guard f.count == 7, let state = NowPlaying.PlayerState(rawValue: f[0]) else { return nil }
        return .track(TrackInfo(state: state, trackID: f[1], title: f[2], artist: f[3], album: f[4],
                                duration: seconds(f[5]), position: seconds(f[6])))
    }

    /// Secondes AppleScript ("213,5" ou "213.5"). Valeur absente ou invalide → 0.
    static func seconds(_ s: String) -> TimeInterval {
        let v = Double(s.trimmingCharacters(in: .whitespaces).replacingOccurrences(of: ",", with: ".")) ?? 0
        return v.isFinite && v > 0 ? v : 0
    }
}
```

`widget/App/AccentReader.swift` :

```swift
import Foundation

/// Couleur d'accent choisie dans Vinyl (champ `accent` de ses réglages).
enum AccentReader {
    static var vinylSettingsURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Application Support/Vinyl/settings.json")
    }

    static func read(from url: URL = vinylSettingsURL) -> String {
        guard let data = try? Data(contentsOf: url),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let hex = obj["accent"] as? String,
              hex.range(of: "^#[0-9a-fA-F]{6}$", options: .regularExpression) != nil
        else { return NowPlaying.defaultAccent }
        return hex
    }
}
```

- [ ] **Step 5: Lancer les tests pour vérifier qu'ils passent**

Run : la commande de test Swift.
Expected : `** TEST SUCCEEDED **`.

- [ ] **Step 6: Commit**

```bash
git add widget/project.yml widget/App widget/Tests
git commit -m "feat(widget): lecture des réponses de Musique et de la couleur de Vinyl

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: L'assistant « Vinyl Widget »

**Files:**
- Create: `widget/App/MusicBridge.swift`, `widget/App/Artwork.swift`, `widget/App/StateReader.swift`, `widget/App/NowPlayingMonitor.swift`, `widget/App/CommandListener.swift`, `widget/App/LoginItem.swift`, `widget/App/URLRoute.swift`, `widget/App/VinylWidgetApp.swift`
- Modify: `widget/project.yml` (cible `Vinyl Widget`, schéma `Vinyl Widget`)

**Interfaces:**
- Consumes : `NowPlaying`, `StateStore`, `Command`, `AppGroup`, `StateDiff` (Tasks 2-3), `MusicParser`, `MusicReading`, `AccentReader` (Task 4).
- Produces :
  - l'app `Vinyl Widget.app` (schéma URL `vinylwidget`) ;
  - `enum BridgeError: Error { permissionDenied, timeout, failed(String) }` ;
  - `final class MusicBridge` : `isRunning`, `readState()`, `writeArtwork(to:)`, `send(_:)`.

- [ ] **Step 1: Déclarer la cible de l'app**

Dans `widget/project.yml`, ajouter sous `targets:` (avant `VinylWidgetTests`) :

```yaml
  Vinyl Widget:
    type: application
    platform: macOS
    sources:
      - path: App
      - path: Shared
    info:
      path: App/Info.plist
      properties:
        CFBundleDisplayName: Vinyl Widget
        LSUIElement: true
        LSApplicationCategoryType: public.app-category.music
        NSAppleEventsUsageDescription: Vinyl Widget lit le morceau en cours dans Musique et le pilote depuis le widget.
        VinylAppGroup: $(TeamIdentifierPrefix)com.peuraud.vinylwidget
        CFBundleURLTypes:
          - CFBundleURLName: com.peuraud.vinylwidget
            CFBundleURLSchemes: [vinylwidget]
    entitlements:
      path: App/VinylWidget.entitlements
      properties:
        com.apple.security.automation.apple-events: true
        com.apple.security.application-groups:
          - $(TeamIdentifierPrefix)com.peuraud.vinylwidget
    settings:
      base:
        PRODUCT_BUNDLE_IDENTIFIER: com.peuraud.vinylwidget
        PRODUCT_NAME: Vinyl Widget
```

et sous `schemes:` :

```yaml
  Vinyl Widget:
    build:
      targets:
        Vinyl Widget: all
    run:
      config: Debug
    archive:
      config: Release
```

- [ ] **Step 2: Écrire `widget/App/MusicBridge.swift`**

```swift
import AppKit

enum BridgeError: Error {
    case permissionDenied
    case timeout
    case failed(String)
}

/// Dialogue avec Musique par /usr/bin/osascript. Ne lance jamais Musique, sauf sur ⏯ (send(.playpause)).
final class MusicBridge {
    static let bundleID = "com.apple.Music"

    var isRunning: Bool { !NSRunningApplication.runningApplications(withBundleIdentifier: Self.bundleID).isEmpty }

    private static let stateScript = """
    if application id "com.apple.Music" is not running then return "closed"
    set US to (ASCII character 31)
    tell application id "com.apple.Music"
        set st to player state as text
        if st is "stopped" then return "stopped"
        set t to current track
        return st & US & (persistent ID of t) & US & (name of t) & US & (artist of t) & US & (album of t) & US & ((duration of t) as text) & US & (player position as text)
    end tell
    """

    private static let artworkScript = """
    on run argv
        if application id "com.apple.Music" is not running then return "none"
        tell application id "com.apple.Music"
            if (count of artworks of current track) is 0 then return "none"
            set d to raw data of artwork 1 of current track
        end tell
        set f to open for access (POSIX file (item 1 of argv)) with write permission
        try
            set eof f to 0
            write d to f
        end try
        close access f
        return "ok"
    end run
    """

    func readState() -> Result<MusicReading, BridgeError> {
        guard isRunning else { return .success(.closed) }
        return run(Self.stateScript).flatMap { out -> Result<MusicReading, BridgeError> in
            guard let reading = MusicParser.parse(out) else { return .failure(.failed("réponse inattendue : \(out)")) }
            return .success(reading)
        }
    }

    /// Écrit les octets bruts de la pochette du morceau courant dans `url`. false si pas de pochette.
    func writeArtwork(to url: URL) -> Bool {
        guard isRunning, case .success(let out) = run(Self.artworkScript, args: [url.path]) else { return false }
        return out.trimmingCharacters(in: .whitespacesAndNewlines) == "ok"
    }

    func send(_ command: Command) {
        switch command {
        case .playpause:
            if isRunning {
                _ = run(#"tell application id "com.apple.Music" to playpause"#)
            } else {
                _ = run(#"tell application id "com.apple.Music" to play"#, timeout: 15) // lance Musique
            }
        case .next:
            guard isRunning else { return }
            _ = run(#"tell application id "com.apple.Music" to next track"#)
        case .previous:
            guard isRunning else { return }
            _ = run(#"tell application id "com.apple.Music" to previous track"#)
        }
    }

    private func run(_ script: String, args: [String] = [], timeout: TimeInterval = 2) -> Result<String, BridgeError> {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        p.arguments = ["-e", script, "--"] + args
        let out = Pipe(), err = Pipe()
        p.standardOutput = out
        p.standardError = err
        let done = DispatchSemaphore(value: 0)
        p.terminationHandler = { _ in done.signal() }
        do { try p.run() } catch { return .failure(.failed(error.localizedDescription)) }
        if done.wait(timeout: .now() + timeout) == .timedOut {
            p.terminate()
            return .failure(.timeout)
        }
        let stdout = String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        let stderr = String(data: err.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        guard p.terminationStatus == 0 else {
            return .failure(stderr.contains("-1743") ? .permissionDenied : .failed(stderr))
        }
        return .success(stdout)
    }
}
```

- [ ] **Step 3: Écrire `widget/App/Artwork.swift`**

```swift
import Foundation
import ImageIO
import UniformTypeIdentifiers

enum Artwork {
    /// Réduit l'image à `maxPixel` de côté au maximum et l'encode en JPEG.
    static func jpegThumbnail(from url: URL, maxPixel: Int = 600) -> Data? {
        guard let src = CGImageSourceCreateWithURL(url as CFURL, nil) else { return nil }
        let opts: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceThumbnailMaxPixelSize: maxPixel,
            kCGImageSourceCreateThumbnailWithTransform: true,
        ]
        guard let img = CGImageSourceCreateThumbnailAtIndex(src, 0, opts as CFDictionary) else { return nil }
        let data = NSMutableData()
        guard let dst = CGImageDestinationCreateWithData(data, UTType.jpeg.identifier as CFString, 1, nil) else { return nil }
        CGImageDestinationAddImage(dst, img, [kCGImageDestinationLossyCompressionQuality: 0.85] as CFDictionary)
        return CGImageDestinationFinalize(dst) ? data as Data : nil
    }
}
```

- [ ] **Step 4: Écrire `widget/App/StateReader.swift`**

```swift
import Foundation

/// Construit le nouvel état à partir de Musique. Appelé hors du fil principal.
enum StateReader {
    /// nil = rien de fiable (erreur passagère) : garder l'état précédent.
    static func read(previous: NowPlaying, bridge: MusicBridge, store: StateStore, now: Date) -> NowPlaying? {
        let accent = AccentReader.read()
        switch bridge.readState() {
        case .failure(.permissionDenied):
            var n = previous
            n.availability = .permissionDenied
            n.accentHex = accent
            return n
        case .failure(let e):
            NSLog("Vinyl Widget : lecture de Musique impossible : %@", "\(e)")
            return nil
        case .success(.closed):
            return previous.closed(accentHex: accent, now: now)
        case .success(.stopped):
            guard previous.trackID != nil else { return .neverPlayed(accentHex: accent, now: now) }
            var n = previous
            n.availability = .running
            n.state = .stopped
            n.position = 0
            n.capturedAt = now
            n.accentHex = accent
            return n
        case .success(.track(let t)):
            var n = NowPlaying(availability: .running, state: t.state, trackID: t.trackID, title: t.title,
                               artist: t.artist, album: t.album, duration: t.duration, position: t.position,
                               capturedAt: now, artworkFile: nil, accentHex: accent)
            if previous.trackID == t.trackID { n.artworkFile = previous.artworkFile }
            if n.artworkFile == nil { n.artworkFile = fetchArtwork(trackID: t.trackID, bridge: bridge, store: store) }
            return n
        }
    }

    private static func fetchArtwork(trackID: String, bridge: MusicBridge, store: StateStore) -> String? {
        let raw = FileManager.default.temporaryDirectory.appendingPathComponent("vinylwidget-artwork.raw")
        try? FileManager.default.removeItem(at: raw)
        defer { try? FileManager.default.removeItem(at: raw) }
        guard bridge.writeArtwork(to: raw), let jpeg = Artwork.jpegThumbnail(from: raw) else { return nil }
        return try? store.saveArtwork(jpeg, trackID: trackID)
    }
}
```

- [ ] **Step 5: Écrire `widget/App/NowPlayingMonitor.swift`**

```swift
import AppKit

/// Écoute Musique et publie l'état pour le widget. Toutes les méthodes s'appellent sur le fil principal.
final class NowPlayingMonitor {
    private let bridge: MusicBridge
    private let store: StateStore
    private let reload: () -> Void
    private let queue = DispatchQueue(label: "com.peuraud.vinylwidget.music")
    private var current: NowPlaying
    private var busy = false
    private var pending = false
    private var observers: [NSObjectProtocol] = []
    private var timer: Timer?

    init(bridge: MusicBridge, store: StateStore, reload: @escaping () -> Void) {
        self.bridge = bridge
        self.store = store
        self.reload = reload
        current = store.load() ?? .neverPlayed(accentHex: AccentReader.read())
    }

    func start() {
        observers.append(DistributedNotificationCenter.default().addObserver(
            forName: Notification.Name("com.apple.Music.playerInfo"), object: nil, queue: .main
        ) { [weak self] _ in self?.refresh() })

        let ws = NSWorkspace.shared.notificationCenter
        observers.append(ws.addObserver(forName: NSWorkspace.didLaunchApplicationNotification,
                                        object: nil, queue: .main) { [weak self] note in
            if Self.isMusic(note) { self?.refresh() }
        })
        // À la fermeture, aucun AppleScript : Musique serait relancée.
        observers.append(ws.addObserver(forName: NSWorkspace.didTerminateApplicationNotification,
                                        object: nil, queue: .main) { [weak self] note in
            guard let self, Self.isMusic(note) else { return }
            self.publish(self.current.closed(accentHex: AccentReader.read(), now: Date()))
        })

        timer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            guard let self, self.current.availability == .running, self.current.state == .playing else { return }
            self.refresh()
        }
        refresh()
    }

    func refresh() {
        if busy { pending = true; return }
        busy = true
        let previous = current
        queue.async { [bridge, store] in
            let next = StateReader.read(previous: previous, bridge: bridge, store: store, now: Date())
            DispatchQueue.main.async {
                self.busy = false
                if let next { self.publish(next) }
                if self.pending {
                    self.pending = false
                    self.refresh()
                }
            }
        }
    }

    private func publish(_ next: NowPlaying) {
        guard StateDiff.shouldPublish(old: current, new: next) else { return }
        do {
            try store.save(next)
        } catch {
            NSLog("Vinyl Widget : écriture de l'état impossible : %@", "\(error)")
            return
        }
        current = next
        reload()
    }

    private static func isMusic(_ note: Notification) -> Bool {
        (note.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication)?.bundleIdentifier
            == MusicBridge.bundleID
    }
}
```

- [ ] **Step 6: Écrire `CommandListener`, `LoginItem`, `URLRoute` et le point d'entrée**

`widget/App/CommandListener.swift` :

```swift
import Foundation

/// Reçoit les ordres du widget (notifications Darwin) et les transmet au fil principal.
enum CommandListener {
    private static var handler: ((Command) -> Void)?

    static func start(_ h: @escaping (Command) -> Void) {
        handler = h
        let center = CFNotificationCenterGetDarwinNotifyCenter()
        for c in Command.allCases {
            CFNotificationCenterAddObserver(center, nil, { _, _, name, _, _ in
                guard let cf = name?.rawValue, let cmd = Command(notificationName: cf as String) else { return }
                DispatchQueue.main.async { CommandListener.handler?(cmd) }
            }, c.notificationName as CFString, nil, .deliverImmediately)
        }
    }
}
```

`widget/App/LoginItem.swift` :

```swift
import Foundation
import ServiceManagement

enum LoginItem {
    /// Inscrit l'app pour qu'elle démarre à chaque ouverture de session.
    static func registerIfNeeded() {
        let service = SMAppService.mainApp
        guard service.status != .enabled else { return }
        do { try service.register() } catch {
            NSLog("Vinyl Widget : démarrage à l'ouverture de session impossible : %@", "\(error)")
        }
    }
}
```

`widget/App/URLRoute.swift` :

```swift
import AppKit

/// URL ouvertes par un clic sur le widget.
enum URLRoute {
    static func handle(_ url: URL) {
        guard url.scheme == "vinylwidget" else { return }
        switch url.host {
        case "music":
            if let app = NSWorkspace.shared.urlForApplication(withBundleIdentifier: MusicBridge.bundleID) {
                NSWorkspace.shared.openApplication(at: app, configuration: NSWorkspace.OpenConfiguration())
            }
        case "automation":
            if let u = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation") {
                NSWorkspace.shared.open(u)
            }
        default:
            break
        }
    }
}
```

`widget/App/VinylWidgetApp.swift` :

```swift
import AppKit
import WidgetKit

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let bridge = MusicBridge()
    private var monitor: NowPlayingMonitor?

    func applicationDidFinishLaunching(_ notification: Notification) {
        LoginItem.registerIfNeeded()
        guard let dir = AppGroup.containerURL else {
            NSLog("Vinyl Widget : App Group indisponible (%@)", AppGroup.identifier)
            return
        }
        let monitor = NowPlayingMonitor(bridge: bridge, store: StateStore(directory: dir),
                                        reload: { WidgetCenter.shared.reloadAllTimelines() })
        self.monitor = monitor
        CommandListener.start { [bridge] cmd in
            DispatchQueue.global(qos: .userInitiated).async { bridge.send(cmd) }
        }
        monitor.start()
    }

    func application(_ application: NSApplication, open urls: [URL]) {
        urls.forEach(URLRoute.handle)
    }
}

@main
enum Main {
    static let delegate = AppDelegate()

    static func main() {
        let app = NSApplication.shared
        app.delegate = delegate
        app.setActivationPolicy(.accessory)
        app.run()
    }
}
```

- [ ] **Step 7: Compiler l'assistant et relancer les tests**

Run :

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl/widget" && xcodegen generate && \
xcodebuild build -project VinylWidget.xcodeproj -scheme "Vinyl Widget" -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO 2>&1 | tail -5
```

Expected : `** BUILD SUCCEEDED **`, sans erreur. Ensuite, la commande de test Swift doit afficher `** TEST SUCCEEDED **`.

- [ ] **Step 8: Commit**

```bash
git add widget/project.yml widget/App
git commit -m "feat(widget): assistant sans fenêtre qui suit Musique et exécute les commandes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: L'extension WidgetKit (3 tailles, boutons)

**Files:**
- Create: `widget/Extension/VinylWidget.swift`, `widget/Extension/Provider.swift`, `widget/Extension/Intents.swift`
- Create: `widget/Extension/Views/Geometry.swift`, `widget/Extension/Views/Color+Hex.swift`, `widget/Extension/Views/DiscView.swift`, `widget/Extension/Views/TonearmView.swift`, `widget/Extension/Views/TurntableView.swift`, `widget/Extension/Views/Parts.swift`, `widget/Extension/Views/FamilyViews.swift`
- Modify: `widget/project.yml` (cible `VinylWidgetExtension`, dépendance de l'app)

**Interfaces:**
- Consumes : `NowPlaying`, `StateStore`, `AppGroup`, `Command`, `VinylEntry`, `TimelineBuilder`, `ArmMath` (Tasks 2-3) ; schéma `vinylwidget://` (Task 5).
- Produces : l'extension `VinylWidgetExtension.appex`, embarquée dans `Vinyl Widget.app/Contents/PlugIns`.

- [ ] **Step 1: Déclarer l'extension et l'embarquer**

Dans `widget/project.yml`, ajouter sous `targets:` :

```yaml
  VinylWidgetExtension:
    type: app-extension
    platform: macOS
    sources:
      - path: Extension
      - path: Shared
    info:
      path: Extension/Info.plist
      properties:
        CFBundleDisplayName: Vinyl
        VinylAppGroup: $(TeamIdentifierPrefix)com.peuraud.vinylwidget
        NSExtension:
          NSExtensionPointIdentifier: com.apple.widgetkit-extension
    entitlements:
      path: Extension/VinylWidgetExtension.entitlements
      properties:
        com.apple.security.app-sandbox: true
        com.apple.security.application-groups:
          - $(TeamIdentifierPrefix)com.peuraud.vinylwidget
    settings:
      base:
        PRODUCT_BUNDLE_IDENTIFIER: com.peuraud.vinylwidget.extension
    dependencies:
      - sdk: WidgetKit.framework
      - sdk: SwiftUI.framework
```

et dans la cible `Vinyl Widget`, ajouter :

```yaml
    dependencies:
      - target: VinylWidgetExtension
        embed: true
        codeSign: true
```

- [ ] **Step 2: Écrire la déclaration, le Provider et les intents**

`widget/Extension/VinylWidget.swift` :

```swift
import SwiftUI
import WidgetKit

extension VinylEntry: TimelineEntry {}

@main
struct VinylWidgetBundle: WidgetBundle {
    var body: some Widget { VinylWidget() }
}

struct VinylWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "VinylWidget", provider: Provider()) { entry in
            WidgetRootView(entry: entry)
        }
        .configurationDisplayName("Vinyl")
        .description("La platine Vinyl pour Apple Music.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
        .contentMarginsDisabled()
    }
}
```

`widget/Extension/Provider.swift` :

```swift
import Foundation
import WidgetKit

extension NowPlaying {
    /// Aperçu de la galerie de widgets.
    static let sample = NowPlaying(availability: .running, state: .playing, trackID: "sample",
                                   title: "Titre du morceau", artist: "Artiste", album: "Album",
                                   duration: 240, position: 80, capturedAt: Date(), artworkFile: nil,
                                   accentHex: defaultAccent)
}

struct Provider: TimelineProvider {
    private func load(now: Date) -> NowPlaying {
        guard let dir = AppGroup.containerURL, let np = StateStore(directory: dir).load() else {
            return .neverPlayed(now: now)
        }
        return np
    }

    func placeholder(in context: Context) -> VinylEntry {
        VinylEntry(date: Date(), nowPlaying: .sample, position: 80)
    }

    func getSnapshot(in context: Context, completion: @escaping (VinylEntry) -> Void) {
        let now = Date()
        let np = context.isPreview ? .sample : load(now: now)
        completion(TimelineBuilder.entries(for: np, now: now).entries[0])
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<VinylEntry>) -> Void) {
        let now = Date()
        let r = TimelineBuilder.entries(for: load(now: now), now: now)
        completion(Timeline(entries: r.entries, policy: r.refreshAtEnd ? .atEnd : .never))
    }
}
```

`widget/Extension/Intents.swift` :

```swift
import AppIntents

// L'extension ne parle jamais à Musique : elle prévient l'assistant.
struct PlayPauseIntent: AppIntent {
    static var title: LocalizedStringResource = "Lecture / pause"
    func perform() async throws -> some IntentResult {
        Command.playpause.post()
        return .result()
    }
}

struct NextIntent: AppIntent {
    static var title: LocalizedStringResource = "Morceau suivant"
    func perform() async throws -> some IntentResult {
        Command.next.post()
        return .result()
    }
}

struct PreviousIntent: AppIntent {
    static var title: LocalizedStringResource = "Morceau précédent"
    func perform() async throws -> some IntentResult {
        Command.previous.post()
        return .result()
    }
}
```

- [ ] **Step 3: Écrire les vues de base**

`widget/Extension/Views/Geometry.swift` :

```swift
import CoreGraphics

/// Géométrie du socle de Vinyl (frontend/src/turntable/arm.ts, ARM_GEOMETRY), en points sur 420×320.
enum Geometry {
    static let width: CGFloat = 420
    static let height: CGFloat = 320
    static let pivot = CGPoint(x: 324, y: 24)
    static let armLength: CGFloat = 222
    static let discCenter = CGPoint(x: 162, y: 160)
    static let discRadius: CGFloat = 132
    static let labelRatio: CGFloat = 0.36
}
```

`widget/Extension/Views/Color+Hex.swift` :

```swift
import SwiftUI

extension Color {
    /// "#rrggbb" → couleur ; valeur invalide → accent par défaut de Vinyl.
    init(hex: String) {
        var s = hex.hasPrefix("#") ? String(hex.dropFirst()) : hex
        if s.count != 6 || UInt32(s, radix: 16) == nil { s = "f5c542" }
        let v = UInt32(s, radix: 16)!
        self.init(red: Double((v >> 16) & 0xff) / 255, green: Double((v >> 8) & 0xff) / 255,
                  blue: Double(v & 0xff) / 255)
    }
}
```

`widget/Extension/Views/DiscView.swift` :

```swift
import AppIntents
import SwiftUI
import WidgetKit

/// Étiquette centrale : pochette, ou couleur d'accent avec une note de musique.
struct LabelView: View {
    let entry: VinylEntry

    private var artwork: NSImage? {
        guard let name = entry.nowPlaying.artworkFile, let dir = AppGroup.containerURL else { return nil }
        return NSImage(contentsOf: StateStore(directory: dir).artworkURL(name))
    }

    var body: some View {
        if let img = artwork {
            Image(nsImage: img).resizable().scaledToFill().clipShape(Circle())
        } else {
            Circle().fill(Color(hex: entry.nowPlaying.accentHex))
                .overlay(Image(systemName: "music.note").font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.black.opacity(0.6)))
        }
    }
}

/// Disque noir : sillons, reflet, étiquette, trou central. `playButton` fait de l'étiquette un bouton ⏯.
struct DiscView: View {
    let entry: VinylEntry
    let diameter: CGFloat
    var playButton = false

    var body: some View {
        ZStack {
            Circle().fill(Color(white: 0.06))
            ForEach(0..<14, id: \.self) { i in
                Circle().stroke(Color.white.opacity(0.05), lineWidth: 0.6)
                    .padding(diameter * (0.04 + CGFloat(i) * 0.022))
            }
            Circle().fill(AngularGradient(colors: [.clear, .white.opacity(0.10), .clear, .clear,
                                                   .white.opacity(0.07), .clear], center: .center))
            label.frame(width: diameter * Geometry.labelRatio, height: diameter * Geometry.labelRatio)
            Circle().fill(Color.black).frame(width: diameter * 0.025, height: diameter * 0.025)
        }
        .frame(width: diameter, height: diameter)
    }

    @ViewBuilder private var label: some View {
        if playButton {
            Button(intent: PlayPauseIntent()) { LabelView(entry: entry) }.buttonStyle(.plain)
        } else {
            LabelView(entry: entry)
        }
    }
}
```

`widget/Extension/Views/TonearmView.swift` :

```swift
import SwiftUI

/// Bras de lecture : angle 0 = vertical vers le bas depuis le pivot, positif vers le disque.
struct TonearmView: View {
    let angle: Double
    let accent: Color
    let scale: CGFloat

    var body: some View {
        Canvas { ctx, _ in
            let s = scale
            let pivot = CGPoint(x: Geometry.pivot.x * s, y: Geometry.pivot.y * s)
            let rad = angle * .pi / 180
            let tip = CGPoint(x: pivot.x - sin(rad) * Geometry.armLength * s,
                              y: pivot.y + cos(rad) * Geometry.armLength * s)
            let back = CGPoint(x: pivot.x + sin(rad) * 20 * s, y: pivot.y - cos(rad) * 20 * s)
            var arm = Path()
            arm.move(to: back)
            arm.addLine(to: tip)
            ctx.stroke(arm, with: .color(Color(white: 0.78)), style: StrokeStyle(lineWidth: 5 * s, lineCap: .round))
            ctx.fill(Path(ellipseIn: CGRect(x: back.x - 11 * s, y: back.y - 11 * s, width: 22 * s, height: 22 * s)),
                     with: .color(accent))
            ctx.fill(Path(ellipseIn: CGRect(x: pivot.x - 16 * s, y: pivot.y - 16 * s, width: 32 * s, height: 32 * s)),
                     with: .color(Color(white: 0.3)))
            ctx.fill(Path(roundedRect: CGRect(x: tip.x - 7 * s, y: tip.y - 10 * s, width: 14 * s, height: 20 * s),
                          cornerRadius: 3 * s), with: .color(Color(white: 0.85)))
        }
    }
}
```

`widget/Extension/Views/TurntableView.swift` :

```swift
import SwiftUI

/// Disque + bras, à l'échelle de la place disponible (rapport 420×320 conservé).
struct TurntableView: View {
    let entry: VinylEntry
    var playButton = false

    var body: some View {
        GeometryReader { geo in
            let s = min(geo.size.width / Geometry.width, geo.size.height / Geometry.height)
            ZStack(alignment: .topLeading) {
                DiscView(entry: entry, diameter: Geometry.discRadius * 2 * s, playButton: playButton)
                    .offset(x: (Geometry.discCenter.x - Geometry.discRadius) * s,
                            y: (Geometry.discCenter.y - Geometry.discRadius) * s)
                TonearmView(angle: entry.armAngle, accent: Color(hex: entry.nowPlaying.accentHex), scale: s)
                    .allowsHitTesting(false)
            }
            .frame(width: Geometry.width * s, height: Geometry.height * s, alignment: .topLeading)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .opacity(entry.nowPlaying.availability == .closed ? 0.55 : 1)
    }
}
```

- [ ] **Step 4: Écrire les éléments et les trois tailles**

`widget/Extension/Views/Parts.swift` :

```swift
import AppIntents
import SwiftUI
import WidgetKit

enum WidgetLinks {
    static let music = URL(string: "vinylwidget://music")!
    static let automation = URL(string: "vinylwidget://automation")!
}

/// Message d'état, ou nil si Musique tourne normalement.
func statusText(_ np: NowPlaying) -> String? {
    switch np.availability {
    case .neverPlayed: return "Lance un morceau dans Musique"
    case .permissionDenied: return "Autorise Vinyl Widget à piloter Musique"
    case .closed: return "Musique fermée"
    case .running: return nil
    }
}

struct InfoView: View {
    let entry: VinylEntry

    var body: some View {
        let np = entry.nowPlaying
        VStack(alignment: .leading, spacing: 2) {
            if np.availability == .neverPlayed || np.availability == .permissionDenied {
                Text(statusText(np) ?? "").font(.headline).foregroundStyle(.white).lineLimit(2)
            } else {
                Text(np.title ?? "").font(.headline).foregroundStyle(.white).lineLimit(1)
                Text(np.artist ?? "").font(.subheadline).foregroundStyle(.white.opacity(0.6)).lineLimit(1)
                if let s = statusText(np) {
                    Text(s).font(.caption).foregroundStyle(.white.opacity(0.45))
                }
            }
        }
    }
}

struct ProgressBarView: View {
    let entry: VinylEntry

    var body: some View {
        GeometryReader { g in
            ZStack(alignment: .leading) {
                Capsule().fill(.white.opacity(0.15))
                Capsule().fill(Color(hex: entry.nowPlaying.accentHex)).frame(width: g.size.width * entry.progress)
            }
        }
        .frame(height: 4)
        .widgetAccentable()
    }
}

struct ControlsView: View {
    let entry: VinylEntry

    var body: some View {
        HStack(spacing: 22) {
            Button(intent: PreviousIntent()) { Image(systemName: "backward.fill") }
            Button(intent: PlayPauseIntent()) {
                Image(systemName: entry.isPlaying ? "pause.fill" : "play.fill").font(.title2)
            }
            Button(intent: NextIntent()) { Image(systemName: "forward.fill") }
        }
        .buttonStyle(.plain)
        .foregroundStyle(.white)
        .widgetAccentable()
    }
}
```

`widget/Extension/Views/FamilyViews.swift` :

```swift
import SwiftUI
import WidgetKit

struct WidgetRootView: View {
    @Environment(\.widgetFamily) private var family
    let entry: VinylEntry

    var body: some View {
        content
            .containerBackground(for: .widget) { Color(red: 0.08, green: 0.08, blue: 0.08) }
            .widgetURL(entry.nowPlaying.availability == .permissionDenied ? WidgetLinks.automation : WidgetLinks.music)
    }

    @ViewBuilder private var content: some View {
        switch family {
        case .systemSmall: SmallView(entry: entry)
        case .systemMedium: MediumView(entry: entry)
        default: LargeView(entry: entry)
        }
    }
}

struct SmallView: View {
    let entry: VinylEntry

    var body: some View {
        ZStack(alignment: .bottom) {
            TurntableView(entry: entry, playButton: true).padding(8)
            if let s = statusText(entry.nowPlaying) {
                Text(s).font(.caption2).foregroundStyle(.white.opacity(0.8)).lineLimit(2)
                    .multilineTextAlignment(.center).padding(.horizontal, 8).padding(.bottom, 6)
            }
        }
    }
}

struct MediumView: View {
    let entry: VinylEntry

    var body: some View {
        HStack(spacing: 12) {
            TurntableView(entry: entry).frame(maxWidth: .infinity)
            VStack(alignment: .leading, spacing: 10) {
                InfoView(entry: entry)
                ProgressBarView(entry: entry)
                ControlsView(entry: entry)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(14)
    }
}

struct LargeView: View {
    let entry: VinylEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            TurntableView(entry: entry)
            InfoView(entry: entry)
            ProgressBarView(entry: entry)
            ControlsView(entry: entry).frame(maxWidth: .infinity)
        }
        .padding(16)
    }
}
```

- [ ] **Step 5: Compiler le tout et relancer les tests**

Run :

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl/widget" && xcodegen generate && \
xcodebuild build -project VinylWidget.xcodeproj -scheme "Vinyl Widget" -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO 2>&1 | tail -5 && \
ls ~/Library/Developer/Xcode/DerivedData/VinylWidget-*/Build/Products/Debug/"Vinyl Widget.app"/Contents/PlugIns
```

Expected : `** BUILD SUCCEEDED **`, puis `VinylWidgetExtension.appex` listé. Ensuite, la commande de test Swift doit afficher `** TEST SUCCEEDED **`.

- [ ] **Step 6: Commit**

```bash
git add widget/project.yml widget/Extension
git commit -m "feat(widget): widget WidgetKit petit, moyen et grand avec ⏮ ⏯ ⏭

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: README du widget et première installation

**Files:**
- Create: `widget/README.md`
- Modify: `README.md` (section « Widget de bureau »)

**Interfaces:** aucune.

- [ ] **Step 1: Écrire `widget/README.md`**

```markdown
# Vinyl Widget

Widget de bureau macOS (petit, moyen, grand) qui affiche et pilote Apple Music.
L'assistant « Vinyl Widget » tourne sans fenêtre et démarre avec ta session.

## Première installation

1. `brew install xcodegen`
2. `cd widget && xcodegen generate && open VinylWidget.xcodeproj`
3. Xcode → Réglages → Comptes → « + » : ajoute ton identifiant Apple.
4. Dans le projet, onglet « Signing & Capabilities » : choisis l'équipe « (Personal Team) » pour
   les cibles « Vinyl Widget » et « VinylWidgetExtension ».
5. Choisis le schéma « Vinyl Widget » et lance (⌘R). Accepte l'accès à Musique.
6. Clic droit sur le bureau → Modifier les widgets → Vinyl → choisis une taille.

Ensuite, les mises à jour se font depuis Vinyl : l'icône ☁️↓ en haut à droite de la fenêtre
apparaît quand la signature expire (compte gratuit : environ 7 jours) ou quand le code du widget change.
En secours : ⌘R depuis Xcode (refaire l'étape 4 si `xcodegen generate` a été relancé).

## Tests

    cd widget && xcodegen generate
    xcodebuild test -project VinylWidget.xcodeproj -scheme VinylWidgetTests -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO

## Recette manuelle

- [ ] Les trois tailles s'ajoutent sur le bureau.
- [ ] Un morceau lancé dans Musique apparaît en quelques secondes ; le bras avance.
- [ ] ⏯ met en pause (bras levé) et relance ; ⏭ et ⏮ changent de morceau.
- [ ] Avancer dans le morceau depuis Musique recale le bras en moins de 15 s.
- [ ] Fermer Musique : « Musique fermée », dernier morceau estompé ; ⏯ relance Musique et la lecture.
- [ ] Un clic hors des boutons ouvre Musique.
- [ ] Refuser l'accès (Réglages Système → Confidentialité → Automatisation) : message dans le widget,
      le clic ouvre ce panneau ; réautoriser rétablit l'affichage.
- [ ] Redémarrer le Mac : le widget fonctionne sans rien ouvrir.

## Si ça ne marche pas

- Widget absent de la galerie : lance une fois « Vinyl Widget » depuis /Applications.
- Widget figé : `killall "Vinyl Widget"; open -a "Vinyl Widget"`.
- Journal de l'assistant : `log stream --predicate 'process == "Vinyl Widget"'`.
```

- [ ] **Step 2: Ajouter une section au `README.md` racine**

Insérer après la section « Spotify » :

```markdown
## Widget de bureau

Le dossier `widget/` contient un vrai widget macOS (Apple Music) indépendant de Vinyl.
Installation et recette : `widget/README.md`. Une fois installé, Vinyl signale ses mises à jour
par une icône ☁️↓ en haut à droite de la fenêtre principale.
```

- [ ] **Step 3: 👤 Pierre — première installation**

Suivre `widget/README.md` § « Première installation », étapes 3 à 6. L'agent s'arrête ici et attend le retour de Pierre. Points à confirmer :
- l'assistant démarre ;
- macOS demande l'accès à Musique ;
- le widget apparaît dans la galerie et affiche le morceau en cours.

Si la galerie ne montre pas « Vinyl », relever la sortie de `log stream --predicate 'process == "Vinyl Widget" OR process == "VinylWidgetExtension"'` pendant le lancement.

- [ ] **Step 4: Commit**

```bash
git add widget/README.md README.md
git commit -m "docs(widget): installation, tests et recette du widget de bureau

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Détection du besoin de mise à jour (Go)

**Files:**
- Create: `internal/widgetupdate/detect.go`
- Test: `internal/widgetupdate/detect_test.go`

**Interfaces:**
- Produces :
  - `type Reason string` avec les constantes `NotInstalled = "notInstalled"`, `Expired = "expired"`, `ExpiresSoon = "expiresSoon"`, `NewVersion = "newVersion"` ;
  - `type Observed struct { Installed bool; Expiry time.Time; SourceTree, InstalledTree string; Now time.Time }` ;
  - `func Detect(o Observed) []Reason` ;
  - `func Message(r Reason, expiry, now time.Time) string` ;
  - `func ParseExpiry(plist string) (time.Time, bool)` ;
  - `func TeamFromPEM(data []byte, now time.Time) (string, bool)`.

- [ ] **Step 1: Écrire les tests**

`internal/widgetupdate/detect_test.go` :

```go
package widgetupdate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"slices"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		o    Observed
		want []Reason
	}{
		{"absent", Observed{SourceTree: "a", Now: testNow}, []Reason{NotInstalled}},
		{"à jour", Observed{Installed: true, Expiry: testNow.Add(72 * time.Hour), SourceTree: "a", InstalledTree: "a", Now: testNow}, nil},
		{"expirée", Observed{Installed: true, Expiry: testNow.Add(-time.Minute), SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{Expired}},
		{"expire pile maintenant", Observed{Installed: true, Expiry: testNow, SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{Expired}},
		{"expire bientôt", Observed{Installed: true, Expiry: testNow.Add(47 * time.Hour), SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{ExpiresSoon}},
		{"nouvelle version", Observed{Installed: true, SourceTree: "b", InstalledTree: "a", Now: testNow}, []Reason{NewVersion}},
		{"expirée puis nouvelle version", Observed{Installed: true, Expiry: testNow.Add(-time.Hour), SourceTree: "b", InstalledTree: "a", Now: testNow}, []Reason{Expired, NewVersion}},
		{"expiration inconnue, dépôt introuvable", Observed{Installed: true, InstalledTree: "a", Now: testNow}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Detect(c.o); !slices.Equal(got, c.want) {
				t.Fatalf("Detect = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	cases := map[string]string{
		Message(NotInstalled, time.Time{}, testNow):           "Widget pas encore installé",
		Message(Expired, testNow, testNow):                     "Signature du widget expirée",
		Message(ExpiresSoon, testNow.Add(47*time.Hour), testNow): "Signature du widget : expire dans 2 jours",
		Message(ExpiresSoon, testNow.Add(5*time.Hour), testNow):  "Signature du widget : expire dans 1 jour",
		Message(NewVersion, time.Time{}, testNow):             "Nouvelle version du widget",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestParseExpiry(t *testing.T) {
	plist := `<?xml version="1.0"?><plist version="1.0"><dict>
	<key>CreationDate</key>
	<date>2026-10-10T08:00:00Z</date>
	<key>ExpirationDate</key>
	<date>2026-10-17T08:00:00Z</date>
	</dict></plist>`
	got, ok := ParseExpiry(plist)
	if !ok || !got.Equal(time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("ParseExpiry = %v, %v", got, ok)
	}
	if _, ok := ParseExpiry("pas un plist"); ok {
		t.Fatal("ParseExpiry doit échouer sur un texte quelconque")
	}
}

func makeCert(t *testing.T, ou string, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Apple Development: Test", OrganizationalUnit: []string{ou}},
		NotBefore:    notAfter.Add(-365 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestTeamFromPEM(t *testing.T) {
	expired := makeCert(t, "OLDTEAM", testNow.Add(-time.Hour))
	valid := makeCert(t, "TEAM123", testNow.Add(300*24*time.Hour))
	if team, ok := TeamFromPEM(append(expired, valid...), testNow); !ok || team != "TEAM123" {
		t.Fatalf("TeamFromPEM = %q, %v", team, ok)
	}
	if _, ok := TeamFromPEM(expired, testNow); ok {
		t.Fatal("un certificat expiré ne doit pas compter")
	}
	if _, ok := TeamFromPEM(nil, testNow); ok {
		t.Fatal("aucun certificat : pas d'équipe")
	}
}
```

- [ ] **Step 2: Lancer les tests pour vérifier qu'ils échouent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test ./internal/widgetupdate/`
Expected: FAIL (`undefined: Observed`).

- [ ] **Step 3: Écrire `internal/widgetupdate/detect.go`**

```go
// Package widgetupdate détecte quand le widget de bureau doit être réinstallé, et le réinstalle.
package widgetupdate

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"regexp"
	"time"
)

type Reason string

const (
	NotInstalled Reason = "notInstalled"
	Expired      Reason = "expired"
	ExpiresSoon  Reason = "expiresSoon"
	NewVersion   Reason = "newVersion"
)

// expirySoon : en dessous de ce délai, on propose de re-signer.
const expirySoon = 48 * time.Hour

// Observed décrit ce qu'on a constaté sur le disque. Valeur zéro d'un champ = inconnue.
type Observed struct {
	Installed     bool
	Expiry        time.Time // ExpirationDate du profil intégré
	SourceTree    string    // git rev-parse HEAD:widget
	InstalledTree string    // empreinte enregistrée à la dernière installation
	Now           time.Time
}

// Detect renvoie les raisons de mettre à jour, par ordre de priorité.
func Detect(o Observed) []Reason {
	if !o.Installed {
		return []Reason{NotInstalled}
	}
	var r []Reason
	if !o.Expiry.IsZero() {
		if !o.Now.Before(o.Expiry) {
			r = append(r, Expired)
		} else if o.Expiry.Sub(o.Now) < expirySoon {
			r = append(r, ExpiresSoon)
		}
	}
	if o.SourceTree != "" && o.SourceTree != o.InstalledTree {
		r = append(r, NewVersion)
	}
	return r
}

// Message est le texte de la bulle au survol de l'icône.
func Message(r Reason, expiry, now time.Time) string {
	switch r {
	case NotInstalled:
		return "Widget pas encore installé"
	case Expired:
		return "Signature du widget expirée"
	case ExpiresSoon:
		days := max(1, int(math.Ceil(expiry.Sub(now).Hours()/24)))
		s := ""
		if days > 1 {
			s = "s"
		}
		return fmt.Sprintf("Signature du widget : expire dans %d jour%s", days, s)
	case NewVersion:
		return "Nouvelle version du widget"
	}
	return ""
}

var expiryRe = regexp.MustCompile(`<key>ExpirationDate</key>\s*<date>([^<]+)</date>`)

// ParseExpiry lit ExpirationDate dans un profil de provisionnement décodé (security cms -D).
func ParseExpiry(plist string) (time.Time, bool) {
	m := expiryRe.FindStringSubmatch(plist)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, m[1])
	return t, err == nil
}

// TeamFromPEM renvoie l'équipe (champ OU) du premier certificat non expiré.
func TeamFromPEM(data []byte, now time.Time) (string, bool) {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return "", false
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || now.After(cert.NotAfter) || len(cert.Subject.OrganizationalUnit) == 0 {
			continue
		}
		return cert.Subject.OrganizationalUnit[0], true
	}
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils passent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test ./internal/widgetupdate/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/widgetupdate
git commit -m "feat(widgetupdate): détection d'une mise à jour du widget nécessaire

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Recompilation et réinstallation (Go)

**Files:**
- Create: `internal/widgetupdate/updater.go`
- Test: `internal/widgetupdate/updater_test.go`

**Interfaces:**
- Consumes : `Observed`, `ParseExpiry`, `TeamFromPEM` (Task 8).
- Produces :
  - `type Runner interface { Run(ctx context.Context, dir, name string, args ...string) (string, error) }` ;
  - `type ExecRunner struct{}` ;
  - `type Updater struct { ProjectDir, AppPath, BuildDir, RecordPath string; XcodeGenPaths []string; Run Runner; Now func() time.Time }` ;
  - `func (u *Updater) Observe(ctx context.Context) Observed` ;
  - `func (u *Updater) Update(ctx context.Context) error` ;
  - `type UserError struct{ Msg string }` ;
  - `func ErrorMessage(err error) string`.

- [ ] **Step 1: Écrire les tests**

`internal/widgetupdate/updater_test.go` :

```go
package widgetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

type call struct {
	dir, name string
	args      []string
}

type fakeRunner struct {
	calls   []call
	respond func(c call) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, dir, name string, args ...string) (string, error) {
	c := call{dir, name, args}
	f.calls = append(f.calls, c)
	return f.respond(c)
}

func (f *fakeRunner) names() []string {
	var n []string
	for _, c := range f.calls {
		n = append(n, filepath.Base(c.name))
	}
	return n
}

// defaultRespond simule une installation réussie ; certPEM = sortie de security find-certificate.
func defaultRespond(certPEM []byte) func(c call) (string, error) {
	return func(c call) (string, error) {
		switch filepath.Base(c.name) {
		case "security":
			if c.args[0] == "find-certificate" {
				return string(certPEM), nil
			}
			return "", errors.New("pas de profil")
		case "ditto":
			dst := c.args[1]
			os.MkdirAll(filepath.Join(dst, "Contents"), 0o755)
			return "", os.WriteFile(filepath.Join(dst, "Contents", "Info.plist"), []byte("new"), 0o644)
		case "git":
			return "tree123\n", nil
		}
		return "", nil
	}
}

func newTestUpdater(t *testing.T, r *fakeRunner) *Updater {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "Document on Mac", "vinyl") // espace dans le chemin, comme chez Pierre
	os.MkdirAll(filepath.Join(project, "widget"), 0o755)
	os.WriteFile(filepath.Join(project, "widget", "project.yml"), []byte("name: VinylWidget\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	os.WriteFile(filepath.Join(root, "bin", "xcodegen"), nil, 0o755)
	os.MkdirAll(filepath.Join(root, "Applications"), 0o755)
	return &Updater{
		ProjectDir:    project,
		AppPath:       filepath.Join(root, "Applications", "Vinyl Widget.app"),
		BuildDir:      filepath.Join(root, "build"),
		RecordPath:    filepath.Join(root, "widget-install.json"),
		XcodeGenPaths: []string{filepath.Join(root, "absent", "xcodegen"), filepath.Join(root, "bin", "xcodegen")},
		Run:           r,
		Now:           func() time.Time { return testNow },
	}
}

func installOld(t *testing.T, u *Updater) {
	t.Helper()
	os.MkdirAll(filepath.Join(u.AppPath, "Contents"), 0o755)
	os.WriteFile(filepath.Join(u.AppPath, "Contents", "Info.plist"), []byte("old"), 0o644)
}

func readInfo(t *testing.T, u *Updater) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(u.AppPath, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdateSuccess(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))}
	u := newTestUpdater(t, r)
	installOld(t, u)

	if err := u.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := readInfo(t, u); got != "new" {
		t.Fatalf("app installée = %q, want new", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(u.AppPath))
	if len(entries) != 1 {
		t.Fatalf("fichiers temporaires restants : %v", entries)
	}
	var rec struct{ Tree string }
	b, _ := os.ReadFile(u.RecordPath)
	json.Unmarshal(b, &rec)
	if rec.Tree != "tree123" {
		t.Fatalf("empreinte enregistrée = %q", rec.Tree)
	}
	want := []string{"security", "xcodegen", "xcodebuild", "killall", "ditto", "open", "git"}
	if !slices.Equal(r.names(), want) {
		t.Fatalf("commandes = %v, want %v", r.names(), want)
	}
	gen, build := r.calls[1], r.calls[2]
	if gen.name != u.XcodeGenPaths[1] || gen.dir != filepath.Join(u.ProjectDir, "widget") {
		t.Fatalf("xcodegen = %+v", gen)
	}
	for _, a := range []string{"DEVELOPMENT_TEAM=TEAM123", "-allowProvisioningUpdates", "Vinyl Widget", "Release", u.BuildDir} {
		if !slices.Contains(build.args, a) {
			t.Errorf("xcodebuild sans %q : %v", a, build.args)
		}
	}
	if build.dir != filepath.Join(u.ProjectDir, "widget") {
		t.Errorf("xcodebuild lancé dans %q", build.dir)
	}
}

func TestUpdateMissingProject(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	u.ProjectDir = filepath.Join(u.ProjectDir, "ailleurs")
	err := u.Update(context.Background())
	if !strings.HasPrefix(ErrorMessage(err), "Dossier du projet introuvable : ") {
		t.Fatalf("message = %q", ErrorMessage(err))
	}
	if len(r.calls) != 0 {
		t.Fatalf("aucune commande attendue : %v", r.names())
	}
}

func TestUpdateMissingXcodeGen(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	u.XcodeGenPaths = u.XcodeGenPaths[:1]
	if got := ErrorMessage(u.Update(context.Background())); got != "Installe XcodeGen : brew install xcodegen" {
		t.Fatalf("message = %q", got)
	}
}

func TestUpdateNoCertificate(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	if got := ErrorMessage(u.Update(context.Background())); got != msgAccount {
		t.Fatalf("message = %q", got)
	}
	if slices.Contains(r.names(), "xcodebuild") {
		t.Fatal("xcodebuild ne doit pas être lancé sans équipe")
	}
}

func TestUpdateBuildFails(t *testing.T) {
	cert := makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300))
	cases := []struct {
		name, out string
		check     func(msg string) bool
	}{
		{"compte", `error: No Account for Team "TEAM123"`, func(m string) bool { return m == msgAccount }},
		{"autre", strings.Repeat("ligne\n", 10) + func() string {
			var b strings.Builder
			for i := 1; i <= 30; i++ {
				fmt.Fprintf(&b, "erreur %d\n", i)
			}
			return b.String()
		}(), func(m string) bool {
			return strings.HasPrefix(m, "La compilation a échoué") && strings.Contains(m, "erreur 30") && !strings.Contains(m, "erreur 10\n")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := defaultRespond(cert)
			r := &fakeRunner{respond: func(cl call) (string, error) {
				if filepath.Base(cl.name) == "xcodebuild" {
					return c.out, errors.New("exit status 65")
				}
				return base(cl)
			}}
			u := newTestUpdater(t, r)
			installOld(t, u)
			msg := ErrorMessage(u.Update(context.Background()))
			if !c.check(msg) {
				t.Fatalf("message = %q", msg)
			}
			if readInfo(t, u) != "old" {
				t.Fatal("l'ancienne version doit rester en place")
			}
		})
	}
}

func TestUpdateTimeout(t *testing.T) {
	base := defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))
	r := &fakeRunner{respond: func(c call) (string, error) {
		if filepath.Base(c.name) == "xcodebuild" {
			return "", context.DeadlineExceeded
		}
		return base(c)
	}}
	u := newTestUpdater(t, r)
	if got := ErrorMessage(u.Update(context.Background())); got != "La mise à jour a pris trop de temps" {
		t.Fatalf("message = %q", got)
	}
}

func TestUpdateCopyFailsKeepsOldApp(t *testing.T) {
	base := defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))
	r := &fakeRunner{respond: func(c call) (string, error) {
		if filepath.Base(c.name) == "ditto" {
			return "ditto: Permission denied", errors.New("exit status 1")
		}
		return base(c)
	}}
	u := newTestUpdater(t, r)
	installOld(t, u)
	if err := u.Update(context.Background()); err == nil {
		t.Fatal("erreur attendue")
	}
	if readInfo(t, u) != "old" {
		t.Fatal("l'ancienne version doit rester en place")
	}
}

func TestObserve(t *testing.T) {
	r := &fakeRunner{respond: func(c call) (string, error) {
		switch filepath.Base(c.name) {
		case "security":
			return "<key>ExpirationDate</key>\n<date>2026-10-17T08:00:00Z</date>", nil
		case "git":
			return "tree456\n", nil
		}
		return "", nil
	}}
	u := newTestUpdater(t, r)
	installOld(t, u)
	os.WriteFile(u.RecordPath, []byte(`{"tree":"tree123","installedAt":"2026-10-10T08:00:00Z"}`), 0o644)

	o := u.Observe(context.Background())
	want := Observed{Installed: true, Expiry: time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC),
		SourceTree: "tree456", InstalledTree: "tree123", Now: testNow}
	if o != want {
		t.Fatalf("Observe = %+v, want %+v", o, want)
	}
	if got := r.calls[0].args[len(r.calls[0].args)-1]; got != filepath.Join(u.AppPath, "Contents", "embedded.provisionprofile") {
		t.Fatalf("profil lu = %q", got)
	}
}

func TestObserveNotInstalled(t *testing.T) {
	r := &fakeRunner{respond: func(c call) (string, error) { return "", errors.New("introuvable") }}
	u := newTestUpdater(t, r)
	if o := u.Observe(context.Background()); o.Installed || o.SourceTree != "" || !o.Expiry.IsZero() {
		t.Fatalf("Observe = %+v", o)
	}
}
```

- [ ] **Step 2: Lancer les tests pour vérifier qu'ils échouent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test ./internal/widgetupdate/`
Expected: FAIL (`undefined: Updater`).

- [ ] **Step 3: Écrire `internal/widgetupdate/updater.go`**

```go
package widgetupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner exécute une commande et renvoie sa sortie combinée (stdout + stderr).
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if ctx.Err() != nil {
		return out.String(), ctx.Err()
	}
	return out.String(), err
}

// UserError porte un message prêt à afficher.
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func userErr(format string, a ...any) error { return &UserError{Msg: fmt.Sprintf(format, a...)} }

const msgAccount = "Connecte ton compte Apple dans Xcode → Réglages → Comptes, puis installe une fois le widget depuis Xcode (voir widget/README.md)"

// ErrorMessage traduit une erreur de mise à jour en texte pour l'interface.
func ErrorMessage(err error) string {
	var ue *UserError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &ue):
		return ue.Msg
	case errors.Is(err, context.DeadlineExceeded):
		return "La mise à jour a pris trop de temps"
	default:
		return err.Error()
	}
}

type Updater struct {
	ProjectDir    string   // racine du dépôt Vinyl (contient widget/)
	AppPath       string   // /Applications/Vinyl Widget.app
	BuildDir      string   // -derivedDataPath
	RecordPath    string   // widget-install.json
	XcodeGenPaths []string // emplacements possibles de xcodegen
	Run           Runner
	Now           func() time.Time
}

type record struct {
	Tree        string    `json:"tree"`
	InstalledAt time.Time `json:"installedAt"`
}

func (u *Updater) widgetDir() string { return filepath.Join(u.ProjectDir, "widget") }

func (u *Updater) sourceTree(ctx context.Context) string {
	out, err := u.Run.Run(ctx, "", "/usr/bin/git", "-C", u.ProjectDir, "rev-parse", "HEAD:widget")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Observe relève l'état sur le disque, sans rien modifier.
func (u *Updater) Observe(ctx context.Context) Observed {
	o := Observed{Now: u.Now()}
	if _, err := os.Stat(u.AppPath); err == nil {
		o.Installed = true
		profile := filepath.Join(u.AppPath, "Contents", "embedded.provisionprofile")
		if out, err := u.Run.Run(ctx, "", "/usr/bin/security", "cms", "-D", "-i", profile); err == nil {
			if t, ok := ParseExpiry(out); ok {
				o.Expiry = t
			}
		}
	}
	o.SourceTree = u.sourceTree(ctx)
	var rec record
	if b, err := os.ReadFile(u.RecordPath); err == nil && json.Unmarshal(b, &rec) == nil {
		o.InstalledTree = rec.Tree
	}
	return o
}

// Update recompile le widget et remplace l'app installée. L'ancienne version reste en place en cas d'échec.
func (u *Updater) Update(ctx context.Context) error {
	if fi, err := os.Stat(filepath.Join(u.widgetDir(), "project.yml")); err != nil || fi.IsDir() {
		return userErr("Dossier du projet introuvable : %s. Corrige widgetProjectDir dans settings.json", u.ProjectDir)
	}
	xcodegen := ""
	for _, p := range u.XcodeGenPaths {
		if _, err := os.Stat(p); err == nil {
			xcodegen = p
			break
		}
	}
	if xcodegen == "" {
		return userErr("Installe XcodeGen : brew install xcodegen")
	}
	pems, _ := u.Run.Run(ctx, "", "/usr/bin/security", "find-certificate", "-a", "-c", "Apple Development", "-p")
	team, ok := TeamFromPEM([]byte(pems), u.Now())
	if !ok {
		return userErr(msgAccount)
	}
	if out, err := u.Run.Run(ctx, u.widgetDir(), xcodegen, "generate"); err != nil {
		return failure(err, "La génération du projet a échoué", out)
	}
	out, err := u.Run.Run(ctx, u.widgetDir(), "/usr/bin/xcodebuild",
		"-project", "VinylWidget.xcodeproj", "-scheme", "Vinyl Widget", "-configuration", "Release",
		"-derivedDataPath", u.BuildDir, "-allowProvisioningUpdates", "DEVELOPMENT_TEAM="+team, "build")
	if err != nil {
		if isAccountProblem(out) {
			return userErr(msgAccount)
		}
		return failure(err, "La compilation a échoué", out)
	}
	built := filepath.Join(u.BuildDir, "Build", "Products", "Release", "Vinyl Widget.app")
	u.Run.Run(ctx, "", "/usr/bin/killall", "Vinyl Widget") // erreur ignorée : l'assistant peut être arrêté
	if err := u.install(ctx, built); err != nil {
		return err
	}
	if out, err := u.Run.Run(ctx, "", "/usr/bin/open", "-a", u.AppPath); err != nil {
		return failure(err, "Widget installé, mais impossible de relancer l'assistant", out)
	}
	return u.saveRecord(ctx)
}

// install copie l'app compilée à côté de l'ancienne puis échange les deux par renommage.
func (u *Updater) install(ctx context.Context, built string) error {
	dir, base := filepath.Dir(u.AppPath), filepath.Base(u.AppPath)
	tmp, old := filepath.Join(dir, "."+base+".new"), filepath.Join(dir, "."+base+".old")
	os.RemoveAll(tmp)
	os.RemoveAll(old)
	if out, err := u.Run.Run(ctx, "", "/usr/bin/ditto", built, tmp); err != nil {
		os.RemoveAll(tmp)
		return failure(err, "Copie impossible vers "+dir, out)
	}
	_, statErr := os.Stat(u.AppPath)
	hadOld := statErr == nil
	if hadOld {
		if err := os.Rename(u.AppPath, old); err != nil {
			os.RemoveAll(tmp)
			return userErr("Remplacement impossible de %s : %v", u.AppPath, err)
		}
	}
	if err := os.Rename(tmp, u.AppPath); err != nil {
		if hadOld {
			os.Rename(old, u.AppPath)
		}
		os.RemoveAll(tmp)
		return userErr("Remplacement impossible de %s : %v", u.AppPath, err)
	}
	os.RemoveAll(old)
	return nil
}

func (u *Updater) saveRecord(ctx context.Context) error {
	b, _ := json.Marshal(record{Tree: u.sourceTree(ctx), InstalledAt: u.Now()})
	if err := os.MkdirAll(filepath.Dir(u.RecordPath), 0o755); err != nil {
		return err
	}
	tmp := u.RecordPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, u.RecordPath)
}

func isAccountProblem(out string) bool {
	low := strings.ToLower(out)
	for _, k := range []string{"no account", "no accounts", "requires a development team", "no signing certificate", "not signed in"} {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// failure garde l'erreur de délai telle quelle (traduite par ErrorMessage), sinon ajoute la fin de la sortie.
func failure(err error, what, out string) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	return userErr("%s :\n%s", what, lastLines(out, 20))
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils passent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test ./internal/widgetupdate/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/widgetupdate
git commit -m "feat(widgetupdate): recompilation et remplacement sûr du widget installé

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: État de la mise à jour, service et branchement dans Vinyl

**Files:**
- Create: `internal/widgetupdate/manager.go`
- Test: `internal/widgetupdate/manager_test.go`
- Modify: `internal/settings/settings.go` (champ `WidgetProjectDir`), `internal/settings/settings_test.go`
- Modify: `internal/app/service.go` (`SetWidgetUpdates`, `WidgetUpdateStatus`, `UpdateWidget`)
- Modify: `main.go` (événement, `projectDir`, création du Manager)
- Modify: `build/darwin/Taskfile.yml:47` (ldflags)

**Interfaces:**
- Consumes : `Updater`, `Detect`, `Message`, `ErrorMessage`, `Observed` (Tasks 8-9).
- Produces :
  - `type Status struct { State, Reason, Message string }`, sérialisé en JSON avec les clés `state`, `reason` et `message`. Valeurs de `State` : `StateNone = "none"`, `StateAvailable = "available"`, `StateRunning = "running"`, `StateError = "error"` ;
  - `type Engine interface { Observe(context.Context) Observed; Update(context.Context) error }` ;
  - `func NewManager(e Engine, emit func(Status)) *Manager` ;
  - méthodes de `Manager` : `Status()`, `Check(ctx)`, `Start() error`, `Run(ctx, every)` ;
  - `var ErrBusy` ;
  - côté service : `VinylService.WidgetUpdateStatus() widgetupdate.Status` et `VinylService.UpdateWidget() error` ;
  - événement Wails `widgetupdate` (`widgetupdate.Status`).

- [ ] **Step 1: Écrire les tests du Manager**

`internal/widgetupdate/manager_test.go` :

```go
package widgetupdate

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeEngine struct {
	observed Observed
	update   func(ctx context.Context) error
}

func (f *fakeEngine) Observe(context.Context) Observed { return f.observed }
func (f *fakeEngine) Update(ctx context.Context) error { return f.update(ctx) }

func collect() (func(Status), <-chan Status) {
	ch := make(chan Status, 16)
	return func(s Status) { ch <- s }, ch
}

func next(t *testing.T, ch <-chan Status) Status {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("aucun état émis")
		return Status{}
	}
}

func TestCheckEmitsOnChangeOnly(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{observed: Observed{Installed: true, Expiry: testNow.Add(-time.Hour), Now: testNow}}
	m := NewManager(e, emit)
	m.Check(context.Background())
	if s := next(t, ch); s != (Status{State: StateAvailable, Reason: "expired", Message: "Signature du widget expirée"}) {
		t.Fatalf("état = %+v", s)
	}
	m.Check(context.Background())
	if len(ch) != 0 {
		t.Fatal("pas de nouvel événement si rien ne change")
	}
	e.observed = Observed{Installed: true, Now: testNow}
	m.Check(context.Background())
	if s := next(t, ch); s.State != StateNone {
		t.Fatalf("état = %+v", s)
	}
}

func TestStartSuccess(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{observed: Observed{Installed: true, Now: testNow}, update: func(context.Context) error { return nil }}
	m := NewManager(e, emit)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if s := next(t, ch); s.State != StateRunning {
		t.Fatalf("état = %+v", s)
	}
	if s := next(t, ch); s.State != StateNone {
		t.Fatalf("état = %+v", s)
	}
}

func TestStartBusy(t *testing.T) {
	emit, ch := collect()
	release := make(chan struct{})
	e := &fakeEngine{update: func(context.Context) error { <-release; return nil }}
	m := NewManager(e, emit)
	m.Start()
	next(t, ch)
	if err := m.Start(); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	m.Check(context.Background())
	if m.Status().State != StateRunning {
		t.Fatal("Check ne doit pas écraser une mise à jour en cours")
	}
	close(release)
	next(t, ch)
}

func TestErrorPersistsWhileReasonsRemain(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{
		observed: Observed{Installed: true, Expiry: testNow.Add(-time.Hour), Now: testNow},
		update:   func(context.Context) error { return &UserError{Msg: "Installe XcodeGen : brew install xcodegen"} },
	}
	m := NewManager(e, emit)
	m.Start()
	next(t, ch)
	if s := next(t, ch); s != (Status{State: StateError, Message: "Installe XcodeGen : brew install xcodegen"}) {
		t.Fatalf("état = %+v", s)
	}
	m.Check(context.Background())
	if m.Status().State != StateError {
		t.Fatal("l'erreur doit rester affichée tant que la mise à jour est nécessaire")
	}
}
```

- [ ] **Step 2: Lancer les tests pour vérifier qu'ils échouent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test ./internal/widgetupdate/`
Expected: FAIL (`undefined: NewManager`).

- [ ] **Step 3: Écrire `internal/widgetupdate/manager.go`**

```go
package widgetupdate

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	StateNone      = "none"
	StateAvailable = "available"
	StateRunning   = "running"
	StateError     = "error"
)

// Status est envoyé au frontend (événement « widgetupdate »).
type Status struct {
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type Engine interface {
	Observe(ctx context.Context) Observed
	Update(ctx context.Context) error
}

var ErrBusy = errors.New("mise à jour du widget déjà en cours")

const updateTimeout = 10 * time.Minute

type Manager struct {
	engine Engine
	emit   func(Status)
	mu     sync.Mutex
	status Status
}

func NewManager(e Engine, emit func(Status)) *Manager {
	return &Manager{engine: e, emit: emit, status: Status{State: StateNone}}
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// set change l'état et l'émet s'il a changé. keep permet de refuser le changement sous verrou.
func (m *Manager) set(s Status, keep func(cur Status) bool) {
	m.mu.Lock()
	if (keep != nil && keep(m.status)) || s == m.status {
		m.mu.Unlock()
		return
	}
	m.status = s
	m.mu.Unlock()
	m.emit(s)
}

// Check relève l'état du widget. Ne touche ni à une mise à jour en cours, ni à une erreur encore pertinente.
func (m *Manager) Check(ctx context.Context) {
	o := m.engine.Observe(ctx)
	reasons := Detect(o)
	next := Status{State: StateNone}
	if len(reasons) > 0 {
		next = Status{State: StateAvailable, Reason: string(reasons[0]), Message: Message(reasons[0], o.Expiry, o.Now)}
	}
	m.set(next, func(cur Status) bool {
		return cur.State == StateRunning || (cur.State == StateError && len(reasons) > 0)
	})
}

// Start lance la mise à jour en arrière-plan.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.status.State == StateRunning {
		m.mu.Unlock()
		return ErrBusy
	}
	s := Status{State: StateRunning, Message: "Mise à jour du widget…"}
	m.status = s
	m.mu.Unlock()
	m.emit(s)
	go m.run()
	return nil
}

func (m *Manager) run() {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	if err := m.engine.Update(ctx); err != nil {
		m.set(Status{State: StateError, Message: ErrorMessage(err)}, nil)
		return
	}
	m.set(Status{State: StateNone}, nil)
	m.Check(context.Background())
}

// Run vérifie tout de suite, puis à chaque intervalle, jusqu'à l'annulation de ctx.
func (m *Manager) Run(ctx context.Context, every time.Duration) {
	m.Check(ctx)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.Check(ctx)
		}
	}
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils passent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test -race ./internal/widgetupdate/ -v`
Expected: PASS, sans data race.

- [ ] **Step 5: Ajouter `widgetProjectDir` aux réglages (test d'abord)**

Ajouter à `internal/settings/settings_test.go` :

```go
func TestWidgetProjectDirIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"widgetProjectDir":"/Users/pierre/Document on Mac/projet_git/vinyl"}`), 0o644)
	s := NewStore(path)
	s.Load()
	if got := s.Get().WidgetProjectDir; got != "/Users/pierre/Document on Mac/projet_git/vinyl" {
		t.Fatalf("WidgetProjectDir = %q", got)
	}
	accent := "#ff6b6b"
	if _, err := s.Update(Patch{Accent: &accent}); err != nil {
		t.Fatal(err)
	}
	s2 := NewStore(path)
	s2.Load()
	if s2.Get().WidgetProjectDir == "" {
		t.Fatal("une mise à jour partielle ne doit pas effacer widgetProjectDir")
	}
}
```

Run: `CGO_ENABLED=1 go test ./internal/settings/` → FAIL (`s.Get().WidgetProjectDir undefined`).

Puis, dans `internal/settings/settings.go`, ajouter à la fin de `Settings` :

```go
	WidgetProjectDir string `json:"widgetProjectDir,omitempty"` // dossier du dépôt (widget/), facultatif
```

Run: `CGO_ENABLED=1 go test ./internal/settings/` → PASS.

- [ ] **Step 6: Exposer la mise à jour dans le service**

Dans `internal/app/service.go` :
- ajouter l'import `"vinyl/internal/widgetupdate"` ;
- ajouter le champ `widget *widgetupdate.Manager` à `VinylService` ;
- ajouter à la fin du fichier :

```go
// SetWidgetUpdates branche le gestionnaire de mise à jour du widget de bureau.
func (s *VinylService) SetWidgetUpdates(m *widgetupdate.Manager) { s.widget = m }

// WidgetUpdateStatus indique si l'icône ☁️↓ doit s'afficher.
func (s *VinylService) WidgetUpdateStatus() widgetupdate.Status {
	if s.widget == nil {
		return widgetupdate.Status{State: widgetupdate.StateNone}
	}
	return s.widget.Status()
}

// UpdateWidget lance la recompilation et la réinstallation du widget en arrière-plan.
func (s *VinylService) UpdateWidget() error {
	if s.widget == nil {
		return errors.New("mise à jour du widget indisponible")
	}
	return s.widget.Start()
}
```

- [ ] **Step 7: Brancher dans `main.go`**

1. Ajouter l'import `"vinyl/internal/widgetupdate"`.
2. Sous la déclaration de `assets`, ajouter :

```go
// projectDir est la racine du dépôt, inscrite à la compilation (-ldflags "-X 'main.projectDir=…'").
var projectDir string
```

3. Dans `init()`, ajouter :

```go
	application.RegisterEvent[widgetupdate.Status]("widgetupdate")
```

4. Juste après `cfg := store.Get()`, ajouter :

```go
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
```

5. Après `go watcher.Run(ctx, 500*time.Millisecond)`, ajouter :

```go
	go widgetMgr.Run(ctx, 6*time.Hour)
```

- [ ] **Step 8: Inscrire le chemin du dépôt à la compilation**

Dans `build/darwin/Taskfile.yml`, ligne 47 (`BUILD_FLAGS` de `build:native`), faire deux remplacements dans la chaîne entre apostrophes. L'apostrophe est doublée (`''`) car la valeur YAML est elle-même entre apostrophes.
- branche dev : remplacer `-gcflags=all="-l"` par `-gcflags=all="-l" -ldflags="-X ''main.projectDir={{.ROOT_DIR}}''"` ;
- branche production : remplacer `-ldflags="-w -s"` par `-ldflags="-w -s -X ''main.projectDir={{.ROOT_DIR}}''"`.

- [ ] **Step 9: Vérifier**

Run :

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go vet . ./internal/... && \
CGO_ENABLED=1 go test . ./internal/... && \
CGO_ENABLED=1 wails3 package && strings bin/vinyl.app/Contents/MacOS/vinyl | grep -c "Document on Mac/projet_git/vinyl"
```

Expected : tests PASS, `wails3 package` réussit, et le `grep -c` affiche au moins `1` (le chemin avec espaces est inscrit en entier).

- [ ] **Step 10: Commit**

```bash
git add internal/widgetupdate internal/settings internal/app/service.go main.go build/darwin/Taskfile.yml
git commit -m "feat(widgetupdate): état de la mise à jour exposé à l'interface, vérifié toutes les 6 h

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: Icône ☁️↓ en haut à droite de la fenêtre de Vinyl

**Files:**
- Create: `frontend/src/widgetupdate.ts`
- Test: `frontend/src/widgetupdate.test.ts`
- Modify: `frontend/index.html`, `frontend/src/api.ts`, `frontend/src/main.ts`
- Regenerate: `frontend/bindings/` (via `wails3 generate bindings`)

**Interfaces:**
- Consumes : `VinylService.WidgetUpdateStatus()`, `VinylService.UpdateWidget()` et l'événement `widgetupdate` (Task 10).
- Produces :
  - `interface WidgetStatus { state: string; reason: string; message: string }` ;
  - `function render(el: HTMLElement, s: WidgetStatus, actions: { update: () => void }): void` ;
  - `onWidgetUpdate(cb)` dans `api.ts`.

- [ ] **Step 1: Écrire les tests**

`frontend/src/widgetupdate.test.ts` :

```ts
import { beforeEach, describe, expect, test, vi } from "vitest";
import { render } from "./widgetupdate";

describe("icône de mise à jour du widget", () => {
  let el: HTMLElement;
  beforeEach(() => {
    el = document.createElement("div");
  });

  test("rien à faire : masquée", () => {
    render(el, { state: "none", reason: "", message: "" }, { update: () => {} });
    expect(el.hidden).toBe(true);
    expect(el.querySelector("button")).toBeNull();
  });

  test("mise à jour disponible : nuage, bulle et clic", () => {
    const update = vi.fn();
    render(el, { state: "available", reason: "expired", message: "Signature du widget expirée" }, { update });
    const btn = el.querySelector("button")!;
    expect(el.hidden).toBe(false);
    expect(btn.title).toBe("Signature du widget expirée");
    expect(btn.querySelector("svg")).not.toBeNull();
    btn.click();
    expect(update).toHaveBeenCalledOnce();
  });

  test("en cours : l'icône tourne et ne réagit pas", () => {
    const update = vi.fn();
    render(el, { state: "running", reason: "", message: "Mise à jour du widget…" }, { update });
    const btn = el.querySelector("button")!;
    expect(btn.classList.contains("animate-spin")).toBe(true);
    expect(btn.disabled).toBe(true);
    btn.click();
    expect(update).not.toHaveBeenCalled();
  });

  test("erreur : ⚠️, bulle avec le message et « Réessayer »", () => {
    const update = vi.fn();
    render(el, { state: "error", reason: "", message: "Installe XcodeGen : brew install xcodegen" }, { update });
    const btn = el.querySelector("button")!;
    expect(btn.textContent).toBe("⚠️");
    btn.click();
    const pop = el.querySelector("[data-popover]")!;
    expect(pop.textContent).toContain("Installe XcodeGen : brew install xcodegen");
    pop.querySelector<HTMLButtonElement>("[data-retry]")!.click();
    expect(update).toHaveBeenCalledOnce();
    btn.click();
    expect(el.querySelector("[data-popover]")).toBeNull();
  });

  test("message hostile affiché comme du texte", () => {
    render(el, { state: "error", reason: "", message: "<img src=x onerror=alert(1)>" }, { update: () => {} });
    el.querySelector("button")!.click();
    expect(el.querySelector("img")).toBeNull();
  });
});
```

- [ ] **Step 2: Lancer les tests pour vérifier qu'ils échouent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl/frontend" && npx vitest run src/widgetupdate.test.ts`
Expected: FAIL (`Failed to resolve import "./widgetupdate"`).

- [ ] **Step 3: Écrire `frontend/src/widgetupdate.ts`**

```ts
// Icône ☁️↓ de mise à jour du widget de bureau (rendu pur, testable sans Wails).
export interface WidgetStatus {
  state: string; // none | available | running | error
  reason: string;
  message: string;
}

const CLOUD = `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
  stroke-linecap="round" stroke-linejoin="round"><path d="M7 18a4.5 4.5 0 0 1-.6-8.96A6 6 0 0 1 18 9.5a4 4 0 0 1-.5 8.5"/>
  <path d="M12 11v8m-3-3 3 3 3-3"/></svg>`;

export function render(el: HTMLElement, s: WidgetStatus, actions: { update: () => void }): void {
  el.replaceChildren();
  el.hidden = s.state === "none";
  if (el.hidden) return;

  const btn = document.createElement("button");
  btn.className = "flex h-8 w-8 items-center justify-center rounded-full bg-neutral-800/80 text-neutral-200 hover:bg-neutral-700";
  btn.dataset.state = s.state;
  if (s.state === "available") {
    btn.innerHTML = CLOUD;
    btn.title = s.message;
    btn.style.color = "var(--accent)";
    btn.onclick = actions.update;
  } else if (s.state === "running") {
    btn.innerHTML = CLOUD;
    btn.title = s.message || "Mise à jour du widget…";
    btn.classList.add("animate-spin");
    btn.disabled = true;
  } else {
    btn.textContent = "⚠️";
    btn.title = "La mise à jour du widget a échoué";
    btn.onclick = () => togglePopover(el, s.message, actions.update);
  }
  el.append(btn);
}

function togglePopover(el: HTMLElement, message: string, update: () => void): void {
  const open = el.querySelector("[data-popover]");
  if (open) {
    open.remove();
    return;
  }
  const pop = document.createElement("div");
  pop.dataset.popover = "";
  pop.className = "absolute right-0 top-10 w-80 rounded-lg bg-neutral-900 p-3 text-xs text-neutral-200 shadow-xl ring-1 ring-white/10";
  const msg = document.createElement("p");
  msg.className = "mb-3 whitespace-pre-wrap";
  msg.textContent = message;
  const retry = document.createElement("button");
  retry.dataset.retry = "";
  retry.className = "rounded px-3 py-1 font-semibold text-neutral-900";
  retry.style.background = "var(--accent)";
  retry.textContent = "Réessayer";
  retry.onclick = update;
  pop.append(msg, retry);
  el.append(pop);
}
```

- [ ] **Step 4: Lancer les tests pour vérifier qu'ils passent**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl/frontend" && npx vitest run src/widgetupdate.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Brancher l'icône dans la fenêtre principale**

Dans `frontend/index.html`, juste après `<div class="drag fixed left-0 top-0 z-40 h-12 w-full"></div>`, ajouter :

```html
  <div id="widget-update" class="no-drag fixed right-4 top-2 z-50" hidden></div>
```

Dans `frontend/src/api.ts`, ajouter l'import en tête puis la fonction après `onSettings` :

```ts
import type { WidgetStatus } from "./widgetupdate";
```

```ts
export function onWidgetUpdate(cb: (s: WidgetStatus) => void): () => void {
  return Events.On("widgetupdate", (ev) => cb(ev.data as WidgetStatus));
}
```

Dans `frontend/src/main.ts` :
- remplacer l'import d'`api` par `import { api, applyAccent, onSettings, onWidgetUpdate, type Settings } from "./api";` ;
- ajouter `import { render as renderWidgetUpdate, type WidgetStatus } from "./widgetupdate";` ;
- ajouter à la fin du fichier :

```ts
const widgetUpdateEl = document.getElementById("widget-update")!;
const showWidgetUpdate = (s: WidgetStatus) =>
  renderWidgetUpdate(widgetUpdateEl, s, { update: () => void api.UpdateWidget().catch(console.error) });
api.WidgetUpdateStatus().then(showWidgetUpdate);
onWidgetUpdate(showWidgetUpdate);
```

- [ ] **Step 6: Régénérer les bindings et vérifier**

Run :

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 wails3 generate bindings -clean=true -ts -i && \
grep -c "WidgetUpdateStatus\|UpdateWidget" frontend/bindings/vinyl/internal/app/vinylservice.ts && \
cd frontend && npx tsc --noEmit && npm test
```

Expected : le `grep -c` affiche au moins `2`, `tsc` ne signale aucune erreur, et tous les tests Vitest passent.

- [ ] **Step 7: Vérifier dans l'app**

Run: `cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 wails3 dev`
Expected :
- si `/Applications/Vinyl Widget.app` n'existe pas, l'icône ☁️↓ s'affiche en haut à droite avec la bulle « Widget pas encore installé » ;
- elle ne gêne pas le déplacement de la fenêtre par sa barre de titre.

Ne pas cliquer : l'installation se teste dans la Task 12.

- [ ] **Step 8: Commit**

```bash
git add frontend/index.html frontend/src/widgetupdate.ts frontend/src/widgetupdate.test.ts frontend/src/api.ts frontend/src/main.ts frontend/bindings
git commit -m "feat(fenêtre): icône ☁️↓ de mise à jour du widget en haut à droite

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: Build final et recette

**Files:** aucun nouveau fichier. Corrections éventuelles dans les fichiers des tâches précédentes.

- [ ] **Step 1: Tous les tests**

Run :

```bash
cd "/Users/pierre/Document on Mac/projet_git/vinyl" && CGO_ENABLED=1 go test . ./internal/... && \
(cd frontend && npm test) && \
(cd widget && xcodegen generate && xcodebuild test -project VinylWidget.xcodeproj -scheme VinylWidgetTests -destination 'platform=macOS' CODE_SIGNING_ALLOWED=NO 2>&1 | tail -3)
```

Expected : tout passe (`ok …`, Vitest vert, `** TEST SUCCEEDED **`).

- [ ] **Step 2: Construire Vinyl**

Run: `CGO_ENABLED=1 wails3 package`, puis copier `bin/vinyl.app` dans `/Applications`, en remplaçant l'ancienne version.

- [ ] **Step 3: 👤 Pierre — recette de la mise à jour depuis Vinyl**

Prérequis : Task 7 Step 3 faite (compte Apple dans Xcode, première installation).
- [ ] Lancer Vinyl : la fenêtre principale s'ouvre, sans disque flottant ; ⌘⇧M l'affiche.
- [ ] Supprimer `~/Library/Application Support/Vinyl/widget-install.json` puis relancer Vinyl : ☁️↓ « Nouvelle version du widget ».
- [ ] Clic sur ☁️↓ : l'icône tourne, puis disparaît (1 à 3 min). Le widget fonctionne toujours.
- [ ] Modifier un texte dans `widget/` (par exemple la description du widget), committer, puis attendre la vérification suivante ou relancer Vinyl : ☁️↓ réapparaît.
- [ ] Xcode → Réglages → Comptes : retirer le compte, puis cliquer ☁️↓ : ⚠️ avec le message « Connecte ton compte Apple… ». Remettre le compte, puis « Réessayer » : succès.
- [ ] Recette du widget : liste de `widget/README.md`.

- [ ] **Step 4: Commit des corrections éventuelles**

```bash
git add -A
git commit -m "fix(widget): corrections issues de la recette

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

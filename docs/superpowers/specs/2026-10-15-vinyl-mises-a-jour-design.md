# Mises à jour de Vinyl dans l'app — conception

Date : 2026-10-15 · Branche : `feat/mises-a-jour`, créée depuis `main`

## But

Quand une nouvelle version de Vinyl est publiée (release GitHub), l'app le dit et propose de se mettre à
jour : un clic télécharge la nouvelle version, l'installe et redémarre Vinyl, sur Mac et sur Windows.
Les versions bêta (tags avec un tiret, ex. `v1.1.0-beta.1`) ne sont proposées qu'aux contributeurs à qui
l'auteur a donné un code d'accès personnel.

Réussi quand :

- une app installée depuis la release `vX` affiche « Vinyl Y est disponible » dès qu'une release stable
  `vY > vX` existe, et « Mettre à jour » aboutit à Vinyl `Y` relancé, sur Mac et sur Windows ;
- sans code contributeur, une bêta n'est jamais proposée ; avec un code valide, la bêta la plus récente
  l'est aussi ;
- un code contributeur ne peut pas être fabriqué sans la clé privée de l'auteur ;
- une version fabriquée à la main (« dev ») ne propose jamais de mise à jour ;
- en cas d'échec (réseau, empreinte, droits), l'app actuelle reste intacte et un lien vers la release est
  proposé.

## Décisions (validées avec l'utilisateur)

- Télécharger et installer (pas seulement prévenir).
- Bêtas réservées aux contributeurs, par **code personnel signé**.
- Approche : le système de mise à jour intégré à Wails v3 (`pkg/updater`, `app.Updater`) avec sa source
  GitHub (`pkg/updater/providers/github`) — pas de système écrit pour Vinyl.

## Version de l'app

- Variable `main.version` (défaut `"dev"`), injectée par la release : `-ldflags "-X main.version=1.0.3"`
  (tag sans le `v`). Le numéro du `Info.plist` (Mac) et de `info.json` (Windows) suit aussi le tag.
- `VinylService.Version()` la renvoie au frontend (affichée dans Réglages : « Vinyl 1.0.3 »).
- `version == "dev"` : le système de mise à jour n'est pas initialisé.

## Vérification et installation (Wails `updater`)

- Initialisation au démarrage : `CurrentVersion: version`, `CheckInterval: 6h`, `Window: updater.WindowNone`
  (l'interface est celle de Vinyl, pas la fenêtre de Wails), une seule source : `channelProvider`.
- `channelProvider` (Vinyl) : enveloppe deux sources GitHub (`Repository: "Peuraud-Pierre/vinyl"`,
  `ChecksumAsset: "SHA256SUMS"`), l'une stable (`Prerelease: false`), l'autre avec bêtas
  (`Prerelease: true`) ; à chaque vérification, il interroge la source avec bêtas seulement si un code
  contributeur valide est enregistré. Un code saisi ou retiré compte dès la vérification suivante.
- Choix du fichier : correspondance par nom de Wails (système et architecture dans le nom ; installateurs
  et fichiers d'empreintes écartés) : `Vinyl-darwin-arm64.zip` (contient `vinyl.app`) et
  `Vinyl-windows-amd64.zip` (contient `vinyl.exe`).
- Vérification d'intégrité : empreinte SHA-256 du fichier, lue dans `SHA256SUMS` publié avec la release.
- Installation : Wails télécharge, vérifie, remplace l'app (le `.app` sur Mac, le `.exe` sur Windows) puis
  redémarre Vinyl ; en cas d'échec, l'app actuelle est conservée.
- Service : `UpdateStatus()` (état, version proposée, notes), `CheckForUpdate()`, `InstallUpdate()`
  (télécharge, installe, redémarre) ; les changements d'état sont diffusés au frontend (événement).

## Interface

- Bandeau discret en haut de la fenêtre (même emplacement que le bandeau du widget) : « Vinyl 1.0.3 est
  disponible » + « Mettre à jour » ; pendant le téléchargement : « Téléchargement… 42 % » ; échec :
  message + « Voir la release » (ouvre la page GitHub de la version).
- Réglages : ligne « Version » (« Vinyl 1.0.3 » + « Rechercher une mise à jour ») et section repliée
  « Contributeur » : champ du code ; une fois valide, « Bêtas activées pour Prénom » + « Retirer ».

## Code contributeur

- Format : `VINYL-` + base64url( nom ‖ signature Ed25519 du nom ). Le nom (prénom de la personne) sert à
  l'afficher.
- Vinyl contient la **clé publique** (`internal/beta/pubkey.go`) ; la **clé privée** reste sur le Mac de
  l'auteur, hors du dépôt (`~/.vinyl/beta-private.key`), créée une fois par le script.
- Script `./beta-code.sh "Prénom"` : crée la paire de clés si besoin (et met à jour la clé publique dans
  le code), puis affiche le code à donner.
- Stockage : le code est enregistré dans les réglages (`betaCode`) ; il est revérifié à chaque lancement.
- Limite assumée : un code ne se révoque pas (pas de serveur) ; les bêtas restent de toute façon
  téléchargeables à la main sur GitHub, le code ne décide que de ce que l'app propose.

## Release (workflow)

- Mac : `Vinyl-darwin-arm64.zip` (remplace `Vinyl-mac.zip`).
- Windows : `Vinyl-Setup.exe` (première installation) + `Vinyl-windows-amd64.zip` (mises à jour).
- `SHA256SUMS` : empreintes de tous les fichiers.
- Version injectée depuis le tag dans les deux fabrications.

## Erreurs

- Pas de réseau / GitHub indisponible : rien n'est affiché (nouvel essai à la vérification suivante).
- Échec du téléchargement ou empreinte invalide : message dans le bandeau + « Voir la release » ; l'app
  actuelle n'est pas touchée.
- Droits insuffisants pour remplacer l'app (ex. app installée par un autre compte) : même message.
- Code contributeur invalide : « Code invalide » sous le champ ; rien n'est enregistré.

## Tests

- `channelProvider` : sans code → source stable seulement ; avec code valide → source avec bêtas ; code
  retiré → retour au stable (sources factices).
- Code contributeur : un code signé valide est accepté et donne le nom ; un code modifié, tronqué, ou signé
  par une autre clé est refusé.
- `version == "dev"` : aucune vérification.
- Service : états diffusés (disponible, téléchargement, erreur) avec un `Updater` factice derrière une
  petite interface.
- Workflow : noms des fichiers produits compatibles avec la correspondance de Wails (test du nommage avec
  `github.DefaultAssetMatcher`).
- Vérification réelle : release de test `v1.0.3` puis `v1.0.4` sur Mac et sur le PC Windows (liste
  ajoutée à `docs/windows-verification.md`).

## Risques

- Le système de mise à jour de Wails est récent (Wails v3 bêta) : à valider en vrai sur les deux systèmes
  dès la première release ; repli possible : bandeau « Voir la release » seul.
- Mac : l'app n'est pas signée par Apple ; un remplacement par Wails n'ajoute pas de quarantaine (pas de
  téléchargement par navigateur), mais c'est à confirmer au premier essai.
- Les installations faites avec `Vinyl-mac.zip` / `Vinyl-Setup.exe` antérieurs à cette fonction n'ont pas
  la vérification : la première mise à jour vers une version qui l'a doit se faire à la main.

# Vérification de Vinyl sous Windows

Installer `Vinyl-Setup.exe` (artefact de la CI ou de la release), lancer Vinyl.

Avec **Spotify** (app de bureau ou du Microsoft Store), puis avec **Apple Music pour Windows** :

1. Lancer un morceau dans l'app : la platine affiche titre, artiste, pochette ; le disque tourne ; le bras avance.
2. Touches Vinyl : lecture/pause, suivant, précédent ; glisser le bras plus loin : la lecture avance.
3. Aléatoire : le témoin suit l'état de l'app.
4. Écouter trois morceaux : la pile se remplit (pochettes visibles).
5. Cliquer sur le 2ᵉ disque de la pile : il rejoue ; à sa fin, retour au morceau interrompu, à la bonne position.
6. Spotify seulement : étagère (albums, playlists), page d'un album, « Lecture » : la lecture démarre
   (Premium) ou l'album s'ouvre dans Spotify (sans Premium).
7. Apple Music : l'étagère affiche « La bibliothèque Apple Music n'est disponible que sur Mac. »
8. Fermer l'app : pas d'erreur ; désinstaller depuis Paramètres → Applications : Vinyl disparaît.

Pour chaque point qui échoue : noter ce qui s'affiche, et joindre une capture d'écran.

## Mises à jour (Mac et Windows)

0. **Avant de publier `vA`** : créer la clé des codes contributeur (`./beta-code.sh "Prénom"`) et commiter
   `internal/beta/pubkey.go`. Sans elle, tout code est refusé (« Code invalide ») dans cette version.
1. Installer à la main la release `vA` (la première qui contient les mises à jour) : `Vinyl-darwin-arm64.zip`
   sur Mac (glisser Vinyl dans Applications : lancée depuis Téléchargements, elle ne peut pas se remplacer),
   `Vinyl-Setup.exe` sur Windows (installée dans `%LOCALAPPDATA%\Programs\Vinyl`, sans droits
   d'administrateur ; désinstaller d'abord une ancienne version 1.0.x).
2. Réglages : la section Version affiche « Vinyl A ».
3. Publier `vB` (`./release.sh`) ; relancer Vinyl : le bandeau « Vinyl B est disponible » apparaît en haut.
4. « Mettre à jour » : « Téléchargement… N % », puis Vinyl se ferme et se relance ; Réglages affiche « Vinyl B ».
5. Publier une bêta `vC-beta.1` : sans code, « Rechercher une mise à jour » répond « Vinyl est à jour ».
   Coller un code (`./beta-code.sh "Prénom"`) dans Réglages > Contributeur : « Bêtas activées pour Prénom »,
   et la bêta est proposée. « Retirer » : elle ne l'est plus.
6. Couper le réseau et relancer : aucun message, l'app démarre normalement.
7. Bêta proposée puis « Retirer » dans Réglages : le bandeau disparaît ; « Mettre à jour » n'installe pas la bêta.
8. Mac : après une mise à jour, macOS peut redemander l'autorisation de piloter Musique (app non signée).

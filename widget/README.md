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

#!/bin/bash
# Publie une release de Vinyl : demande le numéro de version, pose le tag et le pousse sur GitHub.
# GitHub Actions (.github/workflows/release.yml) fabrique ensuite Vinyl-darwin-arm64.zip, Vinyl-Setup.exe,
# Vinyl-windows-amd64.zip et SHA256SUMS et les joint à la release. Un tag avec un tiret (v1.2.0-beta.1) est publié en pré-version.
#
# Utilisation : ./release.sh          (DRY_RUN=1 ./release.sh pour tout vérifier sans rien publier)
set -euo pipefail
cd "$(dirname "$0")"

REPO_URL="https://github.com/Peuraud-Pierre/vinyl"
run() { if [ "${DRY_RUN:-}" = "1" ]; then echo "  (essai) $*"; else "$@"; fi; }

branch=$(git branch --show-current)
if [ -z "$branch" ]; then
  echo "Tu n'es sur aucune branche (HEAD détaché) : place-toi sur une branche d'abord." >&2
  exit 1
fi
if [ ! -f .github/workflows/release.yml ]; then
  echo "La branche « $branch » n'a pas le workflow de release (.github/workflows/release.yml)." >&2
  echo "Place-toi sur une branche qui le contient (par exemple feat/windows)." >&2
  exit 1
fi
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo "Attention : des modifications ne sont pas commitées ; elles ne seront pas dans la release."
  read -r -p "Continuer quand même ? [o/N] " ok
  [[ "$ok" =~ ^[oOyY]$ ]] || exit 1
fi

git fetch --tags --quiet origin || true
last=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -1)
suggest=""
if [[ "$last" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
  suggest="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))"
fi

echo "Dernière version : ${last:-aucune}"
read -r -p "Version à publier${suggest:+ [$suggest]} : " tag
tag=${tag:-$suggest}
[[ "$tag" == v* ]] || tag="v$tag"
if ! [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  echo "Format attendu : v1.2.3 (ou v1.2.3-beta.1 pour une pré-version)." >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null || git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
  echo "Le tag $tag existe déjà : choisis un autre numéro." >&2
  exit 1
fi

read -r -p "Message de la release [Vinyl ${tag#v}] : " msg
msg=${msg:-"Vinyl ${tag#v}"}
kind="release"
[[ "$tag" == *-* ]] && kind="pré-version"

echo
echo "Publier $tag ($kind) depuis la branche « $branch » ($(git log -1 --format='%h %s')) ?"
read -r -p "[o/N] " ok
[[ "$ok" =~ ^[oOyY]$ ]] || { echo "Annulé."; exit 1; }

run git push origin "$branch"
run git tag -a "$tag" -m "$msg"
run git push origin "$tag"

echo
echo "C'est parti : GitHub fabrique la version Mac et la version Windows (quelques minutes)."
echo "Suivi :   $REPO_URL/actions"
echo "Release : $REPO_URL/releases/tag/$tag"

#!/bin/bash
# Fabrique un code contributeur (accès aux bêtas dans Vinyl) : ./beta-code.sh "Prénom"
# Clé privée : variable VINYL_BETA_PRIVATE_KEY ou fichier .env (créé au premier usage, ignoré par git).
set -euo pipefail
cd "$(dirname "$0")"
go run ./cmd/betacode "$@"

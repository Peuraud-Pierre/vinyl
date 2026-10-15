#!/bin/bash
# Fabrique un code contributeur (accès aux bêtas dans Vinyl) : ./beta-code.sh "Prénom"
set -euo pipefail
cd "$(dirname "$0")"
go run ./cmd/betacode "$@"

#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
export GOWORK=off
release_replacements=$(go list -m -f '{{if .Replace}}{{.Path}}{{end}}' all)
if [[ -n "$release_replacements" ]]; then
  echo "release: remove go.mod replacements before packaging" >&2
  exit 1
fi
go test ./...
go tool keel build "$@"

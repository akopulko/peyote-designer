#!/bin/sh
set -eu

RAW_TAG="${1:-}"

if [ -z "$RAW_TAG" ]; then
  echo "missing version tag" >&2
  exit 1
fi

RAW_TAG="${RAW_TAG#refs/tags/}"
RAW_TAG="${RAW_TAG#v}"

OLD_IFS="${IFS}"
IFS='.'
set -- $RAW_TAG
IFS="${OLD_IFS}"

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
  echo "unsupported version format: $RAW_TAG" >&2
  exit 1
fi

for part in "$@"; do
  case "$part" in
    ''|*[!0-9]*)
      echo "unsupported version format: $RAW_TAG" >&2
      exit 1
      ;;
  esac
done

case "$#" in
  1) echo "$1.0.0" ;;
  2) echo "$1.$2.0" ;;
  3) echo "$1.$2.$3" ;;
esac

#!/usr/bin/env bash
set -euo pipefail

version="$1"
os="$2"
arch="$3"
name="mihomo-${os}-${arch}"
dest="core/${name}"
# The plain amd64 build needs x86-64-v3 (AVX2): old Intel Macs and Rosetta lack it.
asset="$name"
[ "$arch" = amd64 ] && asset="${name}-compatible"

case "${version}/${name}" in
  v1.19.34/mihomo-darwin-arm64) want=a55a9073f9b7b1fa596a82c034d2dae3407ec34732102b6e42c5513ce3033f34 ;;
  v1.19.34/mihomo-darwin-amd64) want=3be524b2d34da2caaaac35dec0de3aa8c793add9994a4140dbac048543f7202d ;;
  *) want="" ;;
esac

mkdir -p core
tmp="$(mktemp core/.fetch.XXXXXX)"
trap 'rm -f "$tmp"' EXIT

if [ -n "${CORE_SRC:-}" ]; then
  cp "$CORE_SRC" "$tmp"
else
  url="https://github.com/MetaCubeX/mihomo/releases/download/${version}/${asset}-${version}.gz"
  curl -fL --retry 3 --proto '=https' "$url" | gunzip > "$tmp"
fi

if [ -n "$want" ]; then
  got="$(shasum -a 256 "$tmp" | cut -d' ' -f1)"
  [ "$got" = "$want" ] || { echo "sha256 mismatch for ${name}: got ${got}" >&2; exit 1; }
else
  echo "warning: no pinned sha256 for ${version}/${name}, checking the version string only" >&2
fi

chmod +x "$tmp"
"$tmp" -v | grep -q "$version" || { echo "unexpected core version: $("$tmp" -v)" >&2; exit 1; }
[ "$os" = darwin ] && xattr -d com.apple.quarantine "$tmp" 2>/dev/null || true
mv "$tmp" "$dest"
trap - EXIT
echo "core ready: $dest ($version)"

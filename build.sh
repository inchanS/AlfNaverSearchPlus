#!/bin/sh
# Build the universal (arm64 + amd64) naversearch binary, ad-hoc sign it, and
# place it in workflow/ ready for packaging.
#
# Ad-hoc signing (codesign -s -) is REQUIRED: unsigned Mach-O binaries are
# killed on Apple Silicon. Notarization is intentionally not used; the workflow's
# `run` shim strips the download quarantine at first launch instead.
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
PKG="./cmd/naversearch"
OUT="$ROOT/workflow/naversearch"
DIST="$ROOT/dist"
LDFLAGS="-s -w"

mkdir -p "$DIST"

echo "==> building arm64"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/naversearch-arm64" "$PKG"

echo "==> building amd64"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/naversearch-amd64" "$PKG"

echo "==> lipo -> universal"
lipo -create -output "$OUT" "$DIST/naversearch-arm64" "$DIST/naversearch-amd64"

echo "==> codesign (ad-hoc)"
codesign --force --sign - "$OUT"
codesign --verify --verbose "$OUT"

lipo -info "$OUT"

# Keep AppKit and Carbon off the per-keystroke search binary's launch path.
xcrun clang -fobjc-arc -Os -arch arm64 -arch x86_64 -mmacosx-version-min=12.0 \
    -framework AppKit -framework Carbon -framework CoreGraphics "$ROOT/native/input-source.m" -o "$DIST/input-source"
codesign --force --sign - "$DIST/input-source"
codesign --verify "$DIST/input-source"
cp "$DIST/input-source" "$ROOT/workflow/input-source"
python3 "$ROOT/scripts/build_nn_layouts.py"
echo "==> done: $OUT and workflow/input-source"

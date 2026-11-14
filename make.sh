#!/bin/sh
# Package the workflow into NaverSearchPlus.alfredworkflow.
# Builds the universal, ad-hoc signed binary first, then zips the workflow/
# directory (binary + run shim + info.plist + icons), excluding dev-only files.
set -eu

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# An explicit version also supports source archives without Git metadata.
# Otherwise prefer the release tag in CI, then the latest local version tag.
if [ -z "${WORKFLOW_VERSION:-}" ]; then
    case "${GITHUB_REF:-}" in
        refs/tags/v*) WORKFLOW_VERSION=${GITHUB_REF#refs/tags/v} ;;
        *) WORKFLOW_VERSION=$(git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || true) ;;
    esac
fi
WORKFLOW_VERSION=${WORKFLOW_VERSION#v}
if ! printf '%s\n' "$WORKFLOW_VERSION" | LC_ALL=C grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Cannot determine a release version. Set WORKFLOW_VERSION to a version such as 2.0.3." >&2
    exit 1
fi

# Build workflow/naversearch (universal + ad-hoc signed).
/bin/sh ./build.sh

# Stage a fresh copy on each run. Never overwrite the source version template
# or the previous package unless the new archive is complete.
package_dir=$(mktemp -d "${TMPDIR:-/tmp}/naversearchplus-package.XXXXXX")
trap 'rm -rf "$package_dir"' 0
trap 'exit 1' 1 2 3 15
cp -R workflow "$package_dir/workflow"
cd "$package_dir/workflow"

sed "s/{{VERSION_INFO}}/$WORKFLOW_VERSION/g" < info.plist > info.plist.bak
mv info.plist.bak info.plist

zip -r ../NaverSearchPlus.alfredworkflow . \
    -x ".naversearch-ready/*" \
    -x "*.DS_Store" \
    -x "*.pyc" \
    -x "__pycache__/*" "*/__pycache__/*" \
    -x "error.log" \
    -x "prefs.plist" \
    -x "make.sh" \
    -x "clean.sh"

mv ../NaverSearchPlus.alfredworkflow "$ROOT/NaverSearchPlus.alfredworkflow"

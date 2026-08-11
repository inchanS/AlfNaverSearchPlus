#!/bin/sh
# Package the workflow into NaverSearchPlus.alfredworkflow.
# Builds the universal, ad-hoc signed binary first, then zips the workflow/
# directory (binary + run shim + info.plist + icons), excluding dev-only files.
set -e

rm -f ./NaverSearchPlus.alfredworkflow

# Build workflow/naversearch (universal + ad-hoc signed).
/bin/sh ./build.sh

cd workflow

# Inject the release version (tag v1.2.3 -> 1.2.3) into info.plist.
sed "s/{{VERSION_INFO}}/${GITHUB_REF##*/v}/g" < info.plist > info.plist.bak
mv info.plist.bak info.plist

zip -r ../NaverSearchPlus.alfredworkflow . \
    -x "*.DS_Store" \
    -x "*.pyc" \
    -x "__pycache__/*" "*/__pycache__/*" \
    -x "error.log" \
    -x "prefs.plist" \
    -x "make.sh" \
    -x "clean.sh"

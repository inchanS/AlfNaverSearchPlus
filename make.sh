#!/bin/sh

rm -f ./NaverSearchPlus.alfredworkflow
cd workflow
sed "s/{{VERSION_INFO}}/${GITHUB_REF##*/v}/g" < info.plist > info.plist.bak
mv info.plist.bak info.plist
# 배포에 불필요한 개발용 파일은 제외하고 패키징
zip -r ../NaverSearchPlus.alfredworkflow . \
    -x "*.DS_Store" \
    -x "*.pyc" \
    -x "__pycache__/*" "*/__pycache__/*" \
    -x "error.log" \
    -x "prefs.plist" \
    -x "test_workflow.py" \
    -x "make.sh" \
    -x "clean.sh"

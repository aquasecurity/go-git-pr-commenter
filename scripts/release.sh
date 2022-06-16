#!/usr/bin/env bash

set -e


git checkout master
git fetch --tags --all
git pull

LATEST_TAG=`git describe --tags --abbrev=0`
read -p "The last tag was: ${LATEST_TAG}, what tag should I create? " TAG;

if [[ -z $TAG ]]; then
  echo "you need to specify a new tag"
  exit 1
fi

git tag -a ${TAG} -m ${TAG}
git push --tag


BRANCH_NAME="release-${TAG}"

sed  -e "s/PLACEHOLDERVERSION/${TAG}/g" .github/release_template.yaml > release.yaml
git checkout -b $BRANCH_NAME

git add release.yaml
git commit -m "Updating to latest tag ${TAG}" || true
git push --set-upstream origin $BRANCH_NAME || true

xdg-open "https://github.com/aquasecurity/go-git-pr-commenter/compare/${BRANCH_NAME}"

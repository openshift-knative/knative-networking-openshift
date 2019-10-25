#!/usr/bin/env bash

git cherry-pick "$1"

unmerged=$(git rm -- $(git ls-files -u | awk '{print $4;}'))
[ -n "$unmerged" ] && git rm "$unmerged"

git cherry-pick --continue

#!/usr/bin/env bash

echo "merging base: $1, user changes: $2 with new update: $3"

BASE=$1
USER=$2
UPDATE=$3

tmp_dir=$(mktemp -d)

cd $tmp_dir
echo $PWD
git init
mkdir content
cp $BASE/* content/
git add content/
git ci -m "base"

git co -b user-changes
rm -rf content/*
git add content/
cp $USER/* content/
git add content/
git ci -m "user-changes"


git co master
git co -b updates
rm -rf content/*
git add content/
cp $UPDATE/* content/
git add content/
git ci -m "updates-changes"

git merge user-changes

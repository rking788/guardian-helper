#!/bin/bash

version=$(cat ./VERSION)
hash=$(git rev-parse --short HEAD)

if [ -z $BUILD_NUMBER ]
then
    full_version="$version-$hash"
else
    full_version="$version-$hash-$BUILD_NUMBER"
fi

echo $full_version

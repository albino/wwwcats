#!/bin/bash

# We store the current revision number in a few of the game's source files,
# so that on startup, the client can check that everything is correct
# (web caching is flaky)
# This script updates all of those files at once (HACK)

# We can safely get the revision number from wwwcats.go this way
# because it won't compile if there is more than one 'var REVISION'

if [ ! -f wwwcats.go ]; then
	echo "Can't find source files"
	exit -1
fi

VERSION=$(egrep "^var REVISION = " wwwcats.go | rev | cut -d' ' -f 1 | rev)
NEW_VERSION=$((VERSION+1))

sed -i "s/var REVISION = $VERSION/var REVISION = $NEW_VERSION/" wwwcats.go
sed -i "s/var REVISION = \"$VERSION\";/var REVISION = \"$NEW_VERSION\";/" public_html/init.js
sed -i "s/id=\"REVISION\" value=\"$VERSION\"/id=\"REVISION\" value=\"$NEW_VERSION\"/" public_html/index.html

echo "Updated to version $NEW_VERSION"

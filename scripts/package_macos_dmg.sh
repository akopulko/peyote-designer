#!/bin/sh
set -eu

APP_NAME="peyote-designer"
APP_DISPLAY_NAME="Peyote Designer"
DIST="dist"
APP_BUNDLE="$DIST/$APP_DISPLAY_NAME.app"
DMG_PATH="$DIST/$APP_NAME-macos-arm64.dmg"

if [ ! -d "$APP_BUNDLE" ]; then
  echo "Missing app bundle: $APP_BUNDLE"
  exit 1
fi

rm -rf "$DMG_PATH"

hdiutil create \
  -volname "$APP_DISPLAY_NAME" \
  -fs HFS+ \
  -srcfolder "$APP_BUNDLE" \
  -ov \
  -format UDZO \
  "$DMG_PATH" >/dev/null

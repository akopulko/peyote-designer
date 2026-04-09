#!/bin/sh
set -eu

APP_NAME="peyote-designer"
APP_DISPLAY_NAME="Peyote Designer"
APP_ID="com.kostya.peyote-designer"
DIST="dist"
APP_BUNDLE="$DIST/$APP_DISPLAY_NAME.app"
CONTENTS_DIR="$APP_BUNDLE/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
ICONSET_DIR="$DIST/AppIcon.iconset"
ICNS_PATH="$RESOURCES_DIR/AppIcon.icns"
PNG_ICON="$RESOURCES_DIR/app.png"
BIN_PATH="$MACOS_DIR/$APP_NAME"
LEGACY_BIN="$DIST/$APP_NAME-darwin-arm64"

rm -rf "$APP_BUNDLE" "$ICONSET_DIR" "$LEGACY_BIN"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR" "$ICONSET_DIR"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "$BIN_PATH" ./cmd/peyote-designer
cp icons/app.png "$PNG_ICON"

sips -z 16 16 icons/app.png --out "$ICONSET_DIR/icon_16x16.png" >/dev/null
sips -z 32 32 icons/app.png --out "$ICONSET_DIR/icon_16x16@2x.png" >/dev/null
sips -z 32 32 icons/app.png --out "$ICONSET_DIR/icon_32x32.png" >/dev/null
sips -z 64 64 icons/app.png --out "$ICONSET_DIR/icon_32x32@2x.png" >/dev/null
sips -z 128 128 icons/app.png --out "$ICONSET_DIR/icon_128x128.png" >/dev/null
sips -z 256 256 icons/app.png --out "$ICONSET_DIR/icon_128x128@2x.png" >/dev/null
sips -z 256 256 icons/app.png --out "$ICONSET_DIR/icon_256x256.png" >/dev/null
sips -z 512 512 icons/app.png --out "$ICONSET_DIR/icon_256x256@2x.png" >/dev/null
sips -z 512 512 icons/app.png --out "$ICONSET_DIR/icon_512x512.png" >/dev/null
cp icons/app.png "$ICONSET_DIR/icon_512x512@2x.png"
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_PATH"
rm -rf "$ICONSET_DIR"

cat > "$CONTENTS_DIR/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleDisplayName</key>
  <string>$APP_DISPLAY_NAME</string>
  <key>CFBundleExecutable</key>
  <string>$APP_NAME</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>CFBundleIdentifier</key>
  <string>$APP_ID</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>$APP_DISPLAY_NAME</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
EOF

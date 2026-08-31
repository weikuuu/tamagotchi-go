#!/bin/sh
# Builds "Elygochi.app", a normal double-clickable macOS app —
# no terminal needed to run it afterwards.
#
# Usage: ./scripts/build_macos_app.sh
# Output: ./Elygochi.app (repo root)

set -e

cd "$(dirname "$0")/.."

APP="Elygochi.app"
CONTENTS="$APP/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"

rm -rf "$APP"
mkdir -p "$MACOS" "$RESOURCES"

echo "Building binary..."
go build -o "$MACOS/elygochi" ./cmd/elygochi

echo "Building app icon..."
ICONSET=$(mktemp -d)/AppIcon.iconset
mkdir -p "$ICONSET"
for size in 16 32 128 256 512; do
	sips -z "$size" "$size" scripts/icon_source.png --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
	double=$((size * 2))
	sips -z "$double" "$double" scripts/icon_source.png --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$RESOURCES/AppIcon.icns"
rm -rf "$(dirname "$ICONSET")"

cat > "$CONTENTS/Info.plist" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Elygochi</string>
	<key>CFBundleDisplayName</key>
	<string>Elygochi</string>
	<key>CFBundleIdentifier</key>
	<string>com.elygochi.elysia</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleExecutable</key>
	<string>elygochi</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
</dict>
</plist>
EOF

echo "Done: $APP"
echo "Double-click it in Finder, or drag it to /Applications."

#!/bin/bash
set -euo pipefail

APP_NAME="Pomo"
BUNDLE_ID="com.pomo.app"
VERSION="1.0.0"
KITTY_PATH="/Applications/kitty.app/Contents/MacOS/kitty"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$SCRIPT_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

# Clean previous build
rm -rf "$APP_DIR"

# Check kitty exists
if [ ! -f "$KITTY_PATH" ]; then
    echo "Error: kitty not found at $KITTY_PATH"
    exit 1
fi

# Build Go binary
echo "Building pomo..."
cd "$SCRIPT_DIR"
go build -o ./pomo .
echo "Build complete."

# Create .app structure
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

# Copy binary
cp "$SCRIPT_DIR/pomo" "$RESOURCES_DIR/pomo"
chmod +x "$RESOURCES_DIR/pomo"

# Generate icon from a simple PNG if none exists
ICON_PATH="$RESOURCES_DIR/icon.icns"
if [ -f "$SCRIPT_DIR/icon.png" ]; then
    echo "Generating .icns from icon.png..."
    ICONSET_DIR=$(mktemp -d)/icon.iconset
    mkdir -p "$ICONSET_DIR"
    for SIZE in 16 32 64 128 256 512; do
        sips -z $SIZE $SIZE "$SCRIPT_DIR/icon.png" --out "$ICONSET_DIR/icon_${SIZE}x${SIZE}.png" > /dev/null 2>&1
        DOUBLE=$((SIZE * 2))
        sips -z $DOUBLE $DOUBLE "$SCRIPT_DIR/icon.png" --out "$ICONSET_DIR/icon_${SIZE}x${SIZE}@2x.png" > /dev/null 2>&1
    done
    iconutil -c icns "$ICONSET_DIR" -o "$ICON_PATH"
    rm -rf "$(dirname "$ICONSET_DIR")"
    echo "Icon generated."
else
    echo "No icon.png found in project root — app will use default icon."
    echo "To add a custom icon, place a 1024x1024 icon.png in the project root and re-run."
fi

# Write Info.plist
cat > "$CONTENTS_DIR/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>launcher</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST

# Write launcher script
cat > "$MACOS_DIR/launcher" << 'LAUNCHER'
#!/bin/bash
DIR="$(cd "$(dirname "$0")/../Resources" && pwd)"
/Applications/kitty.app/Contents/MacOS/kitty \
    --title "Pomo" \
    --instance-group pomo \
    -o hide_window_decorations=titlebar-only \
    -o macos_quit_when_last_window_closed=yes \
    -o confirm_os_window_close=0 \
    -o font_size=14 \
    "$DIR/pomo"
LAUNCHER
chmod +x "$MACOS_DIR/launcher"

# Code sign
echo "Code signing..."
codesign --force --deep --sign - "$APP_DIR" 2>/dev/null && echo "Signed." || echo "Warning: code signing failed (app will still work, Gatekeeper may prompt)."

# Copy to /Applications
echo ""
read -p "Copy $APP_NAME.app to /Applications? [y/N] " REPLY
if [[ "$REPLY" =~ ^[Yy]$ ]]; then
    cp -R "$APP_DIR" /Applications/
    echo "Installed to /Applications/$APP_NAME.app"
else
    echo "App built at: $APP_DIR"
fi

echo "Done!"

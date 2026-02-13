PROJECT_DIR := $(shell pwd)
ROOT_DIR := $(PROJECT_DIR)/../../..
DAEMON_DIR := $(ROOT_DIR)/daemon
BUILD_DIR := $(PROJECT_DIR)/build
APP_NAME := PlugMyAI
APP_BUNDLE := $(BUILD_DIR)/Build/Products/Release/$(APP_NAME).app
BINARY_NAME := plug-my-ai
DMG_NAME := PlugMyAI.dmg

# Code signing identity (override with: make sign IDENTITY="Developer ID Application: ...")
IDENTITY ?= -
VERSION ?= 0.1.0
WEBSITE_DIR := $(ROOT_DIR)/website
VERSIONED_DMG := PlugMyAI-$(VERSION).dmg
SERVER_GO := $(DAEMON_DIR)/internal/server/server.go
INFO_PLIST := $(PROJECT_DIR)/PlugMyAI/Resources/Info.plist
GENERATE_APPCAST := $(BUILD_DIR)/SourcePackages/artifacts/sparkle/Sparkle/bin/generate_appcast

# DMG layout
DMG_STAGING := $(BUILD_DIR)/dmg-staging
DMG_RW := $(BUILD_DIR)/$(APP_NAME)-rw.dmg
DMG_WINDOW_W := 540
DMG_WINDOW_H := 380
DMG_ICON_SIZE := 128
DMG_APP_X := 150
DMG_APP_Y := 190
DMG_ALIAS_X := 390
DMG_ALIAS_Y := 190

# AppleScript to style the DMG window (used via: echo "$$DMG_LAYOUT_SCRIPT" | osascript)
define DMG_LAYOUT_SCRIPT
tell application "Finder"
	tell disk "$(APP_NAME)"
		open
		set current view of container window to icon view
		set toolbar visible of container window to false
		set statusbar visible of container window to false
		set bounds of container window to {100, 100, $(shell echo $$((100 + $(DMG_WINDOW_W)))), $(shell echo $$((100 + $(DMG_WINDOW_H))))}
		set theViewOptions to the icon view options of container window
		set icon size of theViewOptions to $(DMG_ICON_SIZE)
		set arrangement of theViewOptions to not arranged
		set position of item "$(APP_NAME).app" of container window to {$(DMG_APP_X), $(DMG_APP_Y)}
		set position of item "Applications" of container window to {$(DMG_ALIAS_X), $(DMG_ALIAS_Y)}
		close
		open
		update without registering applications
		delay 1
		close
	end tell
end tell
endef
export DMG_LAYOUT_SCRIPT

GITHUB_REPO := sxp-studio/PlugMyAI

.PHONY: all clean project app daemon-universal bundle sign dmg notarize bump-version appcast checksums gh-release release

all: bundle

# Generate Xcode project from project.yml
project:
	xcodegen generate

# Build Go daemon as universal binary (arm64 + amd64)
daemon-universal:
	cd $(DAEMON_DIR) && GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		go build -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 ./cmd/plug-my-ai
	cd $(DAEMON_DIR) && GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		go build -o $(BUILD_DIR)/$(BINARY_NAME)-amd64 ./cmd/plug-my-ai
	lipo -create -output $(BUILD_DIR)/$(BINARY_NAME) \
		$(BUILD_DIR)/$(BINARY_NAME)-arm64 \
		$(BUILD_DIR)/$(BINARY_NAME)-amd64
	rm $(BUILD_DIR)/$(BINARY_NAME)-arm64 $(BUILD_DIR)/$(BINARY_NAME)-amd64

# Build the Swift app via xcodebuild
app: project
	xcodebuild -project $(APP_NAME).xcodeproj \
		-scheme $(APP_NAME) \
		-configuration Release \
		-derivedDataPath $(BUILD_DIR) \
		CODE_SIGN_IDENTITY="$(IDENTITY)" \
		CODE_SIGN_INJECT_BASE_ENTITLEMENTS=NO \
		OTHER_CODE_SIGN_FLAGS=--timestamp \
		build

# Copy Go daemon binary into .app bundle
bundle: app daemon-universal
	mkdir -p $(APP_BUNDLE)/Contents/Resources
	cp $(BUILD_DIR)/$(BINARY_NAME) $(APP_BUNDLE)/Contents/Resources/$(BINARY_NAME)
	chmod +x $(APP_BUNDLE)/Contents/Resources/$(BINARY_NAME)
	@echo "✓ $(APP_BUNDLE) is ready"

# Code sign with Developer ID for distribution (inside-out, no --deep)
sign:
	@echo "Signing binaries with $(IDENTITY)..."
	# Sparkle XPC services (innermost first)
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Frameworks/Sparkle.framework/Versions/B/XPCServices/Downloader.xpc
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Frameworks/Sparkle.framework/Versions/B/XPCServices/Installer.xpc
	# Sparkle Updater.app
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Frameworks/Sparkle.framework/Versions/B/Updater.app
	# Sparkle Autoupdate binary
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Frameworks/Sparkle.framework/Versions/B/Autoupdate
	# Sparkle framework itself
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Frameworks/Sparkle.framework
	# Go daemon binary
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)/Contents/Resources/$(BINARY_NAME)
	# Main app bundle (outermost, last)
	codesign --force --options runtime --timestamp \
		--sign "$(IDENTITY)" \
		$(APP_BUNDLE)

# Create distributable DMG with Applications shortcut and clean layout
# Expects a signed .app bundle — run `bundle` then `sign` first
dmg:
	@echo "Creating DMG…"
	rm -rf $(DMG_STAGING) $(DMG_RW) $(BUILD_DIR)/$(DMG_NAME)
	mkdir -p $(DMG_STAGING)
	cp -a $(APP_BUNDLE) $(DMG_STAGING)/$(APP_NAME).app
	ln -s /Applications $(DMG_STAGING)/Applications
	hdiutil create -volname "$(APP_NAME)" \
		-srcfolder $(DMG_STAGING) \
		-ov -format UDRW -fs HFS+ \
		$(DMG_RW)
	hdiutil attach $(DMG_RW) -mountpoint /Volumes/$(APP_NAME) -noverify -noautoopen
	echo "$$DMG_LAYOUT_SCRIPT" | osascript
	chmod -Rf go-w /Volumes/$(APP_NAME) || true
	sync
	hdiutil detach /Volumes/$(APP_NAME)
	hdiutil convert $(DMG_RW) -format UDZO -imagekey zlib-level=9 -o $(BUILD_DIR)/$(DMG_NAME)
	rm -rf $(DMG_RW) $(DMG_STAGING)
	@echo "✓ $(BUILD_DIR)/$(DMG_NAME)"

# Submit to Apple notary service
notarize:
	xcrun notarytool submit $(BUILD_DIR)/$(DMG_NAME) \
		--keychain-profile "notarytool-profile" \
		--wait
	xcrun stapler staple $(BUILD_DIR)/$(DMG_NAME)

# Update version in Info.plist and Go server
bump-version:
	@echo "Bumping version to $(VERSION)"
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" $(INFO_PLIST)
	sed -i '' 's/const Version = ".*"/const Version = "$(VERSION)"/' $(SERVER_GO)
	@echo "✓ Version set to $(VERSION)"

# Copy DMG to website/releases/ and regenerate appcast
appcast:
	mkdir -p $(WEBSITE_DIR)/releases
	cp $(BUILD_DIR)/$(DMG_NAME) $(WEBSITE_DIR)/releases/$(VERSIONED_DMG)
	$(GENERATE_APPCAST) $(WEBSITE_DIR)/releases/
	@echo "✓ Appcast updated at $(WEBSITE_DIR)/appcast.xml"

# Generate SHA-256 checksums for release artifacts (committed to git for verification)
checksums:
	cd $(WEBSITE_DIR)/releases && shasum -a 256 *.dmg > SHA256SUMS
	@echo "✓ Checksums written to $(WEBSITE_DIR)/releases/SHA256SUMS"
	@cat $(WEBSITE_DIR)/releases/SHA256SUMS

# Upload DMG + CLI binaries to GitHub Releases
gh-release:
	gh release create v$(VERSION) \
		--repo $(GITHUB_REPO) \
		--title "v$(VERSION)" \
		--generate-notes \
		$(WEBSITE_DIR)/releases/$(VERSIONED_DMG) \
		$(ROOT_DIR)/bin/$(BINARY_NAME)-darwin-arm64 \
		$(ROOT_DIR)/bin/$(BINARY_NAME)-darwin-amd64 \
		$(ROOT_DIR)/bin/$(BINARY_NAME)-linux-amd64 \
		$(ROOT_DIR)/bin/$(BINARY_NAME)-linux-arm64
	@echo "✓ GitHub Release v$(VERSION) published"

# Full release pipeline
release: bump-version bundle sign dmg notarize appcast checksums
	@echo ""
	@echo "✓ Release $(VERSION) built. Next steps:"
	@echo "  1. git add website/releases/SHA256SUMS && git commit && git push"
	@echo "  2. make gh-release        # upload to GitHub Releases"
	@echo "  3. cd website && ./deploy.sh  # deploy website + appcast"

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(APP_NAME).xcodeproj

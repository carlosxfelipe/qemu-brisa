//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	appName := "QEMU Brisa"
	bundleID := "com.carlosxfelipe.qemu-brisa"
	version := "1.0.0"

	appBundle := appName + ".app"
	contentsDir := filepath.Join(appBundle, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")

	fmt.Printf("📦 Building macOS Application Bundle: %s\n", appBundle)

	// 1. Clean and create bundle directories
	_ = os.RemoveAll(appBundle)
	if err := os.MkdirAll(macOSDir, 0755); err != nil {
		fatalf("failed to create MacOS directory: %v", err)
	}
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		fatalf("failed to create Resources directory: %v", err)
	}

	// 2. Compile native macOS Go binary
	binPath := filepath.Join(macOSDir, appName)
	fmt.Printf("🔨 Compiling native macOS binary -> %s\n", binPath)

	buildCmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", binPath, "main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fatalf("go build failed: %s (%v)", string(out), err)
	}

	// 3. Copy app.icns to Resources
	icnsSrc := filepath.Join("assets", "app.icns")
	icnsDst := filepath.Join(resourcesDir, "app.icns")
	if data, err := os.ReadFile(icnsSrc); err == nil {
		if err := os.WriteFile(icnsDst, data, 0644); err != nil {
			fatalf("failed to write app.icns: %v", err)
		}
		fmt.Printf("🎨 Copied icon: %s -> %s\n", icnsSrc, icnsDst)
	} else {
		fmt.Printf("⚠️  Warning: %s not found, continuing without icon\n", icnsSrc)
	}

	// 4. Generate Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>%s</string>
    <key>CFBundleDisplayName</key>
    <string>%s</string>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleVersion</key>
    <string>%s</string>
    <key>CFBundleShortVersionString</key>
    <string>%s</string>
    <key>CFBundleExecutable</key>
    <string>%s</string>
    <key>CFBundleIconFile</key>
    <string>app.icns</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
`, appName, appName, bundleID, version, version, appName)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		fatalf("failed to write Info.plist: %v", err)
	}

	// 5. Generate PkgInfo
	pkgInfoPath := filepath.Join(contentsDir, "PkgInfo")
	_ = os.WriteFile(pkgInfoPath, []byte("APPL????"), 0644)

	fmt.Printf("\n✨ Successfully generated: %s\n", appBundle)
	fmt.Printf("👉 To run: open %q\n", appBundle)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

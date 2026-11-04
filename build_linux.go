//go:build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	keepAppDir := flag.Bool("keep-appdir", false, "Keep temporary AppDir directory after packaging")
	flag.Parse()

	appName := "QEMU Brisa"
	execName := "qemu-brisa"
	version := "1.0.0"
	description := "Gerenciador de Máquinas Virtuais QEMU com interface moderna Brisa."
	categories := "System;Emulator;Utility;"

	arch := appImageArch(runtime.GOARCH)
	if arch == "" {
		fatalf("unsupported architecture for AppImage: %s", runtime.GOARCH)
	}

	appDirName := strings.ReplaceAll(appName, " ", "_") + ".AppDir"
	outputAppImage := fmt.Sprintf("%s-%s-%s.AppImage", strings.ReplaceAll(appName, " ", "_"), version, arch)

	fmt.Printf("📦 Building Linux AppImage: %s\n", outputAppImage)

	// 1. Clean and setup AppDir structure
	_ = os.RemoveAll(appDirName)
	_ = os.Remove(outputAppImage)

	binDir := filepath.Join(appDirName, "usr", "bin")
	applicationsDir := filepath.Join(appDirName, "usr", "share", "applications")
	iconsDir := filepath.Join(appDirName, "usr", "share", "icons", "hicolor", "256x256", "apps")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		fatalf("failed to create usr/bin directory: %v", err)
	}
	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		fatalf("failed to create applications directory: %v", err)
	}
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		fatalf("failed to create icons directory: %v", err)
	}

	// 2. Compile native Linux Go binary
	binPath := filepath.Join(binDir, execName)
	fmt.Printf("🔨 Compiling Linux binary -> %s\n", binPath)

	buildCmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", binPath, "main.go")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fatalf("go build failed: %s (%v)", string(out), err)
	}

	// 3. Setup desktop icons
	iconSrc := filepath.Join("assets", "icon.png")
	if data, err := os.ReadFile(iconSrc); err == nil {
		// AppDir root icon
		rootIcon := filepath.Join(appDirName, execName+".png")
		_ = os.WriteFile(rootIcon, data, 0644)

		// .DirIcon (used by file managers)
		dirIcon := filepath.Join(appDirName, ".DirIcon")
		_ = os.WriteFile(dirIcon, data, 0644)

		// System icon path
		sysIcon := filepath.Join(iconsDir, execName+".png")
		_ = os.WriteFile(sysIcon, data, 0644)

		fmt.Printf("🎨 Installed application icons from %s\n", iconSrc)
	} else {
		fmt.Printf("⚠️  Warning: %s not found, continuing without icon\n", iconSrc)
	}

	// 4. Generate .desktop entry
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s %%u
Icon=%s
Comment=%s
Categories=%s
Terminal=false
StartupWMClass=%s
`, appName, execName, execName, description, categories, appName)

	rootDesktop := filepath.Join(appDirName, execName+".desktop")
	if err := os.WriteFile(rootDesktop, []byte(desktopContent), 0644); err != nil {
		fatalf("failed to write %s: %v", rootDesktop, err)
	}

	sysDesktop := filepath.Join(applicationsDir, execName+".desktop")
	if err := os.WriteFile(sysDesktop, []byte(desktopContent), 0644); err != nil {
		fatalf("failed to write %s: %v", sysDesktop, err)
	}

	// 5. Generate AppRun entrypoint
	appRunPath := filepath.Join(appDirName, "AppRun")
	appRunContent := fmt.Sprintf(`#!/bin/sh
SELF=$(readlink -f "$0")
HERE=${SELF%%/*}
export PATH="${HERE}/usr/bin:${PATH}"
export LD_LIBRARY_PATH="${HERE}/usr/lib:${LD_LIBRARY_PATH}"
exec "${HERE}/usr/bin/%s" "$@"
`, execName)

	if err := os.WriteFile(appRunPath, []byte(appRunContent), 0755); err != nil {
		fatalf("failed to write AppRun: %v", err)
	}

	// 6. Ensure appimagetool is available
	toolPath, err := ensureAppImageTool(arch)
	if err != nil {
		fatalf("failed to obtain appimagetool: %v", err)
	}

	// 7. Package into AppImage
	fmt.Printf("🚀 Packaging AppImage using %s...\n", filepath.Base(toolPath))
	pkgCmd := exec.Command(toolPath, appDirName, outputAppImage)
	pkgCmd.Env = append(os.Environ(),
		fmt.Sprintf("ARCH=%s", arch),
		"APPIMAGE_EXTRACT_AND_RUN=1",
	)
	pkgCmd.Stdout = os.Stdout
	pkgCmd.Stderr = os.Stderr

	if err := pkgCmd.Run(); err != nil {
		fatalf("appimagetool failed: %v", err)
	}

	if !*keepAppDir {
		_ = os.RemoveAll(appDirName)
	}

	fi, err := os.Stat(outputAppImage)
	if err != nil {
		fatalf("failed to verify output AppImage: %v", err)
	}

	sizeMB := float64(fi.Size()) / (1024 * 1024)
	fmt.Printf("\n✨ Successfully generated: %s (%.2f MB)\n", outputAppImage, sizeMB)
	fmt.Printf("👉 To run: ./%s\n", outputAppImage)
}

func appImageArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i686"
	case "arm":
		return "armhf"
	default:
		return ""
	}
}

func ensureAppImageTool(arch string) (string, error) {
	// 1. Check if appimagetool is in PATH
	if p, err := exec.LookPath("appimagetool"); err == nil {
		return p, nil
	}

	// 2. Check local user cache directory
	userCache, err := os.UserCacheDir()
	if err != nil {
		userCache = filepath.Join(os.TempDir(), "brisa-cache")
	}
	toolsDir := filepath.Join(userCache, "brisa", "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return "", err
	}

	toolFileName := fmt.Sprintf("appimagetool-%s.AppImage", arch)
	toolPath := filepath.Join(toolsDir, toolFileName)

	if fi, err := os.Stat(toolPath); err == nil && fi.Size() > 0 {
		return toolPath, nil
	}

	// 3. Download official appimagetool continuous release
	url := fmt.Sprintf("https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-%s.AppImage", arch)
	fmt.Printf("📥 appimagetool not found. Downloading from %s -> %s\n", url, toolPath)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with HTTP status: %s", resp.Status)
	}

	tmpFile := toolPath + ".tmp"
	out, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpFile)
		return "", fmt.Errorf("failed writing downloaded file: %w", err)
	}
	out.Close()

	if err := os.Rename(tmpFile, toolPath); err != nil {
		return "", err
	}

	if err := os.Chmod(toolPath, 0755); err != nil {
		return "", err
	}

	return toolPath, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

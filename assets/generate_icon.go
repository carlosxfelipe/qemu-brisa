//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

type iconTarget struct {
	name string
	size int
}

func main() {
	baseDir := "."
	if _, err := os.Stat("icon_source.svg"); os.IsNotExist(err) {
		if _, file, _, ok := runtime.Caller(0); ok {
			baseDir = filepath.Dir(file)
		}
	}

	svgFile := filepath.Join(baseDir, "icon_source.svg")
	iconsetDir := filepath.Join(baseDir, "icon.iconset")
	icnsFile := filepath.Join(baseDir, "app.icns")
	pngFile := filepath.Join(baseDir, "icon.png")

	if _, err := os.Stat(svgFile); err != nil {
		fatalf("SVG source not found at %s: %v", svgFile, err)
	}

	_ = os.RemoveAll(iconsetDir)
	if err := os.MkdirAll(iconsetDir, 0o755); err != nil {
		fatalf("failed to create iconset directory: %v", err)
	}
	defer os.RemoveAll(iconsetDir)

	sizes := []iconTarget{
		{name: "icon_16x16.png", size: 16},
		{name: "icon_16x16@2x.png", size: 32},
		{name: "icon_32x32.png", size: 32},
		{name: "icon_32x32@2x.png", size: 64},
		{name: "icon_128x128.png", size: 128},
		{name: "icon_128x128@2x.png", size: 256},
		{name: "icon_256x256.png", size: 256},
		{name: "icon_256x256@2x.png", size: 512},
		{name: "icon_512x512.png", size: 512},
		{name: "icon_512x512@2x.png", size: 1024},
	}

	fmt.Println("Generating PNGs directly from SVG (perfect rasterization)...")

	for _, item := range sizes {
		fmt.Printf("  -> %s (%dx%d)\n", item.name, item.size, item.size)
		outPath := filepath.Join(iconsetDir, item.name)
		if err := rasterizeSVG(svgFile, outPath, item.size); err != nil {
			fatalf("failed to render %s: %v", item.name, err)
		}
	}

	if err := rasterizeSVG(svgFile, pngFile, 512); err != nil {
		fatalf("failed to render %s: %v", pngFile, err)
	}

	fmt.Println("Converting to ICNS using iconutil...")
	cmd := exec.Command("iconutil", "-c", "icns", iconsetDir, "-o", icnsFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		fatalf("iconutil failed: %s (%v)", string(out), err)
	}

	fmt.Printf("Success! Created %s and %s with perfect transparency.\n", icnsFile, pngFile)
}

func rasterizeSVG(svgPath, outPath string, size int) error {
	sizeStr := strconv.Itoa(size)

	if path, err := exec.LookPath("rsvg-convert"); err == nil {
		cmd := exec.Command(path, "-w", sizeStr, "-h", sizeStr, svgPath, "-o", outPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rsvg-convert: %s (%w)", string(out), err)
		}
		return nil
	}

	if path, err := exec.LookPath("resvg"); err == nil {
		cmd := exec.Command(path, "-w", sizeStr, "-h", sizeStr, svgPath, outPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("resvg: %s (%w)", string(out), err)
		}
		return nil
	}

	if path, err := exec.LookPath("sips"); err == nil {
		cmd := exec.Command(path, "-s", "format", "png", "-z", sizeStr, sizeStr, svgPath, "--out", outPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("sips: %s (%w)", string(out), err)
		}
		return nil
	}

	return fmt.Errorf("no SVG rasterizer found in PATH (install librsvg/rsvg-convert or resvg)")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error generating icon: "+format+"\n", args...)
	os.Exit(1)
}


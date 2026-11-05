package vm

import (
	"os"
	"strings"
	"testing"
)

func TestBuildCommandTCGOptimizations(t *testing.T) {
	mgr := NewManager(nil)
	cfg := VMConfig{
		Name:        "TestMint",
		Arch:        "x86_64",
		Cores:       2,
		MemoryMB:    2048,
		EnableAccel: true,
	}

	bin, args, err := mgr.BuildCommand(cfg)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	cmdStr := bin + " " + strings.Join(args, " ")
	t.Logf("Generated command: %s", cmdStr)

	if !strings.Contains(cmdStr, "-accel tcg,thread=multi,tb-size=512") {
		t.Errorf("Expected MTTCG flags, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "-cpu Nehalem") {
		t.Errorf("Expected Nehalem CPU model, got: %s", cmdStr)
	}
}

func TestBuildCommandISODiskCorrection(t *testing.T) {
	mgr := NewManager(nil)
	cfg := VMConfig{
		Name:        "GnomeOS",
		Arch:        "aarch64",
		Cores:       2,
		MemoryMB:    4096,
		DiskPath:    "/Users/carlos/Downloads/gnome_os_51.1-aarch64.iso",
		EnableAccel: true,
	}

	bin, args, err := mgr.BuildCommand(cfg)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	cmdStr := bin + " " + strings.Join(args, " ")
	t.Logf("Generated command: %s", cmdStr)

	// Should NOT map an .iso file as format=qcow2
	if strings.Contains(cmdStr, "format=qcow2") {
		t.Errorf("Expected ISO not to be mapped as format=qcow2: %s", cmdStr)
	}
	// Should map as CD-ROM boot
	if !strings.Contains(cmdStr, "media=cdrom") {
		t.Errorf("Expected ISO to be mapped as cdrom: %s", cmdStr)
	}
}

func TestGetDiskSpaceGB(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home dir")
	}
	freeGB, totalGB := GetDiskSpaceGB(home)
	t.Logf("Disk space at %s: %d GB free / %d GB total", home, freeGB, totalGB)
	if totalGB <= 0 {
		t.Errorf("Expected total disk space > 0, got %d", totalGB)
	}
}

package vm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProcessHandle struct {
	Config    VMConfig
	Cmd       *exec.Cmd
	StartedAt time.Time
	Cancel    chan struct{}
	Logs      []string
}

type Manager struct {
	mu        sync.RWMutex
	processes map[string]*ProcessHandle
	states    map[string]VMState
	disksDir  string
	onEvent   func(event string, data any)
}

func NewManager(onEvent func(event string, data any)) *Manager {
	home, _ := os.UserHomeDir()
	disksDir := filepath.Join(home, ".qemu-brisa", "disks")
	_ = os.MkdirAll(disksDir, 0755)

	return &Manager{
		processes: make(map[string]*ProcessHandle),
		states:    make(map[string]VMState),
		disksDir:  disksDir,
		onEvent:   onEvent,
	}
}

func (m *Manager) DisksDir() string {
	return m.disksDir
}

// LocateBinary looks for an executable in PATH or Homebrew directories.
func LocateBinary(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	candidates := []string{
		filepath.Join("/opt/homebrew/bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/bin", name),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

// LocateFirmware searches for EDK2 / OVMF UEFI firmware files.
func LocateFirmware(filename string) string {
	candidates := []string{
		filepath.Join("/opt/homebrew/share/qemu", filename),
		filepath.Join("/usr/local/share/qemu", filename),
		filepath.Join("/usr/share/qemu", filename),
		filepath.Join("/usr/share/edk2-ovmf", filename),
		filepath.Join("/usr/share/OVMF", filename),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

// GetHostMemoryMB queries total host RAM in MB.
func GetHostMemoryMB() int {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			if err == nil && bytes > 0 {
				return int(bytes / (1024 * 1024))
			}
		}
	}
	return 8192 // Default fallback
}

// GetSystemInfo detects host capabilities and QEMU setup.
func (m *Manager) GetSystemInfo() SystemInfo {
	qemuX86 := LocateBinary("qemu-system-x86_64")
	qemuArm := LocateBinary("qemu-system-aarch64")
	qemuImg := LocateBinary("qemu-img")

	version := "Not found"
	targetBin := qemuX86
	if targetBin == "" {
		targetBin = qemuArm
	}
	if targetBin != "" {
		out, err := exec.Command(targetBin, "--version").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				version = strings.TrimSpace(lines[0])
			}
		}
	}

	accels := []string{"tcg"}
	switch runtime.GOOS {
	case "darwin":
		accels = append(accels, "hvf")
	case "linux":
		accels = append(accels, "kvm")
	}

	return SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		HostCPUs:     runtime.NumCPU(),
		HostMemoryMB: GetHostMemoryMB(),
		QemuVersion:  version,
		QemuX86Path:  qemuX86,
		QemuArmPath:  qemuArm,
		QemuImgPath:  qemuImg,
		Accelerators: accels,
		Edk2ArmPath:  LocateFirmware("edk2-aarch64-code.fd"),
		Edk2X86Path:  LocateFirmware("edk2-x86_64-code.fd"),
		DisksDir:     m.disksDir,
	}
}

// CreateDisk creates a QCOW2 virtual disk image.
func (m *Manager) CreateDisk(name string, sizeGB int) (string, error) {
	qemuImg := LocateBinary("qemu-img")
	if qemuImg == "" {
		return "", errors.New("qemu-img não encontrado. Certifique-se de que o QEMU está instalado")
	}

	if !strings.HasSuffix(name, ".qcow2") && !strings.HasSuffix(name, ".img") {
		name = name + ".qcow2"
	}
	targetPath := filepath.Join(m.disksDir, name)

	// Avoid overwriting existing disk
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil
	}

	sizeStr := fmt.Sprintf("%dG", sizeGB)
	cmd := exec.Command(qemuImg, "create", "-f", "qcow2", targetPath, sizeStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("falha ao criar disco: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	return targetPath, nil
}

// BuildCommand generates the exact executable path and arguments for a VM.
func (m *Manager) BuildCommand(cfg VMConfig) (string, []string, error) {
	// Auto-correct if an .iso was inadvertently entered as a hard disk
	if cfg.ISOPath == "" && strings.HasSuffix(strings.ToLower(cfg.DiskPath), ".iso") {
		cfg.ISOPath = cfg.DiskPath
		cfg.DiskPath = ""
	}

	diskFormat := "qcow2"
	if lower := strings.ToLower(cfg.DiskPath); strings.HasSuffix(lower, ".raw") || strings.HasSuffix(lower, ".img") || strings.HasSuffix(lower, ".iso") {
		diskFormat = "raw"
	}

	var bin string
	switch cfg.Arch {
	case "aarch64":
		bin = LocateBinary("qemu-system-aarch64")
		if bin == "" {
			return "", nil, errors.New("qemu-system-aarch64 não encontrado no sistema")
		}
	default:
		bin = LocateBinary("qemu-system-x86_64")
		if bin == "" {
			return "", nil, errors.New("qemu-system-x86_64 não encontrado no sistema")
		}
	}

	var args []string

	// Basic RAM and SMP
	mem := cfg.MemoryMB
	if mem <= 0 {
		mem = 2048
	}
	cores := cfg.Cores
	if cores <= 0 {
		cores = 2
	}

	args = append(args, "-m", fmt.Sprintf("%dM", mem))
	args = append(args, "-smp", fmt.Sprintf("%d", cores))

	isDarwinArm := runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
	isDarwinX86 := runtime.GOOS == "darwin" && runtime.GOARCH == "amd64"
	isLinuxArm := runtime.GOOS == "linux" && runtime.GOARCH == "arm64"
	isLinuxX86 := runtime.GOOS == "linux" && runtime.GOARCH == "amd64"

	if cfg.Arch == "aarch64" {
		args = append(args, "-M", "virt,highmem=on")
		if cfg.EnableAccel && isDarwinArm {
			args = append(args, "-accel", "hvf", "-cpu", "host")
		} else if cfg.EnableAccel && isLinuxArm {
			args = append(args, "-accel", "kvm", "-cpu", "host")
		} else {
			args = append(args, "-accel", "tcg,thread=multi,tb-size=512", "-cpu", "cortex-a72")
		}

		// UEFI firmware for ARM64 virt machine
		edk2 := LocateFirmware("edk2-aarch64-code.fd")
		if edk2 != "" {
			args = append(args, "-bios", edk2)
		}

		// Display & Devices
		if runtime.GOOS == "darwin" {
			args = append(args, "-display", "cocoa")
		}
		args = append(args, "-device", "virtio-gpu-pci")
		args = append(args, "-device", "qemu-xhci", "-device", "usb-kbd", "-device", "usb-tablet")
		args = append(args, "-netdev", "user,id=net0", "-device", "virtio-net-pci,netdev=net0")

		// Storage
		if cfg.DiskPath != "" {
			args = append(args, "-drive", fmt.Sprintf("file=%s,format=%s,if=none,id=hd0", cfg.DiskPath, diskFormat))
			args = append(args, "-device", "virtio-blk-pci,drive=hd0")
		}
		if cfg.ISOPath != "" {
			args = append(args, "-drive", fmt.Sprintf("file=%s,media=cdrom,if=none,id=cdrom", cfg.ISOPath))
			args = append(args, "-device", "virtio-blk-pci,drive=cdrom")
			args = append(args, "-boot", "d")
		}
	} else {
		// x86_64
		args = append(args, "-M", "q35")
		if cfg.EnableAccel && isDarwinX86 {
			args = append(args, "-accel", "hvf", "-cpu", "host")
		} else if cfg.EnableAccel && isLinuxX86 {
			args = append(args, "-accel", "kvm", "-cpu", "host")
		} else {
			args = append(args, "-accel", "tcg,thread=multi,tb-size=512", "-cpu", "Nehalem")
		}

		// Display & Devices
		if runtime.GOOS == "darwin" {
			args = append(args, "-display", "cocoa")
		}
		args = append(args, "-vga", "virtio")
		args = append(args, "-usb", "-device", "usb-tablet")
		args = append(args, "-netdev", "user,id=net0", "-device", "virtio-net-pci,netdev=net0")

		// Storage
		if cfg.DiskPath != "" {
			args = append(args, "-drive", fmt.Sprintf("file=%s,format=%s,if=virtio", cfg.DiskPath, diskFormat))
		}
		if cfg.ISOPath != "" {
			args = append(args, "-cdrom", cfg.ISOPath)
			args = append(args, "-boot", "d")
		}
	}

	// Extra custom arguments
	if strings.TrimSpace(cfg.ExtraArgs) != "" {
		extras := strings.Fields(cfg.ExtraArgs)
		args = append(args, extras...)
	}

	return bin, args, nil
}

// StartVM launches a VM process.
func (m *Manager) StartVM(cfg VMConfig) (*VMState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if p, ok := m.processes[cfg.ID]; ok {
		if p.Cmd != nil && p.Cmd.Process != nil {
			return nil, fmt.Errorf("a máquina virtual %q já está em execução (PID %d)", cfg.Name, p.Cmd.Process.Pid)
		}
	}

	// Auto-correct if an .iso was inadvertently passed as a hard disk
	if cfg.ISOPath == "" && strings.HasSuffix(strings.ToLower(cfg.DiskPath), ".iso") {
		cfg.ISOPath = cfg.DiskPath
		cfg.DiskPath = ""
	}

	// Auto-create disk if requested and not yet existing
	if cfg.CreateDisk && cfg.DiskPath == "" {
		diskName := fmt.Sprintf("%s.qcow2", sanitizeFileName(cfg.Name))
		size := cfg.DiskSizeGB
		if size <= 0 {
			size = 20
		}
		path, err := m.CreateDisk(diskName, size)
		if err == nil {
			cfg.DiskPath = path
		}
	}

	bin, args, err := m.BuildCommand(cfg)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar stdout do QEMU: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar stderr do QEMU: %w", err)
	}

	if err := cmd.Start(); err != nil {
		state := VMState{
			ID:      cfg.ID,
			Status:  "error",
			LastLog: err.Error(),
		}
		m.states[cfg.ID] = state
		m.emitEvent("vms:status", state)
		return nil, fmt.Errorf("falha ao iniciar processo do QEMU: %w", err)
	}

	handle := &ProcessHandle{
		Config:    cfg,
		Cmd:       cmd,
		StartedAt: time.Now(),
		Cancel:    make(chan struct{}),
		Logs:      make([]string, 0, 100),
	}
	m.processes[cfg.ID] = handle

	state := VMState{
		ID:        cfg.ID,
		Status:    "running",
		PID:       cmd.Process.Pid,
		StartedAt: handle.StartedAt.Format(time.RFC3339),
		UptimeSec: 0,
	}
	m.states[cfg.ID] = state
	m.emitEvent("vms:status", state)

	// Stream logs in goroutines
	go m.consumeLogStream(cfg.ID, stdout)
	go m.consumeLogStream(cfg.ID, stderr)

	// Monitor process termination
	go func(id string, c *exec.Cmd) {
		waitErr := c.Wait()

		m.mu.Lock()
		defer m.mu.Unlock()

		delete(m.processes, id)

		currentState := m.states[id]
		if waitErr != nil {
			currentState.Status = "error"
			currentState.LastLog = fmt.Sprintf("Processo finalizado com erro: %v", waitErr)
		} else {
			currentState.Status = "stopped"
		}
		currentState.PID = 0
		m.states[id] = currentState
		m.emitEvent("vms:status", currentState)
	}(cfg.ID, cmd)

	return &state, nil
}

// StopVM terminates a running VM.
func (m *Manager) StopVM(id string, force bool) error {
	m.mu.Lock()
	p, ok := m.processes[id]
	m.mu.Unlock()

	if !ok || p.Cmd == nil || p.Cmd.Process == nil {
		return fmt.Errorf("máquina virtual %s não está em execução", id)
	}

	if force {
		return p.Cmd.Process.Kill()
	}

	// Try graceful SIGTERM (on UNIX)
	return p.Cmd.Process.Signal(os.Interrupt)
}

func (m *Manager) GetLogs(id string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.processes[id]; ok {
		cp := make([]string, len(p.Logs))
		copy(cp, p.Logs)
		return cp
	}
	return nil
}

func (m *Manager) GetAllStates() map[string]VMState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[string]VMState, len(m.states))
	for k, v := range m.states {
		if p, ok := m.processes[k]; ok && v.Status == "running" {
			v.UptimeSec = int(time.Since(p.StartedAt).Seconds())
		}
		res[k] = v
	}
	return res
}

func (m *Manager) consumeLogStream(id string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		m.mu.Lock()
		if p, ok := m.processes[id]; ok {
			if len(p.Logs) > 300 {
				p.Logs = p.Logs[len(p.Logs)-200:]
			}
			p.Logs = append(p.Logs, text)
		}
		m.mu.Unlock()

		m.emitEvent("vms:log", map[string]any{
			"id":   id,
			"line": text,
			"time": time.Now().Format("15:04:05"),
		})
	}
}

func (m *Manager) emitEvent(event string, data any) {
	if m.onEvent != nil {
		m.onEvent(event, data)
	}
}

func sanitizeFileName(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.ReplaceAll(clean, " ", "_")
	var b strings.Builder
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "vm"
	}
	return b.String()
}

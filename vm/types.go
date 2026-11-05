package vm

// VMConfig defines the user configuration for a virtual machine.
type VMConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name" validate:"required,min=2,max=50"`
	Arch        string `json:"arch" validate:"required,oneof=x86_64 aarch64"`
	Cores       int    `json:"cores" validate:"min=1,max=64"`
	MemoryMB    int    `json:"memoryMB" validate:"min=256,max=131072"`
	ISOPath     string `json:"isoPath"`
	DiskPath    string `json:"diskPath"`
	DiskSizeGB  int    `json:"diskSizeGB"`
	CreateDisk  bool   `json:"createDisk"`
	EnableAccel bool   `json:"enableAccel"`
	ExtraArgs   string `json:"extraArgs"`
	CreatedAt   string `json:"createdAt"`
}

// VMState represents runtime process information for a virtual machine.
type VMState struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // "stopped", "starting", "running", "error"
	PID       int    `json:"pid"`
	StartedAt string `json:"startedAt,omitempty"`
	UptimeSec int    `json:"uptimeSec"`
	LastLog   string `json:"lastLog,omitempty"`
}

// SystemInfo represents host system capabilities and QEMU detection.
type SystemInfo struct {
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	HostCPUs        int      `json:"hostCpus"`
	HostMemoryMB    int      `json:"hostMemoryMB"`
	HostDiskFreeGB  int      `json:"hostDiskFreeGB"`
	HostDiskTotalGB int      `json:"hostDiskTotalGB"`
	QemuVersion     string   `json:"qemuVersion"`
	QemuX86Path     string   `json:"qemuX86Path"`
	QemuArmPath     string   `json:"qemuArmPath"`
	QemuImgPath     string   `json:"qemuImgPath"`
	Accelerators    []string `json:"accelerators"`
	Edk2ArmPath     string   `json:"edk2ArmPath"`
	Edk2X86Path     string   `json:"edk2X86Path"`
	DisksDir        string   `json:"disksDir"`
}

// Request / Response DTOs
type SaveVMRequest struct {
	VM VMConfig `json:"vm" validate:"required"`
}

type StartVMRequest struct {
	ID    string    `json:"id"`
	AdHoc *VMConfig `json:"adHoc,omitempty"` // For Quick Boot without saving
}

type StopVMRequest struct {
	ID    string `json:"id" validate:"required"`
	Force bool   `json:"force"`
}

type DeleteVMRequest struct {
	ID         string `json:"id" validate:"required"`
	DeleteDisk bool   `json:"deleteDisk"`
}

type CreateDiskRequest struct {
	Name   string `json:"name" validate:"required"`
	SizeGB int    `json:"sizeGB" validate:"min=1,max=1000"`
}

type CreateDiskResponse struct {
	Path   string `json:"path"`
	SizeGB int    `json:"sizeGB"`
}

type BuildCommandResponse struct {
	Binary  string   `json:"binary"`
	Args    []string `json:"args"`
	Command string   `json:"command"`
}

type VMListResponse struct {
	Items  []VMConfig         `json:"items"`
	States map[string]VMState `json:"states"`
}

type LogResponse struct {
	Lines []string `json:"lines"`
}

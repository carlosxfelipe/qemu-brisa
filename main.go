package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/carlosxfelipe/brisa"
	"qemu-brisa/vm"
)

//go:embed ui assets/icon.png
var uiAssets embed.FS

type SystemService struct {
	app *brisa.App
	mgr *vm.Manager
}

type BrowseResponse struct {
	Path string `json:"path"`
}

func (s *SystemService) Info() (vm.SystemInfo, error) {
	return s.mgr.GetSystemInfo(), nil
}

func (s *SystemService) BrowseISO() (BrowseResponse, error) {
	selected, err := s.app.Dialog().OpenFile(brisa.FileDialogOptions{
		Title: "Selecionar Imagem ISO / Disco",
		Filters: []brisa.FileFilter{
			{Name: "Imagens de Disco & ISO", Patterns: []string{"*.iso", "*.img", "*.dmg", "*.raw", "*.qcow2"}},
			{Name: "Todos os Arquivos", Patterns: []string{"*.*"}},
		},
	})
	if err != nil {
		return BrowseResponse{}, err
	}
	return BrowseResponse{Path: selected}, nil
}

func (s *SystemService) BrowseDisk() (BrowseResponse, error) {
	selected, err := s.app.Dialog().OpenFile(brisa.FileDialogOptions{
		Title: "Selecionar Disco Virtual",
		Filters: []brisa.FileFilter{
			{Name: "Discos Virtuais QCOW2 / RAW", Patterns: []string{"*.qcow2", "*.img", "*.raw"}},
			{Name: "Todos os Arquivos", Patterns: []string{"*.*"}},
		},
	})
	if err != nil {
		return BrowseResponse{}, err
	}
	return BrowseResponse{Path: selected}, nil
}

func main() {
	headless := os.Getenv("BRISA_HEADLESS") == "1"

	var appRef *brisa.App

	// Manager with event broadcaster to frontend SSE
	mgr := vm.NewManager(func(event string, data any) {
		if appRef != nil {
			appRef.Emit(event, data)
		}
	})

	app := brisa.New(brisa.Config{
		AppID:              "com.carlosxfelipe.qemu-brisa",
		Title:              "QEMU Brisa — Virtual Machine Manager",
		Version:            "1.0.0",
		Description:        "Gerenciador de Máquinas Virtuais QEMU com interface moderna em Go + Brisa.",
		Icon:               "assets/icon.png",
		Theme:              brisa.ThemeMacOS,
		SilenceTagWarnings: true,
		Assets:             uiAssets,
		AssetsSubDir:       "ui",
		Window: brisa.WindowConfig{
			Title:     "QEMU Brisa",
			Width:     1180,
			Height:    780,
			MinWidth:  840,
			MinHeight: 560,
			Headless:  headless,
		},
		Store: brisa.StoreOptions{
			EncryptionKey: "qemu-brisa-local-secret-key",
		},
	})
	appRef = app

	app.Use(brisa.Recover(), brisa.Logger())

	// 1. System service bindings
	sysService := &SystemService{app: app, mgr: mgr}
	brisa.BindService(app, "system", sysService)

	// 2. VM command group
	vms := app.Group("vms").Tags("VirtualMachines")

	// List all saved VMs + runtime states
	brisa.Bind(vms, "list", func(ctx *brisa.Context, _ brisa.Empty) (vm.VMListResponse, error) {
		var list []vm.VMConfig
		_, _ = ctx.Store().Get("vms", &list)
		states := mgr.GetAllStates()
		return vm.VMListResponse{
			Items:  list,
			States: states,
		}, nil
	}).Summary("List all saved virtual machines and their current states")

	// Save or update a VM profile
	brisa.Bind(vms, "save", func(ctx *brisa.Context, req vm.SaveVMRequest) (vm.VMConfig, error) {
		var list []vm.VMConfig
		_, _ = ctx.Store().Get("vms", &list)

		item := req.VM
		if item.ID == "" {
			item.ID = fmt.Sprintf("vm_%d", time.Now().UnixMilli())
			item.CreatedAt = time.Now().Format("02/01/2006 15:04")
			list = append([]vm.VMConfig{item}, list...)
		} else {
			updated := false
			for i, cur := range list {
				if cur.ID == item.ID {
					if item.CreatedAt == "" {
						item.CreatedAt = cur.CreatedAt
					}
					list[i] = item
					updated = true
					break
				}
			}
			if !updated {
				list = append([]vm.VMConfig{item}, list...)
			}
		}

		if err := ctx.Store().Set("vms", list); err != nil {
			return vm.VMConfig{}, err
		}

		ctx.Emit("vms:saved", item)
		return item, nil
	}).Summary("Save or update a virtual machine configuration")

	// Delete a VM profile
	brisa.Bind(vms, "delete", func(ctx *brisa.Context, req vm.DeleteVMRequest) (brisa.Map, error) {
		_ = mgr.StopVM(req.ID, true)

		var list []vm.VMConfig
		_, _ = ctx.Store().Get("vms", &list)

		filtered := make([]vm.VMConfig, 0, len(list))
		var target *vm.VMConfig
		for _, item := range list {
			if item.ID == req.ID {
				target = &item
			} else {
				filtered = append(filtered, item)
			}
		}
		_ = ctx.Store().Set("vms", filtered)

		// Optionally delete associated disk
		if req.DeleteDisk && target != nil && target.DiskPath != "" {
			_ = os.Remove(target.DiskPath)
		}

		ctx.Emit("vms:deleted", brisa.Map{"id": req.ID})
		return brisa.Map{"deleted": req.ID}, nil
	}).Summary("Delete a virtual machine configuration")

	// Start a VM
	brisa.Bind(vms, "start", func(ctx *brisa.Context, req vm.StartVMRequest) (vm.VMState, error) {
		var targetCfg vm.VMConfig

		if req.AdHoc != nil {
			targetCfg = *req.AdHoc
			if targetCfg.ID == "" {
				targetCfg.ID = fmt.Sprintf("quick_%d", time.Now().UnixMilli())
			}
		} else {
			var list []vm.VMConfig
			_, _ = ctx.Store().Get("vms", &list)
			found := false
			for _, item := range list {
				if item.ID == req.ID {
					targetCfg = item
					found = true
					break
				}
			}
			if !found {
				return vm.VMState{}, fmt.Errorf("máquina virtual com ID %q não encontrada", req.ID)
			}
		}

		state, err := mgr.StartVM(targetCfg)
		if err != nil {
			return vm.VMState{}, err
		}

		_ = app.Notify(brisa.NotificationOptions{
			Title: "QEMU Brisa",
			Body:  fmt.Sprintf("Máquina %q iniciada com sucesso (PID %d)", targetCfg.Name, state.PID),
		})

		return *state, nil
	}).Summary("Start a virtual machine")

	// Stop a running VM
	brisa.Bind(vms, "stop", func(ctx *brisa.Context, req vm.StopVMRequest) (brisa.Map, error) {
		if err := mgr.StopVM(req.ID, req.Force); err != nil {
			return nil, err
		}

		_ = app.Notify(brisa.NotificationOptions{
			Title: "QEMU Brisa",
			Body:  "Máquina virtual parada com sucesso.",
		})

		return brisa.Map{"stopped": req.ID}, nil
	}).Summary("Stop a running virtual machine")

	// Build & preview QEMU command
	brisa.Bind(vms, "buildCommand", func(ctx *brisa.Context, req vm.SaveVMRequest) (vm.BuildCommandResponse, error) {
		bin, args, err := mgr.BuildCommand(req.VM)
		if err != nil {
			return vm.BuildCommandResponse{}, err
		}

		cmdString := filepath.Base(bin)
		for _, arg := range args {
			if len(arg) > 0 && (arg[0] == '-' || !containsAny(arg, " \t\n\"'")) {
				cmdString += " " + arg
			} else {
				cmdString += " \"" + arg + "\""
			}
		}

		return vm.BuildCommandResponse{
			Binary:  bin,
			Args:    args,
			Command: cmdString,
		}, nil
	}).Summary("Generate command line representation for preview")

	// Create disk image
	brisa.Bind(vms, "createDisk", func(ctx *brisa.Context, req vm.CreateDiskRequest) (vm.CreateDiskResponse, error) {
		path, err := mgr.CreateDisk(req.Name, req.SizeGB)
		if err != nil {
			return vm.CreateDiskResponse{}, err
		}
		return vm.CreateDiskResponse{
			Path:   path,
			SizeGB: req.SizeGB,
		}, nil
	}).Summary("Create a virtual QCOW2 disk image")

	// Get logs for a VM
	brisa.Bind(vms, "getLogs", func(ctx *brisa.Context, req vm.StopVMRequest) (vm.LogResponse, error) {
		logs := mgr.GetLogs(req.ID)
		return vm.LogResponse{Lines: logs}, nil
	}).Summary("Get execution logs for a virtual machine")

	// Background ticker for uptime & status telemetry
	app.OnReady(func(a *brisa.App) {
		log.Printf("🖥️ QEMU Brisa pronto em %s (Inspector: %s/brisa/inspector)", a.BaseURL(), a.BaseURL())

		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-a.Window().Done():
					return
				case <-ticker.C:
					states := mgr.GetAllStates()
					if len(states) > 0 {
						a.Emit("vms:telemetry", states)
					}
				}
			}
		}()
	})

	app.OnStartup(func(ctx context.Context) error {
		return nil
	})

	log.Fatal(app.Run())
}

func containsAny(s, chars string) bool {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return true
			}
		}
	}
	return false
}

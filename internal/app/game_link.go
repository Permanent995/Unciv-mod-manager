package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UncivInfo holds information about the detected Unciv installation.
type UncivInfo struct {
	HasExe       bool `json:"hasExe"`
	HasJar       bool `json:"hasJar"`
	HasModsDir   bool `json:"hasModsDir"`
	HasSettings  bool `json:"hasSettings"`
	IsValid      bool `json:"isValid"`
}

// ValidateUncivPath checks if the given path is a valid Unciv installation.
func (a *App) ValidateUncivPath(path string) UncivInfo {
	info := UncivInfo{}
	if path == "" {
		return info
	}

	// Check for Unciv.exe
	if _, err := os.Stat(filepath.Join(path, "Unciv.exe")); err == nil {
		info.HasExe = true
		info.IsValid = true
	}

	// Check for Unciv.jar
	if _, err := os.Stat(filepath.Join(path, "Unciv.jar")); err == nil {
		info.HasJar = true
		info.IsValid = true
	}

	// Check for mods/ directory
	if fi, err := os.Stat(filepath.Join(path, "mods")); err == nil && fi.IsDir() {
		info.HasModsDir = true
		info.IsValid = true
	}

	// Check for GameSettings.json
	if _, err := os.Stat(filepath.Join(path, "GameSettings.json")); err == nil {
		info.HasSettings = true
		info.IsValid = true
	}

	return info
}

// AutoDetectUncivPath tries common locations to find Unciv.
func (a *App) AutoDetectUncivPath() string {
	paths := a.getCandidatePaths()
	for _, p := range paths {
		if info := a.ValidateUncivPath(p); info.IsValid {
			return p
		}
	}
	return ""
}

// SelectUncivDir opens a folder picker dialog and validates the selected path.
func (a *App) SelectUncivDir() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 Unciv 安装目录",
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("未选择目录")
	}

	info := a.ValidateUncivPath(path)
	if !info.IsValid {
		return "", fmt.Errorf("未找到有效的 Unciv 安装（需要 Unciv.exe、Unciv.jar 或 mods/ 目录）")
	}

	if err := a.SetUncivPath(path); err != nil {
		return "", err
	}

	return path, nil
}

// GetUncivInfo returns the detected Unciv installation info.
func (a *App) GetUncivInfo() map[string]interface{} {
	if a.config.UncivPath == "" {
		return map[string]interface{}{"found": false}
	}
	info := a.ValidateUncivPath(a.config.UncivPath)
	return map[string]interface{}{
		"found":  info.IsValid,
		"path":   a.config.UncivPath,
		"hasExe": info.HasExe,
		"hasJar": info.HasJar,
	}
}

// IsUncivRunning checks if Unciv is currently running.
func (a *App) IsUncivRunning() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32Snapshot := kernel32.NewProc("CreateToolhelp32Snapshot")
	process32First := kernel32.NewProc("Process32FirstW")
	process32Next := kernel32.NewProc("Process32NextW")
	closeHandle := kernel32.NewProc("CloseHandle")

	snapshot, _, _ := createToolhelp32Snapshot.Call(2, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return false
	}
	defer closeHandle.Call(snapshot)

	var procEntry syscall.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))

	ret, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&procEntry)))
	if ret == 0 {
		return false
	}

	for {
		exeName := syscall.UTF16ToString(procEntry.ExeFile[:])
		if exeName == "Unciv.exe" || exeName == "java.exe" {
			return true
		}

		ret, _, _ = process32Next.Call(snapshot, uintptr(unsafe.Pointer(&procEntry)))
		if ret == 0 {
			break
		}
	}

	return false
}

// LaunchUnciv starts the Unciv game process.
func (a *App) LaunchUnciv() error {
	if a.config.UncivPath == "" {
		return fmt.Errorf("未设置 Unciv 路径")
	}
	// Try .exe first
	exe := filepath.Join(a.config.UncivPath, "Unciv.exe")
	if _, err := os.Stat(exe); err == nil {
		return exec.Command(exe).Start()
	}
	// Fallback to .jar
	jar := filepath.Join(a.config.UncivPath, "Unciv.jar")
	if _, err := os.Stat(jar); err == nil {
		return exec.Command("java", "-jar", jar).Start()
	}
	return fmt.Errorf("未找到 Unciv.exe 或 Unciv.jar")
}

// getCandidatePaths returns common Unciv installation paths to check.
func (a *App) getCandidatePaths() []string {
	var paths []string

	// Current exe directory
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}

	// User's Desktop
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Desktop"), filepath.Join(home, "Desktop", "官方unciv文件"))
	}

	// Program Files
	paths = append(paths,
		filepath.Join("C:", "Program Files", "Unciv"),
		filepath.Join("C:", "Program Files (x86)", "Unciv"),
	)

	// Add saved paths from config
	for _, p := range a.config.SavedPaths {
		paths = append(paths, p)
	}

	return paths
}

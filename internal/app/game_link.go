package app

import (
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetUncivVersion reads the Unciv version from the jar's manifest.
func (a *App) GetUncivVersion() string {
	jarPath := filepath.Join(a.config.UncivPath, "Unciv.jar")
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		// Fallback: try GameSettings.json
		return a.readVersionFromSettings()
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(f.Name, "META-INF/MANIFEST.MF") {
			rc, err := f.Open()
			if err != nil {
				return a.readVersionFromSettings()
			}
			defer rc.Close()
			buf := make([]byte, 4096)
			n, _ := rc.Read(buf)
			content := string(buf[:n])
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(line, "Specification-Version:") {
					v := strings.TrimSpace(strings.TrimPrefix(line, "Specification-Version:"))
					if v != "" {
						return v
					}
				}
			}
		}
	}
	return a.readVersionFromSettings()
}

func (a *App) readVersionFromSettings() string {
	path := filepath.Join(a.config.UncivPath, "GameSettings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// Try lastGameSetup.mapParameters.createdWithVersion
	// The file is JSON but we don't want to import gjson here,
	// use a simple string fallback approach
	content := string(data)
	marker := `"createdWithVersion":"`
	i := strings.Index(content, marker)
	if i < 0 {
		return ""
	}
	i += len(marker)
	j := strings.Index(content[i:], `"`)
	if j < 0 {
		return ""
	}
	return content[i : i+j]
}

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

// UncivPathOption is a detected Unciv installation.
type UncivPathOption struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	HasExe  bool   `json:"hasExe"`
	HasJar  bool   `json:"hasJar"`
}

// AutoDetectUncivPaths scans common locations and returns ALL valid Unciv
// installations found, so the user can choose between multiple versions.
func (a *App) AutoDetectUncivPaths() []UncivPathOption {
	seen := map[string]bool{}
	var results []UncivPathOption

	for _, p := range a.getCandidatePaths() {
		// Resolve to absolute to deduplicate
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		seen[abs] = true

		if info := a.ValidateUncivPath(abs); info.IsValid {
			// Temporarily set path so version detection works
			oldPath := a.config.UncivPath
			a.config.UncivPath = abs
			ver := a.GetUncivVersion()
			a.config.UncivPath = oldPath

			results = append(results, UncivPathOption{
				Path:    abs,
				Version: ver,
				HasExe:  info.HasExe,
				HasJar:  info.HasJar,
			})
		}
	}

	if results == nil {
		results = []UncivPathOption{}
	}
	return results
}

// AutoDetectUncivPath tries common locations to find Unciv.  Legacy wrapper;
// prefer AutoDetectUncivPaths for multi-version detection.
func (a *App) AutoDetectUncivPath() string {
	opts := a.AutoDetectUncivPaths()
	if len(opts) > 0 {
		return opts[0].Path
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

// MigrateUncivData copies mods, saves, and maps from one Unciv installation
// to another.  Existing files in the destination are NOT overwritten.
// GameSettings.json and binaries are deliberately excluded.
func (a *App) MigrateUncivData(fromPath, toPath string) (map[string]int, error) {
	result := map[string]int{"mods": 0, "saves": 0, "maps": 0}

	dirs := []struct {
		src string
		key string
	}{
		{filepath.Join(fromPath, "mods"), "mods"},
		{filepath.Join(fromPath, "SaveFiles"), "saves"},
		{filepath.Join(fromPath, "maps"), "maps"},
	}

	for _, d := range dirs {
		srcDir := d.src
		dstDir := filepath.Join(toPath, filepath.Base(srcDir))

		if _, err := os.Stat(srcDir); err != nil {
			continue // source dir doesn't exist — nothing to copy
		}
		os.MkdirAll(dstDir, 0755)

		entries, err := os.ReadDir(srcDir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			src := filepath.Join(srcDir, e.Name())
			dst := filepath.Join(dstDir, e.Name())

			// Skip if destination already exists (incremental, no overwrite)
			if _, err := os.Stat(dst); err == nil {
				continue
			}

			if e.IsDir() {
				if err := copyDir(src, dst); err == nil {
					result[d.key]++
				}
			} else {
				if err := copyFile(src, dst); err == nil {
					result[d.key]++
				}
			}
		}
	}

	return result, nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	os.MkdirAll(dst, 0755)
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
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

	if home, err := os.UserHomeDir(); err == nil {
		desktop := filepath.Join(home, "Desktop")
		paths = append(paths, desktop)

		// Scan Desktop subdirectories for Unciv installations.
		// Many users keep multiple versions in folders like "Unciv 4.12.2".
		if entries, err := os.ReadDir(desktop); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := strings.ToLower(e.Name())
				if strings.Contains(name, "unciv") {
					paths = append(paths, filepath.Join(desktop, e.Name()))
				}
			}
		}

		// Other common locations
		paths = append(paths,
			filepath.Join(home, "Desktop", "官方unciv文件"),
			filepath.Join(home, "Downloads"),
			filepath.Join(home, "Documents"),
		)

		// Scan Downloads for Unciv folders too
		if entries, err := os.ReadDir(filepath.Join(home, "Downloads")); err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.Contains(strings.ToLower(e.Name()), "unciv") {
					paths = append(paths, filepath.Join(home, "Downloads", e.Name()))
				}
			}
		}
	}

	// Program Files
	paths = append(paths,
		filepath.Join("C:", "Program Files", "Unciv"),
		filepath.Join("C:", "Program Files (x86)", "Unciv"),
		filepath.Join("D:", "Games", "Unciv"),
	)

	// Add saved paths from config
	for _, p := range a.config.SavedPaths {
		paths = append(paths, p)
	}

	return paths
}

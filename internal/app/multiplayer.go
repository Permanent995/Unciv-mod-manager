package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ModSnapshot is a lightweight mod list for multiplayer comparison.
type ModSnapshot struct {
	UncivVersion string          `json:"uncivVersion"`
	Mods         []ModSnapshotItem `json:"mods"`
}

// ModSnapshotItem captures enough info to compare two mod setups.
// Folder is the primary key for matching; Name/Version are advisory.
type ModSnapshotItem struct {
	Folder  string `json:"folder"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  string `json:"author"`
	ModSize int    `json:"modSize"`
	Category string `json:"category"`
}

// MultiplayerDiff describes a mismatch between two mod setups.
type MultiplayerDiff struct {
	Mod      string `json:"mod"`
	Issue    string `json:"issue"` // "missing_in_a" | "missing_in_b" | "version_mismatch"
	ValueA   string `json:"valueA"`
	ValueB   string `json:"valueB"`
}

// ExportModSnapshot writes the current mod list to a JSON file for sharing.
func (a *App) ExportModSnapshot() (string, error) {
	mods, err := a.ScanMods()
	if err != nil {
		return "", err
	}
	var items []ModSnapshotItem
	for _, m := range mods {
		items = append(items, ModSnapshotItem{
			Folder:  m.Folder,
			Name:    m.Name,
			Version: m.LastUpdated,
			Author:  m.Author,
		})
	}
	snap := ModSnapshot{Mods: items}
	// Try to read Unciv version from properties
	snap.UncivVersion = a.readUncivVersion()

	data, _ := json.MarshalIndent(snap, "", "  ")
	outPath := filepath.Join(a.configDir, "umm_mod_snapshot.json")
	// Ensure config directory exists
	os.MkdirAll(a.configDir, 0755)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", err
	}
	return outPath, nil
}

// CompareModSnapshot loads a friend's snapshot and diffs against the local setup.
func (a *App) CompareModSnapshot(snapPath string) ([]MultiplayerDiff, error) {
	data, err := os.ReadFile(snapPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取快照文件: %w", err)
	}
	var remote ModSnapshot
	if err := json.Unmarshal(data, &remote); err != nil {
		return nil, fmt.Errorf("快照格式错误: %w", err)
	}

	local := map[string]ModSnapshotItem{} // folder → item
	mods, _ := a.ScanMods()
	for _, m := range mods {
		local[m.Folder] = ModSnapshotItem{Folder: m.Folder, Name: m.Name, Version: m.LastUpdated}
	}

	remoteMap := map[string]ModSnapshotItem{}
	for _, m := range remote.Mods {
		remoteMap[m.Folder] = m
	}

	var diffs []MultiplayerDiff

	// Mods in local but missing in remote
	for folder, item := range local {
		if _, ok := remoteMap[folder]; !ok {
			label := item.Name
			if label == "" { label = folder }
			diffs = append(diffs, MultiplayerDiff{
				Mod: label, Issue: "missing_in_remote",
				ValueA: item.Version, ValueB: "-",
			})
		}
	}
	// Mods in remote but missing in local
	for folder, item := range remoteMap {
		label := item.Name
		if label == "" { label = folder }
		if localItem, ok := local[folder]; !ok {
			diffs = append(diffs, MultiplayerDiff{
				Mod: label, Issue: "missing_in_local",
				ValueA: "-", ValueB: item.Version,
			})
		} else if item.Version != "" && localItem.Version != "" && item.Version != localItem.Version {
			diffs = append(diffs, MultiplayerDiff{
				Mod: label, Issue: "version_mismatch",
				ValueA: localItem.Version, ValueB: item.Version,
			})
		}
	}

	// Version check
	localVer := a.readUncivVersion()
	if remote.UncivVersion != "" && localVer != "" && remote.UncivVersion != localVer {
		diffs = append([]MultiplayerDiff{{
			Mod: "(Unciv 版本)", Issue: "version_mismatch",
			ValueA: localVer, ValueB: remote.UncivVersion,
		}}, diffs...)
	}

	if diffs == nil {
		diffs = []MultiplayerDiff{}
	}
	return diffs, nil
}

func (a *App) readUncivVersion() string {
	// Try to read version from Unciv.jar manifest or properties
	propsPath := filepath.Join(a.config.UncivPath, "Simplified_Chinese.properties")
	if data, err := os.ReadFile(propsPath); err == nil {
		// Look for a version line — usually not there, but worth trying
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "GameVersion") || strings.HasPrefix(line, "version") {
				return strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
			}
		}
	}
	return "未知"
}

// OpenSnapshotFolder opens the UMM config directory in Explorer and returns its path.
func (a *App) OpenSnapshotFolder() (string, error) {
	dir := a.configDir
	// Ensure the directory exists before trying to open it
	if err := os.MkdirAll(dir, 0755); err != nil {
		return dir, fmt.Errorf("无法创建目录: %w", err)
	}
	// Actually open the folder (more reliable than frontend BrowserOpenURL for local paths)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("explorer", dir)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", dir)
	} else {
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return dir, fmt.Errorf("无法打开目录: %w", err)
	}
	return dir, nil
}

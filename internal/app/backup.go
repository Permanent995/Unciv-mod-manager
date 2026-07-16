package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type ModBackup struct {
	Folder    string `json:"folder"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"` // from ModOptions.json lastUpdated or release tag
	Path      string `json:"path"`
	Size      int64  `json:"size"`
}

// BackupMod creates a timestamped backup of a mod folder before updating.
// Stores version info from ModOptions.json alongside the backup.
func (a *App) BackupMod(modFolder, version string) (string, error) {
	src := filepath.Join(a.config.UncivPath, "mods", modFolder)
	if _, err := os.Stat(src); err != nil {
		return "", nil
	}
	// Read version from ModOptions if not provided
	if version == "" {
		version = readModVersion(src)
	}
	backupRoot := filepath.Join(a.config.UncivPath, "umm_backups")
	os.MkdirAll(backupRoot, 0755)

	ts := time.Now().Format("2006-01-02_150405")
	dst := filepath.Join(backupRoot, modFolder+"_"+ts)
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("备份失败: %w", err)
	}
	// Write backup info
	info := map[string]string{"version": version, "folder": modFolder, "timestamp": ts}
	data, _ := json.Marshal(info)
	os.WriteFile(filepath.Join(dst, "_backup_info.json"), data, 0644)

	return dst, nil
}

// ListBackups returns all backups across all mods, sorted by time desc.
func (a *App) ListBackups() ([]ModBackup, error) {
	backupRoot := filepath.Join(a.config.UncivPath, "umm_backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return []ModBackup{}, nil
	}
	var backups []ModBackup
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		infoPath := filepath.Join(backupRoot, e.Name(), "_backup_info.json")
		version := ""
		modFolder := e.Name()
		if data, err := os.ReadFile(infoPath); err == nil {
			var info map[string]string
			json.Unmarshal(data, &info)
			version = info["version"]
			if f, ok := info["folder"]; ok {
				modFolder = f
			}
		}
		var size int64
		filepath.Walk(filepath.Join(backupRoot, e.Name()), func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() && !strings.HasSuffix(fi.Name(), "_backup_info.json") {
				size += fi.Size()
			}
			return nil
		})
		backups = append(backups, ModBackup{
			Folder: modFolder, Timestamp: e.Name(), Version: version,
			Path: filepath.Join(backupRoot, e.Name()), Size: size,
		})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].Timestamp > backups[j].Timestamp })
	return backups, nil
}

// RestoreBackup restores a backup to mods/ and re-backs up the current version first.
func (a *App) RestoreBackup(backupPath string) error {
	// Read backup info
	infoPath := filepath.Join(backupPath, "_backup_info.json")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return fmt.Errorf("备份信息丢失: %w", err)
	}
	var info map[string]string
	json.Unmarshal(data, &info)
	modFolder := info["folder"]
	if modFolder == "" {
		return fmt.Errorf("备份信息不完整")
	}
	target := filepath.Join(a.config.UncivPath, "mods", modFolder)
	// 如果模组已存在，先备份当前版本再删除
	if _, err := os.Stat(target); err == nil {
		currentVersion := readModVersion(target)
		if _, backupErr := a.BackupMod(modFolder, currentVersion); backupErr != nil {
			return fmt.Errorf("备份当前版本失败: %w", backupErr)
		}
		os.RemoveAll(target)
	}
	if err := os.Rename(backupPath, target); err != nil {
		return fmt.Errorf("恢复备份失败: %w", err)
	}
	// 移除备份残留的元数据文件，避免污染模组目录
	os.Remove(filepath.Join(target, "_backup_info.json"))
	return nil
}

// DeleteBackup removes a single backup folder.
func (a *App) DeleteBackup(backupPath string) error {
	return os.RemoveAll(backupPath)
}

// DeleteMod removes a mod folder permanently (call BackupMod first!).
func (a *App) DeleteMod(modFolder string) error {
	modPath := filepath.Join(a.config.UncivPath, "mods", modFolder)
	return os.RemoveAll(modPath)
}

func readModVersion(modPath string) string {
	op := filepath.Join(modPath, "jsons", "ModOptions.json")
	data, err := os.ReadFile(op)
	if err != nil {
		return ""
	}
	return gjson.Get(string(data), "lastUpdated").String()
}

// CleanupModBackupMeta scans the mods directory and removes any UMM backup
// metadata files that may have been left behind by older versions.
func (a *App) CleanupModBackupMeta() {
	modsDir := filepath.Join(a.config.UncivPath, "mods")
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		os.Remove(filepath.Join(modsDir, e.Name(), "_backup_info.json"))
	}
}

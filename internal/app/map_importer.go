package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MapInfo holds metadata about a discovered map file.
type MapInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Source     string `json:"source"` // "maps" | "mod"
	ModFolder  string `json:"modFolder,omitempty"`
}

// SaveClipboardAsMap writes clipboard text content as a timestamped .civ5map file.
func (a *App) SaveClipboardAsMap(text string) error {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return fmt.Errorf("未设置 Unciv 路径")
	}
	mapsDir := filepath.Join(uncivPath, "maps")
	os.MkdirAll(mapsDir, 0755)
	filename := fmt.Sprintf("clipboard_%d.civ5map", time.Now().Unix())
	return os.WriteFile(filepath.Join(mapsDir, filename), []byte(text), 0644)
}

// ScanMaps finds all .civ5map files in maps/ and mods/*/maps/.
func (a *App) ScanMaps() ([]MapInfo, error) {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return nil, fmt.Errorf("未设置 Unciv 路径")
	}

	var maps []MapInfo

	// 1. Top-level maps/ directory
	mapsDir := filepath.Join(uncivPath, "maps")
	if entries, err := os.ReadDir(mapsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && e.Name() != "backup" {
				maps = append(maps, MapInfo{
					Name:   strings.TrimSuffix(e.Name(), ".civ5map"),
					Path:   filepath.Join(mapsDir, e.Name()),
					Source: "maps",
				})
			}
		}
	}

	// 2. mods/*/maps/ — maps shipped with mods
	modsDir := filepath.Join(uncivPath, "mods")
	if entries, err := os.ReadDir(modsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			modMapsDir := filepath.Join(modsDir, e.Name(), "maps")
			if mapEntries, err := os.ReadDir(modMapsDir); err == nil {
				for _, me := range mapEntries {
					if !me.IsDir() {
						maps = append(maps, MapInfo{
							Name:      strings.TrimSuffix(me.Name(), ".civ5map"),
							Path:      filepath.Join(modMapsDir, me.Name()),
							Source:    "mod",
							ModFolder: e.Name(),
						})
					}
				}
			}
		}
	}

	if maps == nil {
		maps = []MapInfo{}
	}
	return maps, nil
}

// ImportFile handles a dragged or selected file — detects type and places
// it in the correct directory.  Returns a human-readable result message.
func (a *App) ImportFile(sourcePath string) (string, error) {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return "", fmt.Errorf("未设置 Unciv 路径")
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))
	name := filepath.Base(sourcePath)

	switch ext {
	case ".civ5map":
		return a.importMapFile(sourcePath, uncivPath, name)

	case ".map":
		outPath, err := a.ConvertWesnothMap(sourcePath)
		if err != nil {
			return "", err
		}
		// Also back up the original
		a.backupUnknownFile(sourcePath, uncivPath, name)
		return fmt.Sprintf("Wesnoth 地图已转换为 %s，原文件已备份", filepath.Base(outPath)), nil

	case ".zip":
		return a.importZipFile(sourcePath, uncivPath)

	default:
		return a.backupUnknownFile(sourcePath, uncivPath, name)
	}
}

func (a *App) importMapFile(sourcePath, uncivPath, name string) (string, error) {
	mapsDir := filepath.Join(uncivPath, "maps")
	os.MkdirAll(mapsDir, 0755)
	dest := filepath.Join(mapsDir, name)
	if err := copyFile(sourcePath, dest); err != nil {
		return "", fmt.Errorf("复制地图文件失败: %w", err)
	}
	return fmt.Sprintf("地图 %q 已导入到 maps/", name), nil
}

func (a *App) importZipFile(sourcePath, uncivPath string) (string, error) {
	// Peek inside to decide: mod or map bundle?
	r, err := zipOpen(sourcePath)
	if err != nil {
		return "", fmt.Errorf("无法打开 ZIP: %w", err)
	}
	defer r.Close()

	hasModOptions := false
	hasMap := false
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "ModOptions.json") || strings.Contains(f.Name, "jsons/") {
			hasModOptions = true
		}
		if strings.HasSuffix(strings.ToLower(f.Name), ".civ5map") {
			hasMap = true
		}
	}

	if hasModOptions {
		modsDir := filepath.Join(uncivPath, "mods")
		folder, err := a.ExtractMod(sourcePath, modsDir)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("模组 %q 已导入到 mods/", folder), nil
	}

	if hasMap {
		mapsDir := filepath.Join(uncivPath, "maps")
		os.MkdirAll(mapsDir, 0755)

		r2, _ := zipOpen(sourcePath)
		defer r2.Close()
		for _, f := range r2.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".civ5map") {
				src, _ := f.Open()
				name := filepath.Base(f.Name)
				dest, _ := os.Create(filepath.Join(mapsDir, name))
				io.Copy(dest, src)
				src.Close()
				dest.Close()
			}
		}
		return "地图已从 ZIP 中提取到 maps/", nil
	}

	return "", fmt.Errorf("ZIP 中未找到可识别的模组或地图文件")
}

func (a *App) backupUnknownFile(sourcePath, uncivPath, name string) (string, error) {
	backupDir := filepath.Join(uncivPath, "maps", "backup")
	os.MkdirAll(backupDir, 0755)
	dest := filepath.Join(backupDir, name)
	if err := copyFile(sourcePath, dest); err != nil {
		return "", fmt.Errorf("备份文件失败: %w", err)
	}
	return fmt.Sprintf("已备份 %q 到 maps/backup/（格式暂不支持，可手动处理）", name), nil
}

// ── Helpers ───────────────────────────────────────────────────────────

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(d, s)
	d.Close()
	return err
}

// zipOpen is a thin wrapper so importZipFile can read the file list twice.
func zipOpen(path string) (*zipReadCloser, error) {
	return zip.OpenReader(path)
}
type zipReadCloser = zip.ReadCloser

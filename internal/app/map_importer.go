package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// MapInfo holds metadata about a discovered map file.
type MapInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Source     string `json:"source"` // "maps" | "mod"
	ModFolder  string `json:"modFolder,omitempty"`
}

// SaveClipboardMap writes clipboard text as a .civ5map file with the given name.
// nameHint is the user's desired name (without extension); if empty, falls back
// to the map's internal name or "clipboard_<timestamp>".
// If the file already exists, auto-appends a counter.
func (a *App) SaveClipboardMap(text string, nameHint string) (string, error) {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return "", fmt.Errorf("未设置 Unciv 路径")
	}
	if len(text) < 10 {
		return "", fmt.Errorf("剪贴板内容太短，不是有效的地图")
	}

	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "H4sI") && !strings.HasPrefix(trimmed, "{") {
		return "", fmt.Errorf("剪贴板内容不是有效的地图格式——.civ5map 应为压缩数据或 JSON 对象")
	}

	if strings.HasPrefix(trimmed, "{") {
		if !gjson.Valid(trimmed) {
			return "", fmt.Errorf("剪贴板内容不是有效的 JSON 地图")
		}
		if !gjson.Get(trimmed, "tiles").Exists() && !gjson.Get(trimmed, "width").Exists() {
			return "", fmt.Errorf("剪贴板内容缺少地图数据字段（tiles 或 width）")
		}
	}

	// Determine filename
	name := nameHint
	if name == "" {
		if strings.HasPrefix(trimmed, "{") {
			if n := gjson.Get(trimmed, "name").String(); n != "" {
				name = n
			}
		}
	}
	if name == "" {
		name = fmt.Sprintf("clipboard_%s", time.Now().Format("2006-01-02_150405"))
	}

	mapsDir := filepath.Join(uncivPath, "maps")
	os.MkdirAll(mapsDir, 0755)

	// Dedup: if file exists, append (2), (3), etc.
	filename := name + ".civ5map"
	if _, err := os.Stat(filepath.Join(mapsDir, filename)); err == nil {
		for i := 2; ; i++ {
			filename = fmt.Sprintf("%s (%d).civ5map", name, i)
			if _, err := os.Stat(filepath.Join(mapsDir, filename)); os.IsNotExist(err) {
				break
			}
		}
	}

	return filename, os.WriteFile(filepath.Join(mapsDir, filename), []byte(text), 0644)
}

// DeleteMap removes a .civ5map file from maps/ (not from mods/*/maps/).
func (a *App) DeleteMap(path string) error {
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	mapsDir := filepath.Join(a.config.UncivPath, "maps")
	absDir, _ := filepath.Abs(mapsDir)
	absPath, _ := filepath.Abs(path)
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
		return fmt.Errorf("只能删除 maps/ 目录下的文件")
	}
	return os.Remove(absPath)
}

// RenameMap renames a .civ5map file in maps/. newName is without extension.
// Returns the new filename (with dedup counter if needed).
func (a *App) RenameMap(oldPath string, newName string) (string, error) {
	if oldPath == "" || newName == "" {
		return "", fmt.Errorf("参数为空")
	}
	// Safety: only allow renaming files inside maps/
	mapsDir := filepath.Join(a.config.UncivPath, "maps")
	absDir, _ := filepath.Abs(mapsDir)
	absPath, _ := filepath.Abs(oldPath)
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("只能重命名 maps/ 目录下的文件")
	}

	newPath := filepath.Join(absDir, newName+".civ5map")
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("文件名 %q 已存在", newName+".civ5map")
	}
	if err := os.Rename(absPath, newPath); err != nil {
		return "", fmt.Errorf("重命名失败: %w", err)
	}
	return newName + ".civ5map", nil
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
	backupDir := filepath.Join(a.configDir, "umm_backups", "mapback")
	os.MkdirAll(backupDir, 0755)
	dest := filepath.Join(backupDir, name)
	if err := copyFile(sourcePath, dest); err != nil {
		return "", fmt.Errorf("备份文件失败: %w", err)
	}
	return fmt.Sprintf("已备份 %q 到 mapback/（格式暂不支持，可手动处理）", name), nil
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

func zipOpen(path string) (*zipReadCloser, error) {
	return zip.OpenReader(path)
}
type zipReadCloser = zip.ReadCloser

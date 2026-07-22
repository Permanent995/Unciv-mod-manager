package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

// ScanMods scans the Unciv mods directory and returns a list of ModInfo.
func (a *App) ScanMods() ([]ModInfo, error) {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return nil, fmt.Errorf("未设置 Unciv 路径")
	}

	modsDir := filepath.Join(uncivPath, "mods")
	if _, err := os.Stat(modsDir); err != nil {
		// Unciv 可能没运行过，mods/ 还未创建——不是错误
		return []ModInfo{}, nil
	}

	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil, fmt.Errorf("读取 mods 目录失败: %v", err)
	}

	var mods []ModInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modPath := filepath.Join(modsDir, entry.Name())
		modInfo, err := a.parseModInfo(modPath, entry.Name())
		if err != nil {
			continue // Skip invalid mod folders
		}
		mods = append(mods, modInfo)
	}

	return mods, nil
}

// parseModInfo extracts metadata from a mod folder.
func (a *App) parseModInfo(modPath, folderName string) (ModInfo, error) {
	info := ModInfo{
		Folder: folderName,
		Name:   folderName,
	}

	// Check for README
	if _, err := os.Stat(filepath.Join(modPath, "README.md")); err == nil {
		info.HasReadme = true
	}
	// Check for preview image (lazy loaded on demand, not in list view)
	if _, err := os.Stat(filepath.Join(modPath, "preview.png")); err == nil {
		info.HasPreview = true
	}

	// Check for ModOptions.json at jsons/ModOptions.json
	modOptionsPath := filepath.Join(modPath, "jsons", "ModOptions.json")
	data, err := os.ReadFile(modOptionsPath)
	if err != nil {
		// If no ModOptions.json, check for basic structure
		if hasJSONDir(modPath) {
			info.Category = "unclassified"
			info.IsIncomplete = true
			info.Author = "未提供"
			return info, nil
		}
		return info, fmt.Errorf("no ModOptions.json")
	}

	// Use gjson (tolerant JSON parser) for parsing
	content := string(data)
	info.Name = gjson.Get(content, "name").String()
	if info.Name == "" {
		info.Name = folderName
	}
	info.Author = gjson.Get(content, "author").String()
	info.ModURL = gjson.Get(content, "modUrl").String()
	if info.Author == "" {
		if strings.Contains(info.ModURL, "github.com/") {
			u := strings.TrimPrefix(info.ModURL, "https://")
			u = strings.TrimPrefix(u, "http://")
			u = strings.TrimPrefix(u, "github.com/")
			if idx := strings.IndexByte(u, '/'); idx > 0 {
				info.Author = u[:idx]
			}
		}
		if info.Author == "" {
			info.Author = "未提供"
		}
	}
	info.LastUpdated = gjson.Get(content, "lastUpdated").String()

	// Single walk to collect both size and category heuristics
	scan := scanModDirectory(modPath)
	info.ModSize = int(gjson.Get(content, "modSize").Int())
	if info.ModSize == 0 {
		info.ModSize = scan.totalSize
	}
	info.IsBaseRuleset = gjson.Get(content, "isBaseRuleset").Bool()

	// Parse topics
	topicsResult := gjson.Get(content, "topics")
	topicsResult.ForEach(func(key, value gjson.Result) bool {
		info.Topics = append(info.Topics, value.String())
		return true
	})

	// Determine category (uses pre-scanned booleans, no extra walk)
	info.Category = a.categorizeMod(info.Topics, info.IsBaseRuleset, scan.hasImagesOnly, scan.hasMusic, scan.hasJSONDir)

	return info, nil
}

// categorizeMod determines the mod category based on metadata and folder structure.
func (a *App) categorizeMod(topics []string, isBaseRuleset bool, hasImagesOnly, hasMusic, hasJSONDir bool) string {
	topicSet := make(map[string]bool)
	for _, t := range topics {
		topicSet[t] = true
	}

	if isBaseRuleset {
		return "ruleset"
	}
	if topicSet["unciv-mod-rulesets"] {
		return "ruleset"
	}
	if topicSet["unciv-mod-graphics"] {
		return "graphics"
	}
	if topicSet["unciv-mod-audio"] {
		return "audio"
	}
	if topicSet["unciv-mod-expansions"] {
		return "expansion"
	}
	if topicSet["unciv-mod-fun"] {
		return "fun"
	}
	if topicSet["unciv-mod-maps"] {
		return "map"
	}

	// Fallback: check folder structure (uses pre-scanned booleans, no extra walk)
	if hasImagesOnly {
		return "graphics"
	}
	if hasMusic {
		return "audio"
	}
	if hasJSONDir {
		return "unclassified"
	}

	return "unclassified"
}

// modScanResult holds info collected during a single directory walk.
type modScanResult struct {
	totalSize     int
	hasImagesOnly bool // has images AND no JSON files (pure graphics mod)
	hasMusic      bool
	hasJSONDir    bool
}

// scanModDirectory walks a mod folder once to collect size, image, and music info.
func scanModDirectory(modPath string) modScanResult {
	imageExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".atlas": true}
	var result modScanResult
	var totalSize int64
	foundImage := false
	foundJSON := false
	foundMusic := false

	filepath.Walk(modPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		ext := strings.ToLower(filepath.Ext(info.Name()))
		switch ext {
		case ".json":
			foundJSON = true
		case ".mp3", ".ogg":
			foundMusic = true
		default:
			if imageExts[ext] {
				foundImage = true
			}
		}
		return nil
	})

	result.totalSize = int(totalSize)
	result.hasImagesOnly = foundImage && !foundJSON
	result.hasMusic = foundMusic
	// Check for jsons/ directory (separate Stat, not a filesystem walk)
	_, err := os.Stat(filepath.Join(modPath, "jsons"))
	result.hasJSONDir = err == nil
	return result
}

// hasJSONDir checks if the mod folder has a jsons/ subdirectory.
func hasJSONDir(modPath string) bool {
	_, err := os.Stat(filepath.Join(modPath, "jsons"))
	return err == nil
}

// ReadModPreview returns the base64-encoded preview.png for a mod, or empty.
// Image data is never included in list responses — only fetched on demand
// when the user selects a specific mod in the detail view.
func (a *App) ReadModPreview(folderName string) string {
	modPath := filepath.Join(a.config.UncivPath, "mods", folderName, "preview.png")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return ""
	}
	// Return as base64 data URI for direct use in <img src="...">
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// ReadModReadme returns the content of a mod's README.md, or empty string.
func (a *App) ReadModReadme(folderName string) string {
	modPath := filepath.Join(a.config.UncivPath, "mods", folderName, "README.md")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return ""
	}
	return string(data)
}


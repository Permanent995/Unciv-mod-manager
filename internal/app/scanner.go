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
		return nil, fmt.Errorf("mods 目录不存在: %v", err)
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
	info.ModSize = int(gjson.Get(content, "modSize").Int())
	if info.ModSize == 0 {
		info.ModSize = dirSize(modPath)
	}
	info.IsBaseRuleset = gjson.Get(content, "isBaseRuleset").Bool()

	// Parse topics
	topicsResult := gjson.Get(content, "topics")
	topicsResult.ForEach(func(key, value gjson.Result) bool {
		info.Topics = append(info.Topics, value.String())
		return true
	})

	// Determine category
	info.Category = a.categorizeMod(info.Topics, info.IsBaseRuleset, modPath)

	return info, nil
}

// categorizeMod determines the mod category based on metadata and folder structure.
func (a *App) categorizeMod(topics []string, isBaseRuleset bool, modPath string) string {
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

	// Fallback: check folder structure
	if hasOnlyImageFiles(modPath) {
		return "graphics"
	}
	if hasMusicFiles(modPath) {
		return "audio"
	}
	if hasJSONDir(modPath) {
		return "unclassified"
	}

	return "unclassified"
}

// hasJSONDir checks if the mod folder has a jsons/ subdirectory.
func hasJSONDir(modPath string) bool {
	_, err := os.Stat(filepath.Join(modPath, "jsons"))
	return err == nil
}

// hasOnlyImageFiles checks if the mod folder primarily contains image files.
func hasOnlyImageFiles(modPath string) bool {
	imageExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".atlas": true}
	hasImages := false
	hasJSON := false

	filepath.Walk(modPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(info.Name())) == ".json" {
			hasJSON = true
		}
		if imageExts[strings.ToLower(filepath.Ext(info.Name()))] {
			hasImages = true
		}
		return nil
	})

	return hasImages && !hasJSON
}

// hasMusicFiles checks for MP3/OGG files.
func hasMusicFiles(modPath string) bool {
	found := false
	filepath.Walk(modPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext == ".mp3" || ext == ".ogg" {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
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
	return "data:image/png;base64," + base64Encode(data)
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

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// dirSize returns the total size of all files in a directory in bytes.
func dirSize(path string) int {
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return int(total)
}

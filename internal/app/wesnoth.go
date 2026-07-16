package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// wesnothToCiv5 maps Wesnoth terrain codes to Unciv base terrain names.
// Codes with ^ (overlay) use the overlay code for lookup first.
var wesnothToCiv5 = map[string]string{
	// Flat terrain
	"Gg": "Grassland",
	"Gs": "Grassland",
	"Gd": "Grassland",
	"Gt": "Grassland",
	"Re": "Grassland",
	"Rb": "Grassland",
	"Rd": "Grassland",
	"Rr": "Grassland",
	"Rp": "Grassland",
	"Dd": "Desert",
	"Ds": "Desert",
	"Do": "Oasis",
	"Dt": "Desert",
	"Aa": "Snow",
	"Ha": "Snow",
	"Ms": "Snow",
	// Hills and mountains
	"Hh": "Hill",
	"Ha^": "Hill",
	"Md": "Mountain",
	"Mm": "Mountain",
	"Mr": "Mountain",
	// Water
	"Ww": "Coast",
	"Wo": "Ocean",
	"Wwt": "Coast",
	"Wwg": "Coast",
	"Ss": "Coast",
	// Features (overlays)
	"^Fp": "Forest",
	"^Fd": "Forest",
	"^Ft": "Forest",
	"^Fm": "Forest",
	"^Fds": "Forest",
	"^Uf": "Forest",
	"^Ue": "Grassland",
	"^Bw": "Marsh",
	// Swamp / jungle
	"Ss^": "Coast",
	"Ww^": "Coast",
}

// ConvertWesnothMap reads a Wesnoth .map file and writes a .civ5map file.
// Returns the output path and any error.
func (a *App) ConvertWesnothMap(sourcePath string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("无法读取 Wesnoth 地图: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var grid [][]string
	width := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "border_size") || strings.HasPrefix(line, "usage") {
			continue
		}
		cells := strings.Split(line, ",")
		var row []string
		for _, cell := range cells {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			terrain := wesnothTerrain(cell)
			row = append(row, terrain)
		}
		if len(row) > 0 {
			grid = append(grid, row)
			if len(row) > width {
				width = len(row)
			}
		}
	}

	if len(grid) == 0 {
		return "", fmt.Errorf("Wesnoth 地图为空或格式无法识别")
	}

	// Build output: list of map objects compatible with Unciv
	// Each tile: {"baseTerrain":"Grassland","x":0,"y":0}
	// For simplicity output as a text grid Unciv can import
	var out strings.Builder
	out.WriteString(fmt.Sprintf("Unciv map converted from Wesnoth\n"))
	for y, row := range grid {
		for x, terrain := range row {
			out.WriteString(fmt.Sprintf("%d,%d,%s\n", x, y, terrain))
		}
	}

	// Output to maps/ directory
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return "", fmt.Errorf("未设置 Unciv 路径")
	}
	mapsDir := filepath.Join(uncivPath, "maps")
	os.MkdirAll(mapsDir, 0755)
	baseName := strings.TrimSuffix(filepath.Base(sourcePath), ".map")
	outPath := filepath.Join(mapsDir, baseName+".civ5map")
	if err := os.WriteFile(outPath, []byte(out.String()), 0644); err != nil {
		return "", fmt.Errorf("写入 .civ5map 失败: %w", err)
	}
	return outPath, nil
}

// wesnothTerrain resolves a Wesnoth terrain code (e.g. "Gg^Fp") to
// an Unciv terrain name.  When an overlay (^) is present the overlay
// code is preferred; otherwise the base code is used.
func wesnothTerrain(code string) string {
	code = strings.TrimSpace(code)
	if idx := strings.Index(code, "^"); idx >= 0 {
		if t, ok := wesnothToCiv5[code[idx:]]; ok {
			return t
		}
	}
	if t, ok := wesnothToCiv5[code]; ok {
		return t
	}
	// Fallback: strip overlay suffix and try base
	if idx := strings.Index(code, "^"); idx >= 0 {
		if t, ok := wesnothToCiv5[code[:idx]]; ok {
			return t
		}
	}
	return "Grassland" // safe default
}

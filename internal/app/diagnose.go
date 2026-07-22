package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

type DiagIssue struct {
	Mod      string `json:"mod"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Detail   string `json:"detail"`
}

// DiagnoseMods runs per-mod + merged validation mirroring Unciv's RulesetValidator.
func (a *App) DiagnoseMods() ([]DiagIssue, error) {
	mods, err := a.ScanMods()
	if err != nil {
		return nil, err
	}
	var issues []DiagIssue
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return nil, fmt.Errorf("未设置 Unciv 路径")
	}

	if len(mods) == 0 {
		return []DiagIssue{{
			Severity: "info",
			Message:  "未找到任何模组，请先安装模组或检查 Unciv 路径是否正确",
		}}, nil
	}

	// ── Per-mod Checks ──
	for _, mod := range mods {
		dir := filepath.Join(uncivPath, "mods", mod.Folder, "jsons")
		entries, err := os.ReadDir(dir)
		if err != nil {
			issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error", Message: "缺少 jsons/ 目录"})
			continue
		}

		localTypes := map[string]bool{}
		localNames := map[string]bool{}
		var entities []Entity
		nameCount := map[string]int{}

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "ModOptions.json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error", Message: "无法读取 " + e.Name()})
				continue
			}
			content := preprocessUncivJSON(string(data))

			// Deprecated unique patterns
			for _, rule := range deprecatedRules {
				if strings.Contains(content, rule.Pattern) {
					msg := fmt.Sprintf("%s — 废弃语法", e.Name())
					detail := rule.Pattern
					if rule.Description != "" {
						msg = fmt.Sprintf("%s — %s", e.Name(), rule.Description)
					}
					if rule.Replacement != "" {
						detail = "替代: " + rule.Replacement
					}
					issues = append(issues, DiagIssue{
						Mod: mod.Folder, Severity: rule.Severity,
						Message: msg,
						Detail:  detail,
					})
				}
			}

			ft := strings.TrimSuffix(e.Name(), ".json")
			parseEntities(content, ft, mod.Folder, e.Name(), &entities, &localNames, &localTypes, &nameCount)
		}

		if len(entities) == 0 {
			issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "warning", Message: "未找到有效实体"})
			continue
		}

		// ── Unciv checkUnit: self-upgrade ──
		for _, ent := range entities {
			if ent.FileType == "Units.json" && ent.UpgradesTo != "" && ent.UpgradesTo == ent.Name {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error", Message: fmt.Sprintf("%s upgradesTo 指向自身！", ent.Name)})
			}
		}

		// ── Unciv checkUnit: replaces without uniqueTo ──
		for _, ent := range entities {
			if ent.FileType == "Units.json" && ent.Replaces != "" && ent.UniqueTo == "" {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "warning",
					Message: fmt.Sprintf("%s replaces %s 但没有 uniqueTo，该单位将替换对所有文明生效", ent.Name, ent.Replaces)})
			}
		}

		// ── Type references: own + vanilla ──
		for _, ent := range entities {
			for _, ref := range []struct{ field, val string }{
				{"unitType", ent.UnitType},
				{"requiredTech", ent.RequiredTech},
				{"requiredResource", ent.RequiredResource},
			} {
				if ref.val == "" || localTypes[ref.val] || IsVanillaType(ref.val) {
					continue
				}
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error",
					Message: fmt.Sprintf("%s 引用不存在的 %s=%q", ent.Name, ref.field, ref.val)})
			}
		}

		// ── Duplicate names ──
		for key, n := range nameCount {
			if n > 1 {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "warning", Message: fmt.Sprintf("同名实体 %q 定义了 %d 次", key, n)})
			}
		}
	}

	// ── Merged Checks (across all mods, like RulesetValidator) ──

	// 1. Victory type errors
	vtIssues := validateVictoryTypes(mods, uncivPath)
	issues = append(issues, vtIssues...)

	// 2. Religion errors
	rIssues := validateFavoredReligion(mods, uncivPath)
	issues = append(issues, rIssues...)

	// 3. Tech column errors
	tcIssues := validateTechColumns(mods, uncivPath)
	issues = append(issues, tcIssues...)

	if issues == nil {
		issues = []DiagIssue{}
	}
	return issues, nil
}

// ── Checks matching Unciv's RulesetValidator ──

func validateVictoryTypes(mods []ModInfo, uncivPath string) []DiagIssue {
	victoryTypes := map[string]bool{}
	for _, mod := range mods {
		data, _ := os.ReadFile(filepath.Join(uncivPath, "mods", mod.Folder, "jsons", "VictoryTypes.json"))
		gjson.ParseBytes(data).ForEach(func(_, v gjson.Result) bool {
			victoryTypes[v.Get("name").String()] = true
			return true
		})
	}
	// Add standard vanilla victory types
	for _, vt := range []string{"Cultural", "Domination", "Diplomatic", "Scientific", "Time", "Religious", "Conquest", "Technological", "Score", "Neutral"} {
		victoryTypes[vt] = true
	}

	var issues []DiagIssue
	for _, mod := range mods {
		data, err := os.ReadFile(filepath.Join(uncivPath, "mods", mod.Folder, "jsons", "Nations.json"))
		if err != nil {
			continue
		}
		gjson.ParseBytes(data).ForEach(func(_, v gjson.Result) bool {
			pvt := v.Get("preferredVictoryType").String()
			if pvt != "" && !victoryTypes[pvt] {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error",
					Message: fmt.Sprintf("国家 %q 的 preferredVictoryType=%q 不存在", v.Get("name").String(), pvt)})
			}
			return true
		})
	}
	return issues
}

func validateFavoredReligion(mods []ModInfo, uncivPath string) []DiagIssue {
	religions := map[string]bool{}
	for _, mod := range mods {
		data, _ := os.ReadFile(filepath.Join(uncivPath, "mods", mod.Folder, "jsons", "Religions.json"))
		gjson.ParseBytes(data).ForEach(func(_, v gjson.Result) bool {
			religions[v.Get("name").String()] = true
			return true
		})
	}
	// Standard Civ5 religions
	for _, r := range []string{"Buddhism", "Christianity", "Confucianism", "Hinduism", "Islam", "Judaism", "Shinto", "Sikhism", "Taoism", "Zoroastrianism"} {
		religions[r] = true
	}

	var issues []DiagIssue
	for _, mod := range mods {
		data, err := os.ReadFile(filepath.Join(uncivPath, "mods", mod.Folder, "jsons", "Nations.json"))
		if err != nil {
			continue
		}
		gjson.ParseBytes(data).ForEach(func(_, v gjson.Result) bool {
			fr := v.Get("favoredReligion").String()
			if fr != "" && !religions[fr] {
				issues = append(issues, DiagIssue{Mod: mod.Folder, Severity: "error",
					Message: fmt.Sprintf("国家 %q 的 favoredReligion=%q 不存在", v.Get("name").String(), fr)})
			}
			return true
		})
	}
	return issues
}

type tpos struct{ row, col int }

func validateTechColumns(mods []ModInfo, uncivPath string) []DiagIssue {
	occupied := map[tpos]string{}
	var issues []DiagIssue

	for _, mod := range mods {
		data, err := os.ReadFile(filepath.Join(uncivPath, "mods", mod.Folder, "jsons", "Techs.json"))
		if err != nil {
			continue
		}
		content := preprocessUncivJSON(string(data))
		// Handle both flat and column-based tech tree
		walkTechsForPosition(gjson.Parse(content), &occupied, &issues, mod.Folder)
	}
	return issues
}

func walkTechsForPosition(v gjson.Result, occupied *map[tpos]string, issues *[]DiagIssue, modName string) {
	if !v.IsObject() {
		return
	}
	// Column-based: each item has columnNumber + techs[]
	if colNum := v.Get("columnNumber"); colNum.Exists() {
		col := int(colNum.Int())
		if techsArr := v.Get("techs"); techsArr.Exists() && techsArr.IsArray() {
			techsArr.ForEach(func(_, tech gjson.Result) bool {
				row := int(tech.Get("row").Int())
				p := tpos{row, col}
				name := tech.Get("name").String()
				if existing, ok := (*occupied)[p]; ok {
					*issues = append(*issues, DiagIssue{Mod: modName, Severity: "error",
						Message: fmt.Sprintf("%q 和 %q 在同一科技列位置 (第%d行, 第%d列)", name, existing, row, col)})
				} else {
					(*occupied)[p] = name
				}
				return true
			})
		}
		return
	}
	// Flat: check row/column fields directly
	row := int(v.Get("row").Int())
	col := int(v.Get("columnNumber").Int())
	if row > 0 || col > 0 { // has position
		p := tpos{row, col}
		name := v.Get("name").String()
		if existing, ok := (*occupied)[p]; ok {
			*issues = append(*issues, DiagIssue{Mod: modName, Severity: "error",
				Message: fmt.Sprintf("%q 和 %q 在同一科技列位置 (第%d行, 第%d列)", name, existing, row, col)})
		} else {
			(*occupied)[p] = name
		}
	}
}

// parseEntities handles both flat and column-based JSON arrays recursively.
func parseEntities(content, ft, modName, fileName string, entities *[]Entity, localNames, localTypes *map[string]bool, nameCount *map[string]int) {
	gjson.Parse(content).ForEach(func(_, v gjson.Result) bool {
		walkEntity(v, ft, modName, fileName, entities, localNames, localTypes, nameCount)
		return true
	})
}

func walkEntity(v gjson.Result, ft, modName, fileName string, entities *[]Entity, localNames, localTypes *map[string]bool, nameCount *map[string]int) {
	if !v.IsObject() {
		return
	}
	// Column-based tech: each item has a "techs" sub-array
	if sub := v.Get("techs"); sub.Exists() && sub.IsArray() {
		sub.ForEach(func(_, subItem gjson.Result) bool {
			walkEntity(subItem, ft, modName, fileName, entities, localNames, localTypes, nameCount)
			return true
		})
		return
	}
	n := v.Get("name").String()
	if n == "" {
		return
	}
	(*localNames)[n] = true
	if ft == "UnitTypes" || ft == "Techs" || ft == "TileResources" || ft == "Terrains" {
		(*localTypes)[n] = true
	}
	*entities = append(*entities, Entity{
		ModName: modName, FileType: fileName, Name: n,
		UnitType:         v.Get("unitType").String(),
		RequiredTech:     v.Get("requiredTech").String(),
		RequiredResource: v.Get("requiredResource").String(),
		Replaces:         v.Get("replaces").String(),
		UpgradesTo:       v.Get("upgradesTo").String(),
		UniqueTo:         v.Get("uniqueTo").String(),
	})
	(*nameCount)[fileName+":"+n]++
}

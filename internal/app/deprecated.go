package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

// deprecatedRule records a unique/pattern that Unciv deprecates or rejects.
// Source: Unciv source code — temp_UniqueType.kt @Deprecated annotations,
// temp_Countables.kt @Deprecated annotations, and changelogs.
type deprecatedRule struct {
	Since       string // version that started rejecting/warning
	Pattern     string // substring to match in uniques (case-sensitive)
	Description string
	Replacement string // "" = no replacement exists
	Severity    string // "error" for hard break, "warning" for soft deprecation
}

// deprecatedRules — update when Unciv changelogs mention removed/renamed unique syntax.
// Sourced from Unciv Kotlin source @Deprecated annotations.
var deprecatedRules = []deprecatedRule{
	// ── 4.19.10 ──
	{
		Since: "4.19.10", Severity: "warning",
		Pattern: "Food consumption by specialists ",
		Description: "\"Food consumption by specialists\" 已废弃 — specialists 改为占位符 [Specialists]",
		Replacement: "[relativeAmount]% Food consumption by [Specialists] [cityFilter]",
	},
	// ── Countables (never worked) ──
	{
		Since: "4.0+", Severity: "error",
		Pattern: "\"City-States\"",
		Description: "\"City-States\" 作为可计数项从未实际支持",
		Replacement: "Remaining [City-State] Civilizations",
	},
	// ── Planned deprecations (NOT yet enforced by Unciv, but signposted) ──
	{
		Since: "(planned)", Severity: "warning",
		Pattern: "May create improvements on water resources",
		Description: "\"May create improvements on water resources\" 计划改为通用施工动作",
		Replacement: "Can instantly construct a [improvementFilter] improvement <by consuming this unit>",
	},
	{
		Since: "(planned)", Severity: "warning",
		Pattern: "Spaceship part",
		Description: "\"Spaceship part\" 将只用于建筑物，单位端计划废弃",
		Replacement: "使用 Buildings.json 的 \"Spaceship part\"",
	},
	{
		Since: "(planned)", Severity: "warning",
		Pattern: "Enables nuclear weapon",
		Description: "\"Enables nuclear weapon\" 计划重构，写法可能变化",
		Replacement: "关注 Unciv 更新日志",
	},
	{
		Since: "(planned)", Severity: "warning",
		Pattern: "Enables construction of Spaceship parts",
		Description: "\"Enables construction of Spaceship parts\" 计划重构",
		Replacement: "关注 Unciv 更新日志",
	},
}

// CheckDeprecated scans all mod uniques against known deprecated patterns.
func (a *App) CheckDeprecated() ([]DiagIssue, error) {
	mods, err := a.ScanMods()
	if err != nil {
		return nil, err
	}
	var issues []DiagIssue

	for _, mod := range mods {
		dir := filepath.Join(a.config.UncivPath, "mods", mod.Folder, "jsons")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			fp := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(fp)
			if err != nil {
				continue
			}
			content := preprocessUncivJSON(string(data))
			// Extract unique strings using gjson
			arr := gjson.Parse(content)
			if !arr.IsArray() {
				continue
			}
			arr.ForEach(func(_, v gjson.Result) bool {
				if !v.IsObject() {
					return true
				}
				// Check uniques array
				uniques := v.Get("uniques")
				if uniques.Exists() && uniques.IsArray() {
					uniques.ForEach(func(_, u gjson.Result) bool {
						checkDeprecatedIn(mod.Folder, e.Name(), u.String(), &issues)
						return true
					})
				}
				return true
			})
			// Also scan raw text for patterns not inside uniques arrays
			for _, rule := range deprecatedRules {
				if strings.Contains(content, rule.Pattern) {
					// Dedup — only add if not already in issues for this mod+file+pattern
					found := false
					for _, is := range issues {
						if is.Mod == mod.Folder && strings.Contains(is.Message, rule.Pattern[:20]) {
							found = true
							break
						}
					}
					if !found {
						issues = append(issues, DiagIssue{
							Mod: mod.Folder, Severity: rule.Severity,
							Message: fmt.Sprintf("%s — %s", e.Name(), rule.Description),
							Detail:  fmt.Sprintf("替代: %s", rule.Replacement),
						})
					}
				}
			}
		}
	}

	if issues == nil {
		issues = []DiagIssue{}
	}
	return issues, nil
}

func checkDeprecatedIn(mod, file, unique string, issues *[]DiagIssue) {
	for _, rule := range deprecatedRules {
		if strings.Contains(unique, rule.Pattern) {
			*issues = append(*issues, DiagIssue{
				Mod: mod, Severity: rule.Severity,
				Message: fmt.Sprintf("废弃 unique: %q", trimTo(unique, 60)),
				Detail:  fmt.Sprintf("(%s) %s → %s", rule.Since, rule.Description, rule.Replacement),
			})
		}
	}
}

func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

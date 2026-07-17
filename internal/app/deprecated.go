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

// deprecatedRules 已移至 deprecated_gen.go（自动生成，413条）

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
						msg := fmt.Sprintf("%s — 废弃语法", e.Name())
						detail := rule.Pattern
						if rule.Description != "" {
							msg = fmt.Sprintf("%s — %s", e.Name(), rule.Description)
						}
						if rule.Replacement != "" {
							detail = fmt.Sprintf("替代: %s", rule.Replacement)
						}
						issues = append(issues, DiagIssue{
							Mod: mod.Folder, Severity: rule.Severity,
							Message: msg,
							Detail:  detail,
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
			detail := fmt.Sprintf("(%s)", rule.Since)
			if rule.Description != "" {
				detail += " " + rule.Description
			}
			if rule.Replacement != "" {
				detail += " → " + rule.Replacement
			}
			*issues = append(*issues, DiagIssue{
				Mod: mod, Severity: rule.Severity,
				Message: fmt.Sprintf("废弃 unique: %q", trimTo(unique, 60)),
				Detail:  detail,
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

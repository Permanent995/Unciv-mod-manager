package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

// AnalyzeConflicts performs multi-layered conflict analysis.
//
// Architecture:
//   - Rulesets never compared to each other (mutually exclusive)
//   - Each extension checked against EACH ruleset for type existence
//   - Extensions checked against each other for same-name / replaces
//   - Result: per-extension, which rulesets it's compatible with
func (a *App) AnalyzeConflicts() ([]ConflictReport, error) {
	mods, err := a.ScanMods()
	if err != nil {
		return nil, err
	}

	var rulesets []ModInfo
	var extensions []ModInfo
	for _, mod := range mods {
		if mod.Category == "graphics" || mod.Category == "audio" || mod.Category == "map" {
			continue
		}
		if mod.IsBaseRuleset || mod.Category == "ruleset" {
			rulesets = append(rulesets, mod)
		} else {
			extensions = append(extensions, mod)
		}
	}

	type entityKey struct{ FileType, Name string }

	entityCache := make(map[string][]Entity)
	globalIndex := make(map[entityKey][]Entity) // extensions only

	load := func(mod ModInfo) []Entity {
		if c, ok := entityCache[mod.Folder]; ok {
			return c
		}
		e := a.parseModEntities(mod)
		entityCache[mod.Folder] = e
		return e
	}

	// Extensions → cache + globalIndex
	for _, mod := range extensions {
		for _, e := range load(mod) {
			globalIndex[entityKey{e.FileType, e.Name}] = append(globalIndex[entityKey{e.FileType, e.Name}], e)
		}
	}

	// Rulesets → cache only
	for _, mod := range rulesets {
		load(mod)
	}

	var reports []ConflictReport

	// ══ Phase A: Extension vs each Ruleset (type existence, 1:N) ══
	// For each extension, check type refs exist in each ruleset's type set.
	// Only report: extension → rulesets where MISSING types cause errors.
	buildTypeSet := func(mods []ModInfo) map[string]bool {
		s := map[string]bool{}
		for _, m := range mods {
			for _, e := range load(m) {
				ft := strings.TrimSuffix(e.FileType, ".json")
				if ft == "UnitTypes" || ft == "Techs" || ft == "TileResources" {
					s[e.Name] = true
				}
			}
		}
		return s
	}

	for _, ext := range extensions {
		// Collect all types this extension references
		type ref struct{ field, value string }
		refs := map[ref]bool{}
		for _, e := range load(ext) {
			for _, r := range []ref{
				{"unitType", e.UnitType},
				{"requiredTech", e.RequiredTech},
				{"requiredResource", e.RequiredResource},
			} {
				if r.value != "" {
					refs[r] = true
				}
			}
		}

		for _, ruleset := range rulesets {
			rsTypes := buildTypeSet([]ModInfo{ruleset})
			var missing []string
			for r := range refs {
				if rsTypes[r.value] {
					continue
				}
				// Vanilla engine types/techs/resources not in any JSON → skip false positive
				if IsVanillaType(r.value) {
					continue
				}
				missing = append(missing, fmt.Sprintf("%s=%q", r.field, r.value))
			}
			if len(missing) > 0 {
				reports = append(reports, ConflictReport{
					Level:    "risk",
					Category: "compat",
					ModA:     ext.Folder,
					ModB:     ruleset.Folder,
					Message:  fmt.Sprintf("%s 在规则集 %q 下缺失 %d 个类型引用", ext.Folder, ruleset.Folder, len(missing)),
					Detail:   strings.Join(missing, "、"),
				})
			}
		}
	}

	// ══ Phase B: Extension vs Extension — same-name ══════════════
	for key, entities := range globalIndex {
		if len(entities) < 2 {
			continue
		}
		cat := classifyFile(key.FileType)
		for i := 0; i < len(entities); i++ {
			for j := i + 1; j < len(entities); j++ {
				a, b := entities[i], entities[j]
				if a.ModName == b.ModName {
					continue
				}
				lv, msg, detail := evaluateConflict(a, b)
				reports = append(reports, ConflictReport{
					Level: lv, Category: cat,
					ModA: a.ModName, ModB: b.ModName,
					EntityID: key.FileType + ":" + key.Name,
					Message: msg, Detail: detail,
				})
			}
		}
	}

	// ══ Phase C: Cross-mod replaces ═══════════════════════════════
	repl := map[string][]Entity{}
	for _, es := range globalIndex {
		for _, e := range es {
			if e.Replaces != "" {
				repl[e.Replaces] = append(repl[e.Replaces], e)
			}
		}
	}
	for tgt, rs := range repl {
		if len(rs) < 2 {
			continue
		}
		ms := map[string]bool{}
		for _, r := range rs {
			ms[r.ModName] = true
		}
		if len(ms) < 2 {
			continue
		}
		cat := classifyFile(rs[0].FileType)
		for i := 0; i < len(rs); i++ {
			for j := i + 1; j < len(rs); j++ {
				rA, rB := rs[i], rs[j]
				if rA.ModName == rB.ModName {
					continue
				}
				reports = append(reports, ConflictReport{
					Level: "override", Category: cat,
					ModA: rA.ModName, ModB: rB.ModName,
					EntityID: rA.FileType + ":" + rA.Replaces,
					Message:  fmt.Sprintf("两个扩展都替换了 %q：%s→%s，%s→%s", tgt, rA.ModName, rA.Name, rB.ModName, rB.Name),
					Detail:   "加载顺序决定最终哪个替换生效",
				})
			}
		}
	}

	// ══ Phase D: Cross-mod entity references ═════════════════════
	// For each extension, check replaces/upgradesTo targets exist
	// in any loaded mod (rulesets + extensions + vanilla).
	{
		allEntityNames := map[string]bool{}
		for _, mod := range append(rulesets, extensions...) {
			for _, e := range load(mod) {
				allEntityNames[e.Name] = true
			}
		}
		for _, ext := range extensions {
			for _, e := range load(ext) {
				for _, ref := range []struct{ field, val string }{
					{"replaces", e.Replaces},
					{"upgradesTo", e.UpgradesTo},
				} {
					if ref.val == "" || allEntityNames[ref.val] || IsVanillaType(ref.val) {
						continue
					}
					reports = append(reports, ConflictReport{
						Level: "risk", Category: "compat",
						ModA: ext.Folder,
						Message: fmt.Sprintf("%s %s=%q 在所有已安装模组中均未找到", e.Name, ref.field, ref.val),
						Detail: "需确认依赖模组是否已安装",
					})
				}
			}
		}
	}

	// ══ Phase E: Incompatible declarations ═══════════════════════════
	all := append(rulesets, extensions...)
	for _, mod := range all {
		for _, tgt := range a.findIncompatibles(mod) {
			for _, other := range all {
				if mod.Folder == other.Folder {
					continue
				}
				if strings.EqualFold(other.Name, tgt) || strings.Contains(other.Folder, tgt) {
					reports = append(reports, ConflictReport{
						Level: "incompatible", Category: "other",
						ModA: mod.Folder, ModB: other.Folder,
						Message: fmt.Sprintf("%q 声明与 %q 不兼容", mod.Folder, other.Folder),
						Detail:  "作者明确声明不兼容",
					})
				}
			}
		}
	}

	return reports, nil
}

// ── Helpers ──────────────────────────────────────────────────────────

func (a *App) parseModEntities(mod ModInfo) []Entity {
	var out []Entity
	dir := filepath.Join(a.config.UncivPath, "mods", mod.Folder, "jsons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "ModOptions.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		content := string(data)
		// gjson chokes on files starting with /// comments between array elements.
		// Preprocess: strip lines that are only comments outside of strings.
		content = preprocessUncivJSON(content)

		arr := gjson.Parse(content)
		if !arr.IsArray() {
			continue
		}
		arr.ForEach(func(_, v gjson.Result) bool {
			if !v.IsObject() {
				return true
			}
			n := v.Get("name").String()
			if n == "" {
				return true
			}
			out = append(out, Entity{
				ModName:          mod.Folder,
				FileType:         e.Name(),
				Name:             n,
				UnitType:         v.Get("unitType").String(),
				RequiredTech:     v.Get("requiredTech").String(),
				RequiredResource: v.Get("requiredResource").String(),
				Replaces:         v.Get("replaces").String(),
				UniqueTo:         v.Get("uniqueTo").String(),
				Strength:         int(v.Get("strength").Int()),
				Cost:             int(v.Get("cost").Int()),
				Maintenance:      int(v.Get("maintenance").Int()),
			})
			if ma := v.Get("_mergeAction.action").String(); ma != "" {
				out[len(out)-1].MergeAction = ma
			}
			return true
		})
	}
	return out
}

// preprocessUncivJSON removes `///` and `//` comment-only lines that gjson can't handle.
func preprocessUncivJSON(s string) string {
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "///") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

func evaluateConflict(a, b Entity) (level, msg, detail string) {
	if a.MergeAction == "TRY_INJECT" && b.MergeAction == "TRY_INJECT" {
		var diffs []string
		if a.Strength != 0 && b.Strength != 0 && a.Strength != b.Strength {
			diffs = append(diffs, fmt.Sprintf("strength: %d→%d", a.Strength, b.Strength))
		}
		if a.Cost != 0 && b.Cost != 0 && a.Cost != b.Cost {
			diffs = append(diffs, fmt.Sprintf("cost: %d→%d", a.Cost, b.Cost))
		}
		if len(diffs) > 0 {
			return "risk", fmt.Sprintf("两个扩展 TRY_INJECT %q，标量被覆盖", a.Name),
				strings.Join(diffs, "，") + "。数组字段保留追加。"
		}
		return "safe", fmt.Sprintf("两个扩展 TRY_INJECT %q 数组字段", a.Name), "游戏自动追加，无冲突"
	}
	if a.MergeAction == "CREATE_OR_REPLACE" || b.MergeAction == "CREATE_OR_REPLACE" {
		return "override", fmt.Sprintf("%q 被 CREATE_OR_REPLACE 完整覆盖", a.Name), "完全替换同实体定义"
	}
	var diffs []string
	if a.Strength != b.Strength && a.Strength != 0 && b.Strength != 0 {
		diffs = append(diffs, fmt.Sprintf("strength: %d vs %d", a.Strength, b.Strength))
	}
	if a.Cost != b.Cost && a.Cost != 0 && b.Cost != 0 {
		diffs = append(diffs, fmt.Sprintf("cost: %d vs %d", a.Cost, b.Cost))
	}
	detail = strings.Join(diffs, "，")
	if detail != "" {
		detail += "。加载顺序决定生效版本。"
	}
	return "override", fmt.Sprintf("两个扩展都定义了同名 %q，后者覆盖前者", a.Name), detail
}

func (a *App) findIncompatibles(mod ModInfo) []string {
	p := filepath.Join(a.config.UncivPath, "mods", mod.Folder, "jsons", "ModOptions.json")
	d, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var t []string
	gjson.Get(string(d), "uniques").ForEach(func(_, v gjson.Result) bool {
		s := v.String()
		if strings.Contains(s, "incompatible") {
			si, ei := strings.Index(s, "["), strings.Index(s, "]")
			if si != -1 && ei > si {
				t = append(t, s[si+1:ei])
			}
		}
		return true
	})
	return t
}

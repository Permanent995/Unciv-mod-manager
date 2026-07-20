package app

import (
	"net/url"
	"testing"
)

// ── Semver ──

func TestParseVer(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"1.0.0", []int{1, 0, 0}},
		{"v2.3.4", []int{2, 3, 4}},
		{"V5.10.15", []int{5, 10, 15}},
		{"0.1", []int{0, 1}},
		{"1.2.3.4", []int{1, 2, 3, 4}},
		{"abc", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseVer(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseVer(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseVer(%q) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.5.0", "1.4.9", 1},
		{"v1.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"0.9", "1.0", -1},
	}
	for _, tt := range tests {
		got := compareSemVer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemVer(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// ── Vanilla types ──

func TestIsVanillaType(t *testing.T) {
	known := []string{"Swordsman", "Agriculture", "Horses", "Grassland", "Shock I", "Courthouse"}
	for _, name := range known {
		if !IsVanillaType(name) {
			t.Errorf("IsVanillaType(%q) = false, want true", name)
		}
	}
	unknown := []string{"", "Scavenger", "CustomUnit99", "FakeTech"}
	for _, name := range unknown {
		if IsVanillaType(name) {
			t.Errorf("IsVanillaType(%q) = true, want false", name)
		}
	}
}

// ── Classify file ──

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Buildings.json", "建筑"},
		{"Units.json", "单位"},
		{"Techs.json", "科技"},
		{"Unknown.json", "other"},
		{"", "other"},
	}
	for _, tt := range tests {
		got := classifyFile(tt.input)
		if got != tt.want {
			t.Errorf("classifyFile(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Evaluate conflict ──

func TestEvaluateConflict(t *testing.T) {
	base := Entity{Name: "TestUnit", FileType: "Units.json"}
	withMerge := Entity{Name: "TestUnit", FileType: "Units.json", MergeAction: "TRY_INJECT"}

	t.Run("both TRY_INJECT scalar diff", func(t *testing.T) {
		a := Entity{Name: "Spearman", FileType: "Units.json", Strength: 15, MergeAction: "TRY_INJECT"}
		b := Entity{Name: "Spearman", FileType: "Units.json", Strength: 20, MergeAction: "TRY_INJECT"}
		level, _, _ := evaluateConflict(a, b)
		if level != "risk" {
			t.Errorf("TRY_INJECT scalar diff: level=%q, want risk", level)
		}
	})

	t.Run("both TRY_INJECT same", func(t *testing.T) {
		level, _, _ := evaluateConflict(withMerge, withMerge)
		if level != "safe" {
			t.Errorf("TRY_INJECT same: level=%q, want safe", level)
		}
	})

	t.Run("one CREATE_OR_REPLACE", func(t *testing.T) {
		a := Entity{Name: "Spearman", FileType: "Units.json", MergeAction: "TRY_INJECT"}
		b := Entity{Name: "Spearman", FileType: "Units.json", MergeAction: "CREATE_OR_REPLACE"}
		level, _, _ := evaluateConflict(a, b)
		if level != "override" {
			t.Errorf("CREATE_OR_REPLACE: level=%q, want override", level)
		}
	})

	t.Run("no merge action", func(t *testing.T) {
		level, _, _ := evaluateConflict(base, base)
		if level != "override" {
			t.Errorf("no merge action: level=%q, want override", level)
		}
	})
}

// ── TrimTo ──

func TestTrimTo(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := trimTo(tt.input, tt.n)
		if got != tt.want {
			t.Errorf("trimTo(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.want)
		}
	}
}

// ── Deprecated pattern ──

func TestCheckDeprecatedIn(t *testing.T) {
	issues := make([]DiagIssue, 0)
	checkDeprecatedIn("TestMod", "Buildings.json", "Doubles Gold given to enemy if city is captured", &issues)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Mod != "TestMod" {
		t.Errorf("mod = %q, want TestMod", issues[0].Mod)
	}
	if issues[0].Severity != "warning" {
		t.Errorf("severity = %q, want warning", issues[0].Severity)
	}
}

func TestDeprecatedRulesList(t *testing.T) {
	if len(deprecatedRules) == 0 {
		t.Fatal("deprecatedRules is empty")
	}
	for i, r := range deprecatedRules {
		if r.Pattern == "" {
			t.Errorf("deprecatedRules[%d] has empty pattern", i)
		}
		if r.Severity != "error" && r.Severity != "warning" {
			t.Errorf("deprecatedRules[%d] severity = %q, want error or warning", i, r.Severity)
		}
	}
}

// ── Mirrors ──

func TestDefaultMirrors(t *testing.T) {
	m := defaultMirrors()
	if len(m) == 0 {
		t.Fatal("defaultMirrors() returned empty list")
	}
	for i, url := range m {
		if url == "" {
			t.Errorf("mirror[%d] is empty", i)
		}
	}
}

// ── Format speed ──

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bps  int64
		want string
	}{
		{500, "500 B/s"},
		{1024, "1.0 KB/s"},
		{1536, "1.5 KB/s"},
		{1048576, "1.0 MB/s"},
		{2097152, "2.0 MB/s"},
		{0, "0 B/s"},
	}
	for _, tt := range tests {
		got := formatSpeed(tt.bps)
		if got != tt.want {
			t.Errorf("formatSpeed(%d) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

// ── GenTaskID ──

func TestGenTaskID(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := genTaskID()
		if id == "" {
			t.Fatal("genTaskID() returned empty string")
		}
		if seen[id] {
			t.Errorf("duplicate task ID: %s", id)
		}
		seen[id] = true
	}
}

// ── Parse owner/repo ──

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		url        string
		wantOwner  string
		wantRepo   string
		wantErr    bool
	}{
		{"https://github.com/user/repo", "user", "repo", false},
				{"https://github.com/user/repo/", "user", "repo", false},
		{"https://github.com/user/repo/archive/main.zip", "user", "repo", false},
		{"http://github.com/user/repo", "user", "repo", false},
				{"not-a-url", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		owner, repo, err := ParseOwnerRepo(tt.url)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseOwnerRepo(%q) expected error", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseOwnerRepo(%q) unexpected error: %v", tt.url, err)
			continue
		}
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Errorf("ParseOwnerRepo(%q) = (%q, %q), want (%q, %q)",
				tt.url, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}

// ── preprocessUncivJSON ──

func TestPreprocessUncivJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`[{"name":"test"}]`, `[{"name":"test"}]`},
		{``, ``},
	}
	for _, tt := range tests {
		got := preprocessUncivJSON(tt.input)
		if got != tt.want {
			t.Errorf("preprocessUncivJSON(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Save metadata parsing ──

func TestTryParseSaveMetadataEmpty(t *testing.T) {
	s := &SaveInfo{}
	s.tryParseMetadata("nonexistent.json")
	if s.CivName != "" {
		t.Errorf("expected empty civ name for missing file, got %q", s.CivName)
	}
}

// ── wesnothTerrain ──

func TestWesnothTerrain(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"Gg", "Grassland"},
		{"Dd", "Desert"},
		{"Mm", "Mountain"},
		{"Ww", "Coast"},
		{"Hh", "Hill"},
		{"", "Grassland"},
		{"XX", "Grassland"},
	}
	for _, tt := range tests {
		got := wesnothTerrain(tt.code)
		if got != tt.want {
			t.Errorf("wesnothTerrain(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// ── url.QueryEscape ──

func TestUrlQueryEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello+world"},
		{"abc", "abc"},
		{"a b:c", "a+b%3Ac"},
		{"q=go&lang=en", "q%3Dgo%26lang%3Den"},
	}
	for _, tt := range tests {
		got := url.QueryEscape(tt.input)
		if got != tt.want {
			t.Errorf("url.QueryEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── applyMirror ──

func TestApplyMirror(t *testing.T) {
	tests := []struct {
		rawURL string
		mode   string
		mirror string
		want   string
	}{
		{"https://github.com/user/repo", "direct", "", "https://github.com/user/repo"},
		{"https://github.com/user/repo", "mirror", "https://ghproxy.com", "https://ghproxy.com/github.com/user/repo"},
		{"http://github.com/user/repo", "mirror", "https://mirror.example/", "https://mirror.example/github.com/user/repo"},
		{"", "mirror", "https://ghproxy.com", "https://ghproxy.com/"},
	}
	for _, tt := range tests {
		got := applyMirror(tt.rawURL, tt.mode, tt.mirror)
		if got != tt.want {
			t.Errorf("applyMirror(%q, %q, %q) = %q, want %q", tt.rawURL, tt.mode, tt.mirror, got, tt.want)
		}
	}

	t.Run("null mode", func(t *testing.T) {
		got := applyMirror("https://github.com/user/repo", "", "")
		if got != "https://github.com/user/repo" {
			t.Errorf("null mode: got %q", got)
		}
	})
}

func TestEvaluateConflict_Extended(t *testing.T) {
	t.Run("TRY_INJECT same Strength zero one side", func(t *testing.T) {
		// Strength=0 on one side means "not set" — should not flag diff
		a := Entity{Name: "Spearman", FileType: "Units.json", Strength: 0, MergeAction: "TRY_INJECT"}
		b := Entity{Name: "Spearman", FileType: "Units.json", Strength: 15, MergeAction: "TRY_INJECT"}
		level, _, _ := evaluateConflict(a, b)
		if level != "safe" {
			t.Errorf("zero Strength on one side: level=%q, want safe", level)
		}
	})

	t.Run("TRY_INJECT Cost diff only", func(t *testing.T) {
		a := Entity{Name: "Library", FileType: "Buildings.json", Cost: 80, MergeAction: "TRY_INJECT"}
		b := Entity{Name: "Library", FileType: "Buildings.json", Cost: 120, MergeAction: "TRY_INJECT"}
		level, _, _ := evaluateConflict(a, b)
		if level != "risk" {
			t.Errorf("Cost diff: level=%q, want risk", level)
		}
	})

	t.Run("TRY_INJECT both zero Cost and Strength", func(t *testing.T) {
		a := Entity{Name: "Warrior", FileType: "Units.json", Strength: 0, Cost: 0, MergeAction: "TRY_INJECT"}
		b := Entity{Name: "Warrior", FileType: "Units.json", Strength: 0, Cost: 0, MergeAction: "TRY_INJECT"}
		level, _, _ := evaluateConflict(a, b)
		if level != "safe" {
			t.Errorf("both zero: level=%q, want safe", level)
		}
	})

	t.Run("one empty one TRY_INJECT", func(t *testing.T) {
		a := Entity{Name: "Settler", FileType: "Units.json", MergeAction: ""}
		b := Entity{Name: "Settler", FileType: "Units.json", Strength: 5, MergeAction: "TRY_INJECT"}
		level, _, _ := evaluateConflict(a, b)
		if level != "override" {
			t.Errorf("empty vs TRY_INJECT: level=%q, want override", level)
		}
	})

	t.Run("both empty same values", func(t *testing.T) {
		a := Entity{Name: "Worker", FileType: "Units.json", Strength: 10}
		b := Entity{Name: "Worker", FileType: "Units.json", Strength: 10}
		level, _, _ := evaluateConflict(a, b)
		if level != "override" {
			t.Errorf("both empty same: level=%q, want override", level)
		}
	})

	t.Run("both empty different Strength", func(t *testing.T) {
		a := Entity{Name: "Swordman", FileType: "Units.json", Strength: 15}
		b := Entity{Name: "Swordman", FileType: "Units.json", Strength: 20}
		level, msg, detail := evaluateConflict(a, b)
		if level != "override" {
			t.Errorf("both empty diff: level=%q, want override", level)
		}
		if detail == "" {
			t.Errorf("expected non-empty detail for scalar diff")
		}
		if msg == "" {
			t.Errorf("expected non-empty message")
		}
	})
}

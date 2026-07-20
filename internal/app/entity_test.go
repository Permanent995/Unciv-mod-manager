package app

import (
	"testing"

	"github.com/tidwall/gjson"
)

// ── categorizeMod ──

func TestCategorizeMod(t *testing.T) {
	a := &App{}
	tests := []struct {
		name          string
		topics        []string
		isBaseRuleset bool
		modPath       string
		want          string
	}{
		{"isBaseRuleset", nil, true, "", "ruleset"},
		{"topic rulesets", []string{"unciv-mod-rulesets"}, false, "", "ruleset"},
		{"topic graphics", []string{"unciv-mod-graphics"}, false, "", "graphics"},
		{"topic audio", []string{"unciv-mod-audio"}, false, "", "audio"},
		{"topic expansion", []string{"unciv-mod-expansions"}, false, "", "expansion"},
		{"topic fun", []string{"unciv-mod-fun"}, false, "", "fun"},
		{"topic maps", []string{"unciv-mod-maps"}, false, "", "map"},
		{"multiple topics", []string{"unciv-mod-graphics", "unciv-mod-fun"}, false, "", "graphics"},
		{"empty topics no folder", nil, false, "/nonexistent", "unclassified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.categorizeMod(tt.topics, tt.isBaseRuleset, false, false, false)
			if got != tt.want {
				t.Errorf("categorizeMod() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── walkEntity ──

func TestWalkEntity(t *testing.T) {
	t.Run("flat unit entity", func(t *testing.T) {
		v := gjson.Parse(`{"name":"Warrior","unitType":"Land","strength":15}`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 1 {
			t.Fatalf("expected 1 entity, got %d", len(entities))
		}
		if entities[0].Name != "Warrior" {
			t.Errorf("Name = %q, want Warrior", entities[0].Name)
		}
		if entities[0].UnitType != "Land" {
			t.Errorf("UnitType = %q, want Land", entities[0].UnitType)
		}
		if !localNames["Warrior"] {
			t.Errorf("Warrior should be in localNames")
		}
	})

	t.Run("tech type adds to localTypes", func(t *testing.T) {
		v := gjson.Parse(`{"name":"Agriculture","row":0,"columnNumber":0}`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Techs", "TestMod", "Techs.json", &entities, &localNames, &localTypes, &nameCount)

		if !localTypes["Agriculture"] {
			t.Errorf("Agriculture should be in localTypes")
		}
	})

	t.Run("column-based techs unwraps nested array", func(t *testing.T) {
		v := gjson.Parse(`{"columnNumber":1,"techs":[{"name":"AnimalHusbandry","row":0},{"name":"Archery","row":1}]}`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Techs", "TestMod", "Techs.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 2 {
			t.Fatalf("expected 2 entities, got %d", len(entities))
		}
		if entities[0].Name != "AnimalHusbandry" {
			t.Errorf("[0] = %q, want AnimalHusbandry", entities[0].Name)
		}
		if entities[1].Name != "Archery" {
			t.Errorf("[1] = %q, want Archery", entities[1].Name)
		}
	})

	t.Run("empty name is skipped", func(t *testing.T) {
		v := gjson.Parse(`{"name":""}`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 0 {
			t.Errorf("expected 0 entities for empty name, got %d", len(entities))
		}
	})

	t.Run("non-object value is skipped", func(t *testing.T) {
		v := gjson.Parse(`"just a string"`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 0 {
			t.Errorf("expected 0 entities for non-object, got %d", len(entities))
		}
	})

	t.Run("entity with Replaces and UpgradesTo", func(t *testing.T) {
		v := gjson.Parse(`{"name":"Hwacha","replaces":"Trebuchet","upgradesTo":"Artillery","uniqueTo":"Korea"}`)
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		walkEntity(v, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 1 {
			t.Fatalf("expected 1 entity, got %d", len(entities))
		}
		if entities[0].Replaces != "Trebuchet" {
			t.Errorf("Replaces = %q, want Trebuchet", entities[0].Replaces)
		}
		if entities[0].UpgradesTo != "Artillery" {
			t.Errorf("UpgradesTo = %q, want Artillery", entities[0].UpgradesTo)
		}
		if entities[0].UniqueTo != "Korea" {
			t.Errorf("UniqueTo = %q, want Korea", entities[0].UniqueTo)
		}
	})
}

// ── parseEntities ──

func TestParseEntities(t *testing.T) {
	t.Run("flat array returns all entities", func(t *testing.T) {
		content := `[{"name":"Warrior"},{"name":"Archer"}]`
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}

		parseEntities(content, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)

		if len(entities) != 2 {
			t.Fatalf("expected 2 entities, got %d", len(entities))
		}
		if nameCount["Units.json:Warrior"] != 1 {
			t.Errorf("Warrior count = %d, want 1", nameCount["Units.json:Warrior"])
		}
	})

	t.Run("empty array", func(t *testing.T) {
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}
		parseEntities("[]", "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)
		if len(entities) != 0 {
			t.Errorf("expected 0 entities, got %d", len(entities))
		}
	})

	t.Run("non-array JSON produces no entities", func(t *testing.T) {
		var entities []Entity
		localNames := map[string]bool{}
		localTypes := map[string]bool{}
		nameCount := map[string]int{}
		parseEntities(`{"object":true}`, "Units", "TestMod", "Units.json", &entities, &localNames, &localTypes, &nameCount)
		if len(entities) != 0 {
			t.Errorf("expected 0 entities, got %d", len(entities))
		}
	})
}

// ── mirrorURL ──

func TestMirrorURL(t *testing.T) {
	tests := []struct {
		rawURL string
		mirror string
		want   string
	}{
		{"https://github.com/user/repo", "https://ghproxy.com/", "https://ghproxy.com/github.com/user/repo"},
		{"http://example.com/file.zip", "https://mirror.example/", "https://mirror.example/example.com/file.zip"},
		{"https://github.com/user/repo", "https://mirror", "https://mirror/github.com/user/repo"},
		{"", "https://ghproxy.com/", "https://ghproxy.com/"},
		{"https://github.com/user/repo", "https://ghproxy.com", "https://ghproxy.com/github.com/user/repo"},
	}
	for _, tt := range tests {
		got := mirrorURL(tt.rawURL, tt.mirror)
		if got != tt.want {
			t.Errorf("mirrorURL(%q, %q) = %q, want %q", tt.rawURL, tt.mirror, got, tt.want)
		}
	}
}

// ── buildMirrorDownloadURL ──

func TestBuildMirrorDownloadURL(t *testing.T) {
	got := mirrorURL("https://github.com/user/repo", "https://ghproxy.com")
	want := "https://ghproxy.com/github.com/user/repo"
	if got != want {
		t.Errorf("buildMirrorDownloadURL() = %q, want %q", got, want)
	}
}

// ── extractHost ──

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://ghproxy.com/", "ghproxy.com"},
		{"https://mirror.ghproxy.com/", "mirror.ghproxy.com"},
		{"https://gh.api.99988866.xyz/", "gh.api.99988866.xyz"},
		{"", ""},
		{"not a url", ""},
	}
	for _, tt := range tests {
		got := extractHost(tt.url)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ── preprocessUncivJSON expanded ──

func TestPreprocessUncivJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"removes triple-slash comments", "line1\n/// comment\nline2", "line1\nline2"},
		{"removes double-slash comments", "line1\n// comment\nline2", "line1\nline2"},
		{"keeps inline URL slashes", `{"url":"https://example.com"}`, `{"url":"https://example.com"}`},
		{"handles empty string", "", ""},
		{"handles only comments", "// comment\n/// another", ""},
		{"keeps indented content", "  data  ", "  data  "},
		{"removes leading whitespace comments", "   // comment", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessUncivJSON(tt.input)
			if got != tt.want {
				t.Errorf("preprocessUncivJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

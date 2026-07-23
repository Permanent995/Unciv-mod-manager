package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	UMMRepo = "Permanent995/unciv-mod-manager"
)

// UMMVersion is set at build time via -ldflags.
// Falls back to "dev" when running from source (wails dev).
var UMMVersion = "dev"

// ModInfo represents a single mod installed under Unciv's mods/ directory.
// Folder is the subdirectory name (e.g. "Go-Astray").
type ModInfo struct {
	Name          string   `json:"name"`
	// Folder is the subdirectory name under mods/ (e.g. "Go-Astray").
	Folder        string   `json:"folder"`
	// Author is extracted from ModOptions.json; omitempty means the JSON
	// field is omitted when empty so the frontend receives a cleaner object.
	Author        string   `json:"author,omitempty"`
	IsBaseRuleset bool     `json:"isBaseRuleset"`
	Topics        []string `json:"topics,omitempty"`
	ModURL        string   `json:"modUrl,omitempty"`
	LastUpdated   string   `json:"lastUpdated,omitempty"`
	ModSize       int      `json:"modSize"`
	Category      string   `json:"category"`
	IsIncomplete  bool     `json:"isIncomplete"`
	HasReadme     bool     `json:"hasReadme"`
	HasPreview    bool     `json:"hasPreview"`
}

// Entity is a generic parsed JSON entity from any Unciv ruleset file.
// It is NOT specific to units — it covers buildings, techs, resources,
// and all other entity types. FileType distinguishes them.
type Entity struct {
	ModName          string `json:"modName"`
	FileType         string `json:"fileType"`
	Name             string `json:"name"`
	UnitType         string `json:"unitType,omitempty"`
	RequiredTech     string `json:"requiredTech,omitempty"`
	RequiredResource string `json:"requiredResource,omitempty"`
	Replaces         string `json:"replaces,omitempty"`
	UpgradesTo       string `json:"upgradesTo,omitempty"`
	UniqueTo         string `json:"uniqueTo,omitempty"`
	Strength         int    `json:"strength,omitempty"`
	Cost             int    `json:"cost,omitempty"`
	Maintenance      int    `json:"maintenance,omitempty"`
	MergeAction      string `json:"mergeAction,omitempty"`
}

var fileCategory = map[string]string{
	"Buildings.json": "建筑", "Units.json": "单位",
	// Both UnitPromotions.json and Promotions.json map to "单位晋升".
	// Unciv mods use either name; fileCategory is just a label lookup,
	// so duplicates are harmless — they don't affect conflict detection.
	"UnitTypes.json": "单位类型", "UnitPromotions.json": "单位晋升",
	"Promotions.json": "单位晋升", "Techs.json": "科技",
	"TileResources.json": "地块资源", "Terrains.json": "地形",
	"TileImprovements.json": "地块改良", "Improvements.json": "地块改良",
	"Nations.json": "国家", "Beliefs.json": "信仰",
	"Religions.json": "宗教", "Policies.json": "政策",
	"Events.json": "事件", "Quests.json": "任务",
	"Ruins.json": "遗迹", "Difficulties.json": "难度",
	"Eras.json": "时代", "Speeds.json": "速度",
	"Specialists.json": "专家", "CityStateTypes.json": "城邦",
	"GlobalUniques.json": "全局规则", "Tutorials.json": "教程",
	"VictoryTypes.json": "胜利条件",
}

func classifyFile(ft string) string {
	if c, ok := fileCategory[ft]; ok {
		return c
	}
	return "other"
}

// ConflictReport describes a single conflict between two mods.
// ModA and ModB are the two conflicting parties. If three mods conflict,
// multiple reports are generated (A-B, A-C, B-C) — all mods participate.
type ConflictReport struct {
	Level    string `json:"level"`
	Category string `json:"category"`
	ModA     string `json:"modA"`
	ModB     string `json:"modB"`
	EntityID string `json:"entityID"`		
	Message  string `json:"message"`
	Detail   string `json:"detail"`
}

type AppConfig struct {
	UncivPath             string   `json:"uncivPath"`
	SavedPaths            []string `json:"savedPaths"`
	LastActivePath        string   `json:"lastActivePath"`
	// ZoomLevel is the UI zoom percentage. 100 = normal, range [80, 150].
	ZoomLevel             int      `json:"zoomLevel"`
	SidebarPos            string   `json:"sidebarPos"`		
	SidebarWidth          int      `json:"sidebarWidth"`
	HiddenNav             []string `json:"hiddenNav"`
	Theme                 string   `json:"theme"`
	TranslateProvider     string   `json:"translateProvider"`
	TranslateCustomURL    string   `json:"translateCustomUrl"`
	TranslateCustomKey    string   `json:"translateCustomKey"`
	TranslateCustomModel  string   `json:"translateCustomModel"`
	GitHubToken           string   `json:"githubToken"`			
	MPServer              string   `json:"mpServer"`
	MPUID                 string   `json:"mpUid"`
	MPPassword            string   `json:"mpPassword"`
		CustomMirrors         []string `json:"customMirrors"`
	MaxSaves              int      `json:"maxSaves"`
	ThemeVariant          string   `json:"themeVariant"`
		MirrorMode            string   `json:"mirrorMode"`
		SelectedMirror        string   `json:"selectedMirror"`
}

type App struct {
	ctx        context.Context
	config     AppConfig
	configDir  string
	configPath string

	dlTasks     map[string]*dlTask
	dlMu        sync.Mutex
	dlDir       string
	searchCache map[string]searchCacheEntry
	searchMu    sync.Mutex
	lastAPICall time.Time
}

func NewApp() *App { return &App{} }

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initConfig()
	a.initLogger()
	a.CleanupModBackupMeta()
}

func (a *App) initLogger() {
	logDir := filepath.Join(a.configDir, "logs")
	InitLogger(LogConfig{
		LogDir:     logDir,
		MaxSize:    10 * 1024 * 1024, // 10 MB
		MaxBackups: 3,
	})
}

func (a *App) initConfig() {
	d, _ := os.UserConfigDir()
	if d == "" {
		return
	}
	a.configDir = filepath.Join(d, "UncivModManager")
	a.configPath = filepath.Join(a.configDir, "config.json")
	if data, err := os.ReadFile(a.configPath); err == nil {
		json.Unmarshal(data, &a.config)
	}
	if a.config.ZoomLevel == 0 {
		a.config.ZoomLevel = 100
	}
	if a.config.SidebarWidth == 0 {
		a.config.SidebarWidth = 220
	}
	if a.config.Theme == "" {
		a.config.Theme = "light"
	}
	if a.config.ThemeVariant == "" {
		a.config.ThemeVariant = "pure"
	}
	if a.config.TranslateProvider == "" {
		a.config.TranslateProvider = "microsoft"
	}
	if a.config.MirrorMode == "" {
		a.config.MirrorMode = "auto"
	}
	if a.config.CustomMirrors == nil {
		a.config.CustomMirrors = []string{}
	}
	if a.config.MaxSaves == 0 {
		a.config.MaxSaves = 100
	}
}

func (a *App) saveConfig() error {
	if a.configDir == "" {
		return nil
	}
	os.MkdirAll(a.configDir, 0755)
	data, _ := json.MarshalIndent(a.config, "", "  ")
	return os.WriteFile(a.configPath, data, 0644)
}

func (a *App) GetAppConfig() AppConfig { return a.config }

func (a *App) SaveAppConfig(cfg AppConfig) error {
	a.config = cfg
	return a.saveConfig()
}

func (a *App) SetUncivPath(path string) error {
	a.config.UncivPath = path
	a.config.LastActivePath = path
	for _, p := range a.config.SavedPaths {
		if p == path {
			return a.saveConfig()
		}
	}
	a.config.SavedPaths = append(a.config.SavedPaths, path)
	return a.saveConfig()
}

func (a *App) GetUMMVersion() string { return UMMVersion }

// ExportLogFile opens a Save dialog and copies the newest log file to the chosen location.
func (a *App) ExportLogFile() (string, error) {
	logDir := filepath.Join(a.configDir, "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return "", fmt.Errorf("没有日志文件")
	}
	// Find newest .log file
	var newest string
	var newestMod time.Time
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			fi, _ := e.Info()
			if fi == nil {
				continue
			}
			if fi.ModTime().After(newestMod) {
				newest = filepath.Join(logDir, e.Name())
				newestMod = fi.ModTime()
			}
		}
	}
	if newest == "" {
		return "", fmt.Errorf("没有日志文件")
	}

	dest, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出日志文件",
		DefaultFilename: "umm-log-" + time.Now().Format("2006-01-02") + ".txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "文本文件 (*.txt)", Pattern: "*.txt"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("保存对话框出错: %w", err)
	}
	if dest == "" {
		return "", nil // user cancelled
	}

	data, err := os.ReadFile(newest)
	if err != nil {
		return "", fmt.Errorf("读取日志文件失败: %w", err)
	}
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", fmt.Errorf("写入日志文件失败: %w", err)
	}
	return dest, nil
}

func (a *App) SetZoomLevel(level int) int {
	if level < 80 {
		level = 80
	}
	if level > 150 {
		level = 150
	}
	a.config.ZoomLevel = level
	a.saveConfig()
	return level
}

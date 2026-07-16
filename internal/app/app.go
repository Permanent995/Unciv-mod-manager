package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	UMMVersion = "1.3.0"
	UMMRepo    = "Permanent995/unciv-mod-manager" // TODO: 替换为实际仓库
)
type ModInfo struct {
	Name          string   `json:"name"`
	Folder        string   `json:"folder"`
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
	a.CleanupModBackupMeta()
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
	if a.config.TranslateProvider == "" {
		a.config.TranslateProvider = "microsoft"
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

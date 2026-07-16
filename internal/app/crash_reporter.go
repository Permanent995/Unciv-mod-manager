package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CrashInfo holds parsed crash report data.
type CrashInfo struct {
	Found       bool   `json:"found"`
	FilePath    string `json:"filePath"`
	LastModTime string `json:"lastModTime"`
	Raw         string `json:"raw"`
	Diagnosis   string `json:"diagnosis"`
	Suggestion  string `json:"suggestion"`
	HasMatch    bool   `json:"hasMatch"`
}

// crashPattern maps a regex to a human-readable diagnosis + suggestion.
type crashPattern struct {
	re         *regexp.Regexp
	diagnosis  string
	suggestion string
}

var crashPatterns = []crashPattern{
	{
		re:         regexp.MustCompile(`NullPointerException[\s\S]*?CityState`),
		diagnosis:  "很可能与模组有关 — 城邦功能异常",
		suggestion: "模组可能修改了城邦数据导致空指针。建议逐个禁用涉及城邦功能的扩展模组排查。",
	},
	{
		re:         regexp.MustCompile(`NullPointerException[\s\S]*?(Tile|Map|Terrain)`),
		diagnosis:  "很可能与模组有关 — 地图/地形数据异常",
		suggestion: "地形或资源模组冲突可能导致地图格子数据异常。检查最近启用的地形/资源类模组。",
	},
	{
		re:         regexp.MustCompile(`NullPointerException`),
		diagnosis:  "空指针异常，可能与模组数据不完整有关",
		suggestion: "某个模组的 JSON 文件可能缺少必要字段，或引用了不存在的实体。使用冲突检测工具排查。",
	},
	{
		re:         regexp.MustCompile(`OutOfMemoryError`),
		diagnosis:  "内存不足",
		suggestion: "Unciv 内存耗尽。尝试调低画质、使用较小地图，或关闭占用内存的应用后重试。",
	},
	{
		re:         regexp.MustCompile(`FileNotFoundException[\s\S]*?jsons?[/\\]`),
		diagnosis:  "模组文件缺失",
		suggestion: "某个模组的 JSON 文件不完整或被删除。检查最近安装的模组是否包含完整的 jsons/ 目录。",
	},
	{
		re:         regexp.MustCompile(`ClassCastException`),
		diagnosis:  "模组与 Unciv 版本不兼容",
		suggestion: "该模组可能使用了旧版 Unciv 的数据结构。检查模组是否有更新版本，或联系作者适配。",
	},
	{
		re:         regexp.MustCompile(`Json(Parse|Syntax)Exception`),
		diagnosis:  "模组 JSON 格式错误",
		suggestion: "某个模组的 JSON 文件格式有误（如多余逗号、缺少引号）。可使用 JSON 校验工具检查。",
	},
	{
		re:         regexp.MustCompile(`NoSuchMethodError|NoClassDefFoundError`),
		diagnosis:  "Unciv 版本与模组不兼容",
		suggestion: "模组依赖的 Unciv API 已变更。检查 Unciv 和模组的版本是否匹配，或更新 Unciv。",
	},
	{
		re:         regexp.MustCompile(`ConcurrentModificationException`),
		diagnosis:  "模组数据导致的并发修改异常",
		suggestion: "模组的 uniques 或事件可能在游戏循环中触发了数据修改。联系作者检查 triggers 逻辑。",
	},
}

// ReadCrashReport reads lasterror.txt from the Unciv directory and returns
// a parsed diagnosis.
func (a *App) ReadCrashReport() CrashInfo {
	info := CrashInfo{Found: false}

	if a.config.UncivPath == "" {
		return info
	}

	path := filepath.Join(a.config.UncivPath, "lasterror.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return info
	}

	raw := string(data)
	if strings.TrimSpace(raw) == "" {
		return info
	}

	info.Found = true
	info.FilePath = path
	info.Raw = raw

	if fi, err := os.Stat(path); err == nil {
		info.LastModTime = fi.ModTime().Format("2006-01-02 15:04:05")
	}

	// Match against known patterns
	for _, p := range crashPatterns {
		if p.re.MatchString(raw) {
			info.Diagnosis = p.diagnosis
			info.Suggestion = p.suggestion
			info.HasMatch = true
			return info
		}
	}

	// No match — return raw for manual review
	info.Diagnosis = "未识别错误模式，请查看原始堆栈"
	info.Suggestion = "建议将日志提交到 Unciv GitHub Issues 或模组管理器反馈渠道。"
	return info
}

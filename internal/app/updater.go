package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
)

// ModUpdateInfo describes a mod that may have an available update.
type ModUpdateInfo struct {
	Folder     string `json:"folder"`
	Name       string `json:"name"`
	CurrentVer string `json:"currentVer"` // local lastUpdated
	LatestVer  string `json:"latestVer"`  // remote pushed_at from cache
	ModURL     string `json:"modUrl"`
	HasUpdate  bool   `json:"hasUpdate"`
}

// CheckModUpdates compares every installed mod against ModListCache.json
// (Unciv's cached GitHub index) to find out-of-date mods.
//
// This is the same approach Unciv itself uses — zero GitHub API calls,
// purely local cache comparison of pushed_at vs lastUpdated.
func (a *App) CheckModUpdates() ([]ModUpdateInfo, error) {
	cachePath := filepath.Join(a.config.UncivPath, "ModListCache.json")
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 ModListCache.json（请先运行一次 Unciv 以生成该文件）: %w", err)
	}

	arr := gjson.ParseBytes(raw)
	if !arr.IsArray() {
		return nil, fmt.Errorf("ModListCache.json 格式异常")
	}

	// Build owner/repo → pushed_at map from cache
	cacheMap := map[string]string{}
	arr.ForEach(func(_, entry gjson.Result) bool {
		r := entry.Get("repo")
		if !r.Exists() {
			return true
		}
		fullName := r.Get("full_name").String()
		pushedAt := r.Get("pushed_at").String()
		if fullName != "" && pushedAt != "" {
			cacheMap[fullName] = pushedAt
		}
		return true
	})

	// Scan installed mods
	mods, err := a.ScanMods()
	if err != nil {
		return nil, err
	}

	var results []ModUpdateInfo
	for _, mod := range mods {
		if mod.ModURL == "" {
			continue
		}
		owner, repo, err := ParseOwnerRepo(mod.ModURL)
		if err != nil {
			continue
		}
		key := owner + "/" + repo
		cachePushedAt, ok := cacheMap[key]
		if !ok {
			// Not in cache at all — can't determine update status
			continue
		}

		info := ModUpdateInfo{
			Folder:     mod.Folder,
			Name:       mod.Name,
			CurrentVer: mod.LastUpdated,
			LatestVer:  cachePushedAt,
			ModURL:     mod.ModURL,
		}

		// Compare: if both are ISO-like dates, string comparison works
		// (e.g. "2026-03-15T12:00:00Z" > "2025-11-20T08:30:00Z")
		if mod.LastUpdated == "" {
			// Local has no version field → treat as potentially outdated
			info.HasUpdate = true
		} else if cachePushedAt > mod.LastUpdated {
			info.HasUpdate = true
		}

		results = append(results, info)
	}

	if results == nil {
		results = []ModUpdateInfo{}
	}
	return results, nil
}

// DownloadModUpdate downloads the latest version of a mod from its GitHub
// default branch, reusing the existing download queue system.
func (a *App) DownloadModUpdate(folder string) (string, error) {
	mods, err := a.ScanMods()
	if err != nil {
		return "", err
	}
	var target ModInfo
	for _, m := range mods {
		if m.Folder == folder {
			target = m
			break
		}
	}
	if target.ModURL == "" {
		return "", fmt.Errorf("模组 %q 没有 GitHub 地址，无法自动更新", folder)
	}

	dlURL, err := BuildDefaultBranchURL(target.ModURL)
	if err != nil {
		return "", err
	}

	filename := target.Folder + ".zip"
	return a.StartDownload(dlURL, filename)
}

// DownloadAllUpdates downloads updates for all outdated mods sequentially.
// Returns a list of task IDs for tracking in the download queue.
func (a *App) DownloadAllUpdates(updates []ModUpdateInfo) ([]string, error) {
	var ids []string
	for _, u := range updates {
		if !u.HasUpdate {
			continue
		}
		id, err := a.DownloadModUpdate(u.Folder)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("没有可更新的模组")
	}
	return ids, nil
}


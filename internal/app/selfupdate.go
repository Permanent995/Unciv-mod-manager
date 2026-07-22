package app

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// githubAPIBase is the GitHub API base URL.  Overridden in tests.
var githubAPIBase = "https://api.github.com"

// SelfUpdateInfo describes the latest available release.
type SelfUpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	DownloadURL    string `json:"downloadUrl"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseName    string `json:"releaseName"`
	CachedAt       string `json:"cachedAt,omitempty"` // non-empty = from offline cache
}

// selfUpdateCache is persisted to configDir so offline checks still work.
type selfUpdateCache struct {
	LatestVersion string `json:"latestVersion"`
	DownloadURL   string `json:"downloadUrl"`
	ReleaseName   string `json:"releaseName"`
	CachedAt      string `json:"cachedAt"`
}

// CheckSelfUpdate checks GitHub Releases for a newer version of UMM.
// Tries direct GitHub API first; falls back to the 302-redirect trick
// through download mirrors (same approach lytvpk uses).
func (a *App) CheckSelfUpdate() (SelfUpdateInfo, error) {
	info := SelfUpdateInfo{
		CurrentVersion: UMMVersion,
		HasUpdate:      false,
	}
	// ── 1. Primary: GitHub API (rich info + asset list) ──
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, UMMRepo)
	var release struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		ZipballURL string `json:"zipball_url"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	apiErr := fetchJSON(apiURL, &release)

	if apiErr != nil {
		// ── 2. Fallback: mirror redirect trick ──
		// Mirror proxies the HTML page github.com/repo/releases/latest,
		// which 302-redirects to /releases/tag/vX.Y.Z — parse tag from the
		// final URL.  This works because mirrors proxy web pages, not APIs.
		for _, m := range a.getAllMirrors() {
			tag, err := fetchLatestTagViaMirror(UMMRepo, m)
			if err != nil {
				continue
			}
			latestVer := strings.TrimPrefix(tag, "v")
			info.LatestVersion = latestVer
			info.ReleaseName = tag
			if compareSemVer(latestVer, UMMVersion) > 0 {
				info.HasUpdate = true
				// Best-effort: guess from known naming convention.
				// Must match CI upload name: unciv-mod-manager.exe
				info.DownloadURL = fmt.Sprintf(
					"https://github.com/%s/releases/download/%s/unciv-mod-manager.exe",
					UMMRepo, tag)
			}
			a.writeSelfUpdateCache(info)
			return info, nil
		}

		// ── 3. All live sources failed — try offline cache ──
		if cached := a.readSelfUpdateCache(); cached != nil {
			cached.CurrentVersion = UMMVersion
			cached.HasUpdate = compareSemVer(
				strings.TrimPrefix(cached.LatestVersion, "v"),
				UMMVersion) > 0
			return *cached, nil
		}

		if strings.Contains(apiErr.Error(), "404") {
			info.LatestVersion = UMMVersion
			return info, nil
		}
		return info, fmt.Errorf("无法获取最新版本信息（直连和镜像均失败）")
	}

	// ── 3. API 成功：正常解析 ──
	latestVer := strings.TrimPrefix(release.TagName, "v")
	info.LatestVersion = latestVer
	info.ReleaseName = release.Name

	if compareSemVer(latestVer, UMMVersion) > 0 {
		info.HasUpdate = true
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, ".old") {
				info.DownloadURL = asset.BrowserDownloadURL
				break
			}
			if strings.HasSuffix(name, ".exe") && !strings.HasSuffix(name, ".old") {
				info.DownloadURL = asset.BrowserDownloadURL
			}
		}
		if info.DownloadURL == "" {
			info.DownloadURL = release.ZipballURL
		}
	}
	a.writeSelfUpdateCache(info)
	return info, nil
}

// ── offline cache helpers ──

func (a *App) selfUpdateCachePath() string {
	return filepath.Join(a.configDir, "selfupdate_cache.json")
}

func (a *App) writeSelfUpdateCache(info SelfUpdateInfo) {
	cache := selfUpdateCache{
		LatestVersion: info.LatestVersion,
		DownloadURL:   info.DownloadURL,
		ReleaseName:   info.ReleaseName,
		CachedAt:      time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(a.selfUpdateCachePath(), data, 0644)
}

func (a *App) readSelfUpdateCache() *SelfUpdateInfo {
	data, err := os.ReadFile(a.selfUpdateCachePath())
	if err != nil {
		return nil
	}
	var cache selfUpdateCache
	if json.Unmarshal(data, &cache) != nil {
		return nil
	}
	return &SelfUpdateInfo{
		LatestVersion: cache.LatestVersion,
		DownloadURL:   cache.DownloadURL,
		ReleaseName:   cache.ReleaseName,
		CachedAt:      cache.CachedAt,
	}
}

// DownloadSelfUpdate enqueues the UMM update download using the existing
// download queue, so it benefits from mirror acceleration.
func (a *App) DownloadSelfUpdate(downloadURL string) (string, error) {
	filename := "umm_update.zip"
	if strings.HasSuffix(strings.ToLower(downloadURL), ".exe") {
		filename = "umm_update.exe"
	}
	return a.StartDownload(downloadURL, filename)
}

// InstallSelfUpdate locates the downloaded update file (raw exe or zip) and
// replaces the running executable (Windows: rename running exe → .bak, place new exe).
//
// Works on Windows because the OS allows renaming a running executable —
// the old file handle stays valid while the filename is freed for the new file.
func (a *App) InstallSelfUpdate() (map[string]bool, error) {
	result := map[string]bool{"restartRequired": true}

	// Locate the downloaded update file
	if a.dlDir == "" {
		a.dlDir = filepath.Join(os.TempDir(), "unciv-mm-downloads")
	}
	candidates := []string{
		filepath.Join(a.dlDir, "umm_update.zip"),
		filepath.Join(a.dlDir, "umm_update.exe"),
	}
	var dlPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			dlPath = p
			break
		}
	}
	if dlPath == "" {
		return result, fmt.Errorf("未找到更新文件，请先下载更新")
	}

	// Determine the new exe path: if the download is a zip, extract from it;
	// if it's already an exe, use it directly.
	var newExePath string

	if strings.HasSuffix(strings.ToLower(dlPath), ".zip") {
		r, err := zip.OpenReader(dlPath)
		if err != nil {
			return result, fmt.Errorf("无法解压更新包: %w", err)
		}
		defer r.Close()

		var exeInZip *zip.File
		for _, f := range r.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".exe") && !strings.HasSuffix(strings.ToLower(f.Name), ".old") {
				exeInZip = f
				break
			}
		}
		if exeInZip == nil {
			return result, fmt.Errorf("更新包中未找到可执行文件")
		}

		// Extract exe to a temp file alongside the install dir
		currentExe, err := os.Executable()
		if err != nil {
			return result, fmt.Errorf("无法定位当前程序路径: %w", err)
		}
		installDir := filepath.Dir(currentExe)
		newExePath = filepath.Join(installDir, filepath.Base(currentExe)+".new")
		outFile, err := os.Create(newExePath)
		if err != nil {
			return result, fmt.Errorf("无法创建临时文件: %w", err)
		}
		rc, err := exeInZip.Open()
		if err != nil {
			outFile.Close()
			os.Remove(newExePath)
			return result, fmt.Errorf("无法读取更新包: %w", err)
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			os.Remove(newExePath)
			return result, fmt.Errorf("解压失败: %w", err)
		}
	} else {
		// Raw exe downloaded — use it directly
		newExePath = dlPath
	}

	// ── Replace logic (works on Windows: running exe CAN be renamed) ──
	currentExe, err := os.Executable()
	if err != nil {
		return result, fmt.Errorf("无法定位当前程序路径: %w", err)
	}

	// For raw exe downloads, copy to install dir first so the temp file
	// doesn't disappear on reboot (temp dir may be cleaned)
	doReplace := newExePath
	if !strings.HasSuffix(strings.ToLower(dlPath), ".zip") {
		installDir := filepath.Dir(currentExe)
		staged := filepath.Join(installDir, filepath.Base(currentExe)+".new")
		if err := copyFile(dlPath, staged); err != nil {
			return result, fmt.Errorf("无法复制更新文件: %w", err)
		}
		doReplace = staged
	}

	backupPath := currentExe + ".bak"
	os.Remove(backupPath) // clean up any previous backup

	if err := os.Rename(currentExe, backupPath); err != nil {
		os.Remove(doReplace)
		return result, fmt.Errorf("备份当前程序失败（可能无权限），请手动覆盖: %w", err)
	}
	if err := os.Rename(doReplace, currentExe); err != nil {
		// Rollback: restore backup
		os.Rename(backupPath, currentExe)
		os.Remove(doReplace)
		return result, fmt.Errorf("替换程序文件失败，已自动回滚: %w", err)
	}

	result["restartRequired"] = true
	return result, nil
}

// ── helpers ──

// fetchLatestTagViaMirror probes a mirror for the latest release tag using
// the 302-redirect trick: request <mirror>/https://github.com/<repo>/releases/latest
// and parse the final URL (which ends in /releases/tag/vX.Y.Z).
// This works because mirrors proxy web pages but not the GitHub API.
func fetchLatestTagViaMirror(repo, mirror string) (string, error) {
	// Ensure exactly one / between mirror and proxied URL
	base := strings.TrimRight(mirror, "/")
	target := fmt.Sprintf("%s/https://github.com/%s/releases/latest", base, repo)

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Get(target)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Drain body to allow HTTP connection reuse
	io.Copy(io.Discard, resp.Body)

	finalURL := resp.Request.URL.String()
	parts := strings.Split(finalURL, "/tag/")
	if len(parts) < 2 {
		return "", fmt.Errorf("无法从 URL 解析版本号: %s", finalURL)
	}
	return parts[len(parts)-1], nil
}

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "umm/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain body so connection can be reused
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	// Cap JSON response at 1 MiB to prevent OOM from a rogue server
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
}

// compareSemVer compares two semantic version strings.
// Returns 1 if a > b, -1 if a < b, 0 if equal.
func compareSemVer(a, b string) int {
	pa := parseVer(a)
	pb := parseVer(b)
	for i := 0; i < 3; i++ {
		va := 0
		vb := 0
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func parseVer(v string) []int {
	v = strings.TrimLeft(v, "vV")
	parts := strings.Split(v, ".")
	var nums []int
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	return nums
}

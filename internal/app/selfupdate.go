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

// SelfUpdateInfo describes the latest available release.
type SelfUpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	DownloadURL    string `json:"downloadUrl"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseName    string `json:"releaseName"`
}

// CheckSelfUpdate checks GitHub Releases for a newer version of UMM.
func (a *App) CheckSelfUpdate() (SelfUpdateInfo, error) {
	info := SelfUpdateInfo{
		CurrentVersion: UMMVersion,
		HasUpdate:      false,
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", UMMRepo)
	var release struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		ZipballURL string `json:"zipball_url"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	err := fetchJSON(apiURL, &release)
	if err != nil {
		// Try mirrors
		for _, m := range a.getAllMirrors() {
			err = fetchJSON(mirrorURL(apiURL, m), &release)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "404") {
				info.LatestVersion = UMMVersion
				return info, nil
			}
		}
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				info.LatestVersion = UMMVersion
				return info, nil
			}
			return info, fmt.Errorf("无法获取最新版本信息（直连和镜像均失败）")
		}
	}

	latestVer := strings.TrimPrefix(release.TagName, "v")
	info.LatestVersion = latestVer
	info.ReleaseName = release.Name

	if compareSemVer(latestVer, UMMVersion) > 0 {
		info.HasUpdate = true
		for _, asset := range release.Assets {
			if strings.Contains(strings.ToLower(asset.Name), "windows") &&
				strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
				info.DownloadURL = asset.BrowserDownloadURL
				break
			}
		}
		if info.DownloadURL == "" {
			info.DownloadURL = release.ZipballURL
		}
	}
	return info, nil
}

// DownloadSelfUpdate enqueues the UMM update download using the existing
// download queue, so it benefits from mirror acceleration.
func (a *App) DownloadSelfUpdate(downloadURL string) (string, error) {
	return a.StartDownload(downloadURL, "umm_update.zip")
}

// InstallSelfUpdate locates the downloaded update zip and replaces the
// running executable (Windows: rename running exe → .bak, place new exe).
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
	}
	var zipPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			zipPath = p
			break
		}
	}
	if zipPath == "" {
		return result, fmt.Errorf("未找到更新文件（umm_update.zip），请先下载更新")
	}

	// Open the zip and find the exe
	r, err := zip.OpenReader(zipPath)
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

	// Locate current executable
	currentExe, err := os.Executable()
	if err != nil {
		return result, fmt.Errorf("无法定位当前程序路径: %w", err)
	}
	installDir := filepath.Dir(currentExe)

	// Extract new exe to a temp file alongside the install dir
	newExePath := filepath.Join(installDir, filepath.Base(currentExe)+".new")
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

	// ── Replace logic (works on Windows: running exe CAN be renamed) ──
	backupPath := currentExe + ".bak"
	os.Remove(backupPath) // clean up any previous backup

	if err := os.Rename(currentExe, backupPath); err != nil {
		os.Remove(newExePath)
		return result, fmt.Errorf("备份当前程序失败（可能无权限），请手动解压覆盖: %w", err)
	}
	if err := os.Rename(newExePath, currentExe); err != nil {
		// Rollback: restore backup
		os.Rename(backupPath, currentExe)
		os.Remove(newExePath)
		return result, fmt.Errorf("替换程序文件失败，已自动回滚: %w", err)
	}

	result["restartRequired"] = true
	return result, nil
}

// ── helpers ──

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
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
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

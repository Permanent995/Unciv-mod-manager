package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// GHRelease represents a single GitHub release.
type GHRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	PublishedAt string `json:"published_at"`
	ZipballURL  string `json:"zipball_url"`
	Assets      []GHAsset `json:"assets"`
}

// GHAsset is a downloadable file attached to a release.
type GHAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

// ParseOwnerRepo extracts owner and repo from a GitHub URL like
// https://github.com/user/repo or https://github.com/user/repo/releases.
func ParseOwnerRepo(url string) (owner, repo string, err error) {
	url = strings.TrimRight(url, "/")
	// Strip protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("无法解析 GitHub 仓库地址，格式应为 github.com/用户/仓库")
	}
	return parts[0], parts[1], nil
}

// FetchReleases fetches the releases list for a GitHub repo via a mirror.
func (a *App) FetchReleases(githubURL string) ([]GHRelease, error) {
	owner, repo, err := ParseOwnerRepo(githubURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=20", owner, repo)

	// Try direct first, then mirror
	releases, err := a.fetchReleasesFrom(apiURL)
	if err != nil {
		// Fallback via mirror
		mirrorURL := "https://ghproxy.com/" + apiURL
		releases, err = a.fetchReleasesFrom(mirrorURL)
		if err != nil {
			return nil, fmt.Errorf("无法获取 Releases 列表（直连和镜像均失败）: %w", err)
		}
	}
	return releases, nil
}

func (a *App) fetchReleasesFrom(apiURL string) ([]GHRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unciv-mod-manager")
	if a.config.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.GitHubToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API 限流，请填入 Token 或稍后再试")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 %d", resp.StatusCode)
	}

	var releases []GHRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// BuildReleaseDownloadURL returns the best download URL for a release.
// Uses archive direct link (no auth needed) instead of API zipball.
func BuildReleaseDownloadURL(owner, repo string, release GHRelease) (string, string) {
	// Prefer the first .zip asset
	for _, a := range release.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
			return a.DownloadURL, a.Name
		}
	}
	// Fallback: direct archive URL (no API auth needed)
	dlURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", owner, repo, release.TagName)
	return dlURL, release.TagName + ".zip"
}

// BuildDefaultBranchURL builds the main-branch archive URL for a repo.
func BuildDefaultBranchURL(githubURL string) (string, error) {
	owner, repo, err := ParseOwnerRepo(githubURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.zip", owner, repo), nil
}

// OnlineMod is a lightweight mod entry for the browse view.
type OnlineMod struct {
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updatedAt"`
	Topics      []string `json:"topics"`
	HTMLURL     string `json:"htmlUrl"`
}

// ReadLocalModCache loads Unciv's ModListCache.json using gjson for version-tolerant parsing.
// Only the essential fields are extracted; extra/renamed fields in newer Unciv won't break.
func (a *App) ReadLocalModCache() ([]OnlineMod, error) {
	cachePath := filepath.Join(a.config.UncivPath, "ModListCache.json")
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	arr := gjson.ParseBytes(raw)
	if !arr.IsArray() {
		return nil, fmt.Errorf("ModListCache.json 格式异常")
	}
	var mods []OnlineMod
	arr.ForEach(func(_, entry gjson.Result) bool {
		r := entry.Get("repo")
		if !r.Exists() {
			return true
		}
		mods = append(mods, OnlineMod{
			Name:        r.Get("full_name").String(),
			Owner:       r.Get("owner.login").String(),
			Repo:        r.Get("name").String(),
			Description: entry.Get("description").String(),
			Stars:       int(r.Get("stargazers_count").Int()),
			UpdatedAt:   r.Get("pushed_at").String(),
			Topics:      stringSlice(r.Get("topics")),
			HTMLURL:     r.Get("html_url").String(),
		})
		return true
	})
	return mods, nil
}

func stringSlice(r gjson.Result) []string {
	if !r.Exists() {
		return nil
	}
	var out []string
	r.ForEach(func(_, v gjson.Result) bool { out = append(out, v.String()); return true })
	return out
}

// SearchOnlineMods browses Unciv mods from the local ModListCache.json.
// Falls back to GitHub API only if the cache is unavailable.
func (a *App) SearchOnlineMods(query string) ([]OnlineMod, error) {
	mods, err := a.ReadLocalModCache()
	if err != nil {
		// Fallback to API
		return a.searchOnlineModsAPI(query)
	}
	if query != "" {
		ql := strings.ToLower(query)
		var f []OnlineMod
		for _, m := range mods {
			if strings.Contains(strings.ToLower(m.Name), ql) || strings.Contains(strings.ToLower(m.Description), ql) {
				f = append(f, m)
			}
		}
		mods = f
	}
	return mods, nil
}

func (a *App) searchOnlineModsAPI(query string) ([]OnlineMod, error) {
	q := "topic:unciv-mod"
	if query != "" {
		q += " " + query
	}

	// Throttle: at least 5 s between any API calls
	a.searchMu.Lock()
	if time.Since(a.lastAPICall) < 5*time.Second {
		a.searchMu.Unlock()
		return nil, fmt.Errorf("请求太频繁，请 %d 秒后再试", int(5-time.Since(a.lastAPICall).Seconds())+1)
	}

	// Check cache (2 min TTL)
	if cached, ok := a.searchCache[q]; ok {
		if time.Since(cached.at) < 2*time.Minute {
			a.searchMu.Unlock()
			return cached.mods, nil
		}
	}
	a.lastAPICall = time.Now()
	a.searchMu.Unlock()

	apiURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=100",
		urlEncode(q))

	result, err := a.ghRequest(apiURL)
	if err != nil {
		// Fallback via mirror
		mirrorURL := "https://ghproxy.com/" + apiURL
		result, err = a.ghRequest(mirrorURL)
		if err != nil {
			return nil, err
		}
	}

	var mods []OnlineMod
	for _, r := range result.Items {
		mods = append(mods, OnlineMod{
			Name:        r.FullName,
			Owner:       r.Owner.Login,
			Repo:        r.Name,
			Description: r.Description,
			Stars:       r.StargazersCount,
			UpdatedAt:   r.UpdatedAt,
			Topics:      r.Topics,
			HTMLURL:     r.HTMLURL,
		})
	}
	// Store in cache (prune entries older than 10 min to prevent unbounded growth)
	a.searchMu.Lock()
	if a.searchCache == nil {
		a.searchCache = map[string]searchCacheEntry{}
	}
	a.searchCache[q] = searchCacheEntry{mods: mods, at: time.Now()}
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, v := range a.searchCache {
		if v.at.Before(cutoff) {
			delete(a.searchCache, k)
		}
	}
	a.searchMu.Unlock()
	return mods, nil
}

type ghSearchResult struct {
	Items []struct {
		FullName        string   `json:"full_name"`
		Name            string   `json:"name"`
		Description     string   `json:"description"`
		StargazersCount int      `json:"stargazers_count"`
		UpdatedAt       string   `json:"updated_at"`
		Topics          []string `json:"topics"`
		HTMLURL         string   `json:"html_url"`
		Owner           struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"items"`
}

// ghRequest is the unified GitHub API caller with optional token auth.
// It returns the raw JSON decoder for the caller to unmarshal as needed.
func (a *App) ghRequest(apiURL string) (*ghSearchResult, error) {
	return a.ghRequestInto(apiURL)
}

func (a *App) ghRequestInto(apiURL string) (*ghSearchResult, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unciv-mod-manager")
	if a.config.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.GitHubToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API 限流，请在设置中填入 Token 或稍后再试")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 %d", resp.StatusCode)
	}
	var result ghSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

type searchCacheEntry struct {
	mods []OnlineMod
	at   time.Time
}

// FetchReadme fetches the README.md from a GitHub repo and returns its content.
// For users in China, it tries mirrors first since raw.githubusercontent.com
// is often blocked there.
func (a *App) FetchReadme(owner, repo string) (string, error) {
	// Try mirrors in parallel first (faster for Chinese users)
	type attempt struct {
		url  string
		err  error
		body string
	}
	ch := make(chan attempt, 5)

	urls := []string{
		fmt.Sprintf("https://ghproxy.com/https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo),
		fmt.Sprintf("https://mirror.ghproxy.com/https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo),
		fmt.Sprintf("https://gh.api.99988866.xyz/https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/README.md", owner, repo),
	}

	client := &http.Client{Timeout: 8 * time.Second}
	for _, u := range urls {
		go func(url string) {
			resp, err := client.Get(url)
			if err != nil {
				ch <- attempt{url: url, err: err}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				ch <- attempt{url: url, err: fmt.Errorf("HTTP %d", resp.StatusCode)}
				return
			}
			data := make([]byte, 0, 65536)
			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					data = append(data, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
			ch <- attempt{url: url, body: string(data)}
		}(u)
	}

	// Wait for first successful result
	timeout := time.After(20 * time.Second)
	received := 0
	for received < len(urls) {
		select {
		case a := <-ch:
			received++
			if a.body != "" {
				return a.body, nil
			}
		case <-timeout:
			return "", fmt.Errorf("所有线路均无法获取 README，请检查网络或稍后重试")
		}
	}
	return "该仓库没有 README.md", nil
}

func urlEncode(s string) string {
	// Simple URL encoding for GitHub search query
	s = strings.ReplaceAll(s, " ", "%20")
	s = strings.ReplaceAll(s, ":", "%3A")
	return s
}

func applyMirror(rawURL, mode, mirror string) string {
	if mode == "" || mode == "direct" {
		return rawURL
	}
	clean := strings.TrimPrefix(rawURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	return strings.TrimRight(mirror, "/") + "/" + clean
}

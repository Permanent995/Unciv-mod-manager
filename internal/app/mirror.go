package app

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ── Types ─────────────────────────────────────────────────────────────

// MirrorInfo holds the health check result for a single mirror endpoint.
type MirrorInfo struct {
	URL         string `json:"url"`
	Label       string `json:"label"`
	Latency     int64  `json:"latency"`      // ms; -1 = untested, 0 = timed out / failed
	Alive       bool   `json:"alive"`
	IsCustom    bool   `json:"isCustom"`
	LastChecked string `json:"lastChecked"` // ISO 8601
}

// probeTimeout is the HTTP timeout used for mirror health checks.
const probeTimeout = 6 * time.Second

// ── Default mirror list ─────────────────────────────────────────────

// defaultMirrors is the built-in mirror list.  A var so tests can swap it.
var defaultMirrors = func() []string {
	return []string{
		// ghproxy-style (prefix proxy)
		"https://ghfast.top/",           // ✅ 确认可用
		"https://ghp.ci/",               // Cloudflare 加速，推荐
		"https://ghproxy.net/",          // ghproxy 同类
		"https://moeyy.cn/gh-proxy/",    // 功能全面
		"https://github.akams.cn/",      // 支持 Release、Raw、Clone
		// clone-style (full GitHub mirror)
		"https://kkgithub.com/",         // GitHub 克隆站
		"https://bgithub.xyz/",          // 响应快
		"https://hub.fastgit.org/",      // 经典镜像
		"https://gitclone.com/",         // 附带 git clone 加速
	}
}

// ── Health check cache ───────────────────────────────────────────────

var (
	mirrorHealthCache []MirrorInfo
	mirrorHealthAt    time.Time
	mirrorHealthMu    sync.Mutex
)

const mirrorHealthTTL = 30 * time.Second

// ── App methods ──────────────────────────────────────────────────────

// getAllMirrors returns default mirrors merged with user custom mirrors (deduplicated).
func (a *App) getAllMirrors() []string {
	defs := defaultMirrors()
	seen := make(map[string]bool, len(defs)+len(a.config.CustomMirrors))
	out := make([]string, 0, len(defs)+len(a.config.CustomMirrors))

	for _, m := range defs {
		normalized := strings.TrimRight(m, "/")
		if !seen[normalized] {
			seen[normalized] = true
			out = append(out, m)
		}
	}
	for _, m := range a.config.CustomMirrors {
		normalized := strings.TrimRight(m, "/")
		if !seen[normalized] {
			seen[normalized] = true
			out = append(out, m)
		}
	}
	return out
}

// isMirrorDomain checks whether the given domain (or URL fragment) belongs
// to any known mirror.  Used by ValidateDownloadURL.
func (a *App) isMirrorDomain(domain string) bool {
	for _, m := range a.getAllMirrors() {
		u, err := url.Parse(m)
		if err != nil {
			continue
		}
		if strings.Contains(domain, u.Host) {
			return true
		}
	}
	return false
}

// GetMirrorHealth concurrently probes every known mirror and returns
// structured health information.  Results are cached for mirrorHealthTTL
// (30 s) to avoid redundant probing on rapid page switches.
func (a *App) GetMirrorHealth() []MirrorInfo {
	mirrorHealthMu.Lock()
	if mirrorHealthCache != nil && time.Since(mirrorHealthAt) < mirrorHealthTTL {
		out := make([]MirrorInfo, len(mirrorHealthCache))
		copy(out, mirrorHealthCache)
		mirrorHealthMu.Unlock()
		return out
	}
	mirrorHealthMu.Unlock()

	mirrors := a.getAllMirrors()
	ch := make(chan MirrorInfo, len(mirrors))
	var wg sync.WaitGroup

	for idx, m := range mirrors {
		wg.Add(1)
		go func(mirrorURL string, isCustom bool) {
			defer wg.Done()
			client := &http.Client{Timeout: probeTimeout}
			start := time.Now()
			resp, err := client.Head(buildMirrorURL(probeURL, mirrorURL))
			elapsed := time.Since(start).Milliseconds()

			r := MirrorInfo{
				URL:         mirrorURL,
				Label:       extractHost(mirrorURL),
				IsCustom:    isCustom,
				LastChecked: time.Now().Format(time.RFC3339),
			}
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					r.Alive = true
					r.Latency = elapsed
				}
			}
			ch <- r
		}(m, idx >= len(defaultMirrors()))
	}
	wg.Wait()
	close(ch)

	out := make([]MirrorInfo, 0, len(mirrors))
	for r := range ch {
		out = append(out, r)
	}

	mirrorHealthMu.Lock()
	mirrorHealthCache = out
	mirrorHealthAt = time.Now()
	mirrorHealthMu.Unlock()
	return out
}

// TestSingleMirror probes one mirror URL and returns its latency in
// milliseconds, or -1 if the probe failed.
func (a *App) TestSingleMirror(mirrorURL string) int64 {
	client := &http.Client{Timeout: probeTimeout}
	start := time.Now()
	resp, err := client.Head(buildMirrorURL(probeURL, mirrorURL))
	if err != nil {
		return -1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return -1
	}
	return time.Since(start).Milliseconds()
}

// ── URL construction ────────────────────────────────────────────────

// probeURL is a small well-known URL used to measure mirror latency.
const probeURL = "https://raw.githubusercontent.com/yairm210/Unciv/master/README.md"

// isCloneMirror detects whether the mirror is a full GitHub clone (e.g. kkgithub.com)
// rather than a reverse-proxy / ghproxy-style mirror.
func isCloneMirror(mirror string) bool {
	mirror = strings.ToLower(mirror)
	return strings.Contains(mirror, "kkgithub") ||
		strings.Contains(mirror, "bgithub") ||
		strings.Contains(mirror, "fastgit") ||
		strings.Contains(mirror, "gitclone")
}

// buildMirrorURL constructs a proxied URL appropriate for the mirror's type.
//
//   - ghproxy-style (default): prepend the mirror base to the host+path.
//     Example: https://ghproxy.com/ + github.com/user/repo → https://ghproxy.com/github.com/user/repo
//
//   - clone-style (kkgithub, nuaa.cf): replace the github.com domain with the
//     mirror's own domain.
//     Example: https://github.com/user/repo → https://kkgithub.com/user/repo
func buildMirrorURL(rawURL, mirror string) string {
	mirror = strings.TrimRight(mirror, "/")
	if isCloneMirror(mirror) {
		u, err := url.Parse(mirror)
		if err == nil {
			r := strings.Replace(rawURL, "github.com", u.Host, 1)
			r = strings.Replace(r, "raw.githubusercontent.com", "raw."+u.Host, 1)
			if r != rawURL {
				return r
			}
		}
	}
	// ghproxy-style: prepend mirror to the path portion
	clean := strings.TrimPrefix(rawURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	return mirror + "/" + clean
}

// mirrorURL builds a proxied URL by prepending the mirror base to the
// raw URL's host+path.  Delegates to buildMirrorURL for type-aware dispatch.
func mirrorURL(rawURL, mirror string) string {
	return buildMirrorURL(rawURL, mirror)
}

// applyMirror transforms rawURL according to the given mode and mirror.
// mode "direct" or "" → returns rawURL unchanged.
// mode "mirror"       → uses buildMirrorURL (type-aware).
// mode "custom"       → returns mirror + rawURL (full proxy).
// This is the canonical implementation; kept for backward compatibility
// with existing callers (BuildDownloadURL) and tests.
func applyMirror(rawURL, mode, mirror string) string {
	if mode == "" || mode == "direct" {
		return rawURL
	}
	return buildMirrorURL(rawURL, mirror)
}

// ── Helpers ─────────────────────────────────────────────────────────

func extractHost(mirrorURL string) string {
	u, err := url.Parse(mirrorURL)
	if err != nil {
		return mirrorURL
	}
	return u.Host
}


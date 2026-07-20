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

// defaultMirrors returns the built-in mirror list.
func defaultMirrors() []string {
	return []string{
		"https://ghproxy.com/",
		"https://mirror.ghproxy.com/",
		"https://gh.api.99988866.xyz/",
		"https://ghfast.top/",
		"https://kkgithub.com/",
		"https://hub.nuaa.cf/",
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
			resp, err := client.Head(mirrorURL + strings.TrimPrefix(probeURL, "https://"))
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
	resp, err := client.Head(mirrorURL + strings.TrimPrefix(probeURL, "https://"))
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

// mirrorURL builds a proxied URL by prepending the mirror base to the
// raw URL's host+path.
func mirrorURL(rawURL, mirror string) string {
	clean := strings.TrimPrefix(rawURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	return strings.TrimRight(mirror, "/") + "/" + clean
}

// applyMirror transforms rawURL according to the given mode and mirror.
// mode "direct" or "" → returns rawURL unchanged.
// mode "mirror"       → prepends the mirror base.
// mode "custom"       → returns mirror + rawURL (full proxy).
// This is the canonical implementation; kept for backward compatibility
// with existing callers (BuildDownloadURL) and tests.
func applyMirror(rawURL, mode, mirror string) string {
	if mode == "" || mode == "direct" {
		return rawURL
	}
	clean := strings.TrimPrefix(rawURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	return strings.TrimRight(mirror, "/") + "/" + clean
}

// ── Helpers ─────────────────────────────────────────────────────────

func extractHost(mirrorURL string) string {
	u, err := url.Parse(mirrorURL)
	if err != nil {
		return mirrorURL
	}
	return u.Host
}


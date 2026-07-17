package app

import (
	"context"
	"fmt"
	"io"
	"crypto/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── Types ─────────────────────────────────────────────────────────────

// ProxyConfig describes how to route download requests.
type ProxyConfig struct {
	Mode        string `json:"mode"`        // "direct" | "mirror" | "custom"
	MirrorURL   string `json:"mirrorUrl"`
	CustomProxy string `json:"customProxy"`
}

// DownloadTask is the public-facing download status returned to the frontend.
type DownloadTask struct {
	ID         string  `json:"id"`
	URL        string  `json:"url"`
	Filename   string  `json:"filename"`
	Status     string  `json:"status"` // queued | downloading | paused | completed | failed
	TotalSize  int64   `json:"totalSize"`
	Downloaded int64   `json:"downloaded"`
	Percent    float64 `json:"percent"`
	Speed      string  `json:"speed"`
	Error      string  `json:"error,omitempty"`
}

// internal task wraps the public struct with lifecycle controls.
type dlTask struct {
	DownloadTask
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	file     *os.File
	filePath string

	// speed tracking — 3 s sliding window
	speedSamples []speedSample
}

type speedSample struct {
	bytes int64
	at    time.Time
}

// ── Mirror list ───────────────────────────────────────────────────────

func getMirrors() []string {
	return []string{
		"https://ghproxy.com/",
		"https://mirror.ghproxy.com/",
		"https://gh.api.99988866.xyz/",
		"https://ghfast.top/",
		"https://kkgithub.com/",
		"https://hub.nuaa.cf/",
	}
}

// probeURL is a small well-known URL used to measure mirror latency.
const probeURL = "https://raw.githubusercontent.com/yairm210/Unciv/master/README.md"

// ── URL builder ───────────────────────────────────────────────────────

// BuildDownloadURL transforms a raw URL through the configured proxy.
func (a *App) BuildDownloadURL(rawURL string, cfg ProxyConfig) string {
	if !strings.HasPrefix(rawURL, "http") {
		return rawURL
	}
	switch cfg.Mode {
	case "direct":
		return rawURL
	case "mirror":
		clean := strings.TrimPrefix(rawURL, "https://")
		clean = strings.TrimPrefix(clean, "http://")
		return strings.TrimRight(cfg.MirrorURL, "/") + "/" + clean
	case "custom":
		return cfg.CustomProxy + rawURL
	default:
		return rawURL
	}
}

// ── Latency test ──────────────────────────────────────────────────────

// TestMirrorsLatency concurrently probes every mirror by fetching a small
// known URL through it, returning latency in milliseconds.  Mirrors that
// time out (>5 s) or return non-200 are omitted.
func (a *App) TestMirrorsLatency() map[string]int64 {
	mirrors := getMirrors()
	type result struct {
		url     string
		latency int64
	}
	ch := make(chan result, len(mirrors))
	var wg sync.WaitGroup

	for _, m := range mirrors {
		wg.Add(1)
		go func(mirror string) {
			defer wg.Done()
			client := &http.Client{Timeout: 6 * time.Second}
			// Build full probe URL through mirror
			testURL := strings.TrimRight(mirror, "/") + "/" + probeURL
			start := time.Now()
			resp, err := client.Head(testURL)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return
			}
			ch <- result{mirror, time.Since(start).Milliseconds()}
		}(m)
	}
	wg.Wait()
	close(ch)

	out := map[string]int64{}
	for r := range ch {
		out[r.url] = r.latency
	}
	return out
}

// ── Public download API ──────────────────────────────────────────────

// initDownloads ensures the download maps are initialised on first use.
func (a *App) initDownloads() {
	if a.dlTasks == nil {
		a.dlTasks = map[string]*dlTask{}
	}
	if a.dlDir == "" {
		a.dlDir = filepath.Join(os.TempDir(), "unciv-mm-downloads")
		os.MkdirAll(a.dlDir, 0755)
	}
}

// StartDownloadWithMirror builds a proxied URL and starts the download.
// If mirror is empty or "direct", the raw URL is used unchanged.
func (a *App) StartDownloadWithMirror(rawURL, filename, mirror string) (string, error) {
	dlURL := rawURL
	if mirror != "" && mirror != "direct" {
		clean := strings.TrimPrefix(rawURL, "https://")
		clean = strings.TrimPrefix(clean, "http://")
		dlURL = strings.TrimRight(mirror, "/") + "/" + clean
	}
	return a.StartDownload(dlURL, filename)
}

// ValidateDownloadURL checks a URL for obvious dangers before download.
// Returns an error string if rejected, empty string if ok, or a warning
// string if the URL is allowed but looks suspicious.
func (a *App) ValidateDownloadURL(rawURL string) (warning string, err error) {
	lower := strings.ToLower(rawURL)

	// Block dangerous protocols
	for _, p := range []string{"javascript:", "data:", "file:", "vbscript:"} {
		if strings.HasPrefix(lower, p) {
			return "", fmt.Errorf("不安全的协议: %s", strings.SplitN(p, ":", 2)[0])
		}
	}

	// Require HTTPS for non-localhost URLs
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://localhost") && !strings.HasPrefix(lower, "http://127.") {
		return "", fmt.Errorf("仅支持 HTTPS 链接，安全起见不接受明文 HTTP")
	}

	// Known safe domains (no warning)
	safeDomains := []string{
		"github.com/", "raw.githubusercontent.com/",
		"objects.githubusercontent.com/", "codeload.github.com/",
		"ghproxy.com/", "mirror.ghproxy.com/", "gh.api.99988866.xyz/",
		"ghfast.top/", "kkgithub.com/", "hub.nuaa.cf/",
	}
	for _, d := range safeDomains {
		if strings.Contains(lower, d) {
			return "", nil
		}
	}
	return "该 URL 非 GitHub 官方域名，请确认来源可信", nil
}

// StartDownload enqueues a new download.  Up to 2 downloads run concurrently;
// additional tasks are queued and started automatically when a slot frees.
func (a *App) StartDownload(url, filename string) (string, error) {
	a.dlMu.Lock()
	a.initDownloads()
	a.pruneOldTasks()

	active := 0
	for _, t := range a.dlTasks {
		if t.Status == "downloading" {
			active++
		}
	}

	id := genTaskID()
	status := "downloading"
	if active >= 2 {
		status = "queued"
	}
	t := &dlTask{
		DownloadTask: DownloadTask{
			ID:       id,
			URL:      url,
			Filename: filename,
			Status:   status,
		},
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	a.dlTasks[id] = t
	a.dlMu.Unlock()

	if status == "downloading" {
		go a.runDownload(t)
	}
	return id, nil
}

// startNextQueued kicks off the oldest queued download, if any.
func (a *App) startNextQueued() {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	for _, t := range a.dlTasks {
		if t.Status == "queued" {
			t.Status = "downloading"
			t.ctx, t.cancel = context.WithCancel(context.Background())
			go a.runDownload(t)
			return
		}
	}
}

// pruneOldTasks removes completed/failed tasks older than 1 hour to prevent
// unbounded map growth.
func (a *App) pruneOldTasks() {
	// Quick check: if fewer than 50 tasks, skip
	if len(a.dlTasks) < 50 {
		return
	}
	for id, t := range a.dlTasks {
		if t.Status == "completed" || t.Status == "failed" {
			// Clean up any leftover file references
			if t.file != nil {
				t.file.Close()
				t.file = nil
			}
			t.speedSamples = nil
			delete(a.dlTasks, id)
		}
	}
}

// PauseDownload pauses an active download.  Partial progress is kept on
// disk so it can be resumed later.
func (a *App) PauseDownload(taskID string) error {
	a.dlMu.Lock()
	t, ok := a.dlTasks[taskID]
	a.dlMu.Unlock()
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskID)
	}
	t.mu.Lock()
	if t.Status == "downloading" {
		t.cancel() // kill current workers
		t.Status = "paused"
	}
	t.mu.Unlock()
	return nil
}

// ResumeDownload restarts a paused download from where it left off.
func (a *App) ResumeDownload(taskID string) error {
	a.dlMu.Lock()
	t, ok := a.dlTasks[taskID]
	a.dlMu.Unlock()
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskID)
	}
	t.mu.Lock()
	if t.Status != "paused" {
		t.mu.Unlock()
		return fmt.Errorf("任务不在暂停状态")
	}
	t.Status = "downloading"
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.mu.Unlock()

	go a.runDownload(t)
	return nil
}

// CancelDownload stops a download and removes its partial file.
func (a *App) CancelDownload(taskID string) error {
	a.dlMu.Lock()
	t, ok := a.dlTasks[taskID]
	a.dlMu.Unlock()
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskID)
	}
	t.mu.Lock()
	t.cancel()
	t.Status = "failed"
	t.Error = "用户取消"
	if t.file != nil {
		t.file.Close()
	}
	t.mu.Unlock()

	if t.filePath != "" {
		os.Remove(t.filePath)
	}
	return nil
}

// RemoveDownload deletes a task from the list.  Active downloads are
// cancelled first.
func (a *App) RemoveDownload(taskID string) error {
	a.CancelDownload(taskID) // best-effort cancel
	a.dlMu.Lock()
	delete(a.dlTasks, taskID)
	a.dlMu.Unlock()
	return nil
}

// RetryDownload re-queues a failed download.
func (a *App) RetryDownload(taskID string) error {
	a.dlMu.Lock()
	t, ok := a.dlTasks[taskID]
	if !ok {
		a.dlMu.Unlock()
		return fmt.Errorf("任务不存在")
	}
	t.Downloaded = 0
	t.Percent = 0
	t.Error = ""
	t.Status = "downloading"
	t.ctx, t.cancel = context.WithCancel(context.Background())
	a.dlMu.Unlock()
	go a.runDownload(t)
	return nil
}

// GetDownloadList returns every tracked download task.
func (a *App) GetDownloadList() []DownloadTask {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	a.initDownloads()
	var out []DownloadTask
	for _, t := range a.dlTasks {
		t.mu.Lock()
		out = append(out, t.DownloadTask)
		t.mu.Unlock()
	}
	if out == nil {
		out = []DownloadTask{}
	}
	return out
}

// ── Internal download runner ──────────────────────────────────────────

func (a *App) runDownload(t *dlTask) {
	defer RecoverLog("download")

	// 1. HEAD to get total size + range support
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Head(t.URL)
	if err != nil {
		a.failTask(t, "无法连接: "+err.Error())
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.failTask(t, fmt.Sprintf("服务器返回 %d，请检查 URL 是否为直链（GitHub 需用 archive/...zip 格式）", resp.StatusCode))
		return
	}

	// Reject non-binary responses (e.g. HTML pages served as zip)
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		a.failTask(t, "URL 返回的是网页而非文件，请使用直链（如 .../archive/...zip）")
		return
	}

	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"

	t.mu.Lock()
	t.TotalSize = totalSize
	t.mu.Unlock()

	// 2. Create final file and pre-allocate
	filePath := filepath.Join(a.dlDir, t.Filename)
	file, err := os.Create(filePath)
	if err != nil {
		a.failTask(t, "无法创建文件: "+err.Error())
		return
	}
	if totalSize > 0 {
		file.Truncate(totalSize)
	}

	t.mu.Lock()
	t.file = file
	t.filePath = filePath
	t.mu.Unlock()

	// 3. Download
	if supportsRange && totalSize > 0 {
		a.concurrentDownload(t, totalSize)
	} else {
		a.singleThreadDownload(t)
	}

	// 4. Check outcome
	t.mu.Lock()
	finished := t.Status == "downloading" // not cancelled/failed mid-way
	if finished {
		LogInfo("下载", "完成: id=%s filename=%s", t.ID, t.Filename)
		t.Status = "completed"
		t.Percent = 100
	}
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}
	t.mu.Unlock()

	if finished {
		LogInfo("下载", "完成: id=%s filename=%s", t.ID, t.Filename)
		runtime.EventsEmit(a.ctx, "download:complete", map[string]interface{}{
			"id":       t.ID,
			"filename": t.Filename,
			"filePath": t.filePath,
		})
	}
		a.startNextQueued()
}

// ── Concurrent chunked download ──────────────────────────────────────

func (a *App) concurrentDownload(t *dlTask, totalSize int64) {
	const chunkSize = 5 * 1024 * 1024 // 5 MiB
	totalChunks := int(totalSize / chunkSize)
	if totalSize%chunkSize != 0 {
		totalChunks++
	}

	// semaphore limits concurrent workers to 3
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	errCh := make(chan error, totalChunks)

	for i := 0; i < totalChunks; i++ {
		// Respect pause / cancel
		t.mu.Lock()
		if t.Status != "downloading" {
			t.mu.Unlock()
			break
		}
		t.mu.Unlock()

		sem <- struct{}{}
		wg.Add(1)

		go func(chunkIdx int) {
			defer wg.Done()
			defer func() { <-sem }()

			start := int64(chunkIdx) * chunkSize
			end := start + chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}

			buf := make([]byte, end-start+1)

			for retry := 0; retry < 3; retry++ {
				t.mu.Lock()
				if t.Status != "downloading" {
					t.mu.Unlock()
					return
				}
				t.mu.Unlock()

				req, _ := http.NewRequestWithContext(t.ctx, "GET", t.URL, nil)
				req.Header.Set("User-Agent", "unciv-mod-manager/1.0")
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					if retry == 2 {
						errCh <- fmt.Errorf("chunk %d failed: %w", chunkIdx, err)
					}
					continue
				}
				n, err := io.ReadFull(resp.Body, buf)
				resp.Body.Close()
				if err != nil && err != io.ErrUnexpectedEOF {
					if retry == 2 {
						errCh <- fmt.Errorf("chunk %d read failed: %w", chunkIdx, err)
					}
					continue
				}

				t.mu.Lock()
				if t.file != nil {
					t.file.WriteAt(buf[:n], start)
				}
				t.Downloaded += int64(n)
				a.recalcSpeed(t, int64(n))
				if t.TotalSize > 0 {
					t.Percent = float64(t.Downloaded) / float64(t.TotalSize) * 100
				}
				t.mu.Unlock()

				a.emitProgress(t)
				return // success
			}

			errCh <- fmt.Errorf("chunk %d exhausted retries", chunkIdx)
		}(i)
	}
	wg.Wait()
	close(errCh)

	// If any chunk errored, fail the task
	for err := range errCh {
		if err != nil {
			a.failTask(t, err.Error())
			return
		}
	}
}

// ── Single-thread fallback ────────────────────────────────────────────

func (a *App) singleThreadDownload(t *dlTask) {
	req, _ := http.NewRequestWithContext(t.ctx, "GET", t.URL, nil)
	req.Header.Set("User-Agent", "unciv-mod-manager/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.failTask(t, "下载失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.failTask(t, fmt.Sprintf("服务器返回 %d，请确认 URL 为有效直链", resp.StatusCode))
		return
	}
	if t.TotalSize == 0 {
		t.TotalSize = resp.ContentLength
	}

	buf := make([]byte, 32*1024) // 32 KiB copy buffer
	for {
		t.mu.Lock()
		if t.Status != "downloading" {
			t.mu.Unlock()
			return
		}
		t.mu.Unlock()

		n, err := resp.Body.Read(buf)
		if n > 0 {
			t.mu.Lock()
			if t.file != nil {
				t.file.Write(buf[:n])
			}
			t.Downloaded += int64(n)
			a.recalcSpeed(t, int64(n))
			if t.TotalSize > 0 {
				t.Percent = float64(t.Downloaded) / float64(t.TotalSize) * 100
			}
			t.mu.Unlock()
			a.emitProgress(t)
		}
		if err != nil {
			if err != io.EOF {
				a.failTask(t, "下载中断: "+err.Error())
			}
			return
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────

func (a *App) failTask(t *dlTask, msg string) {
	LogWarn("下载", "失败: id=%s url=%s error=%s", t.ID, t.URL, msg)
	t.mu.Lock()
	t.Status = "failed"
	t.Error = msg
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}
	t.mu.Unlock()
	if t.filePath != "" {
		os.Remove(t.filePath)
	}
	runtime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
		"id":    t.ID,
		"error": msg,
	})
}

func (a *App) emitProgress(t *dlTask) {
	runtime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
		"id":         t.ID,
		"downloaded": t.Downloaded,
		"totalSize":  t.TotalSize,
		"percent":    t.Percent,
		"speed":      t.Speed,
		"status":     t.Status,
	})
}

// recalcSpeed maintains a 3-second sliding window of samples.
func (a *App) recalcSpeed(t *dlTask, bytes int64) {
	now := time.Now()
	t.speedSamples = append(t.speedSamples, speedSample{bytes: bytes, at: now})
	// prune samples older than 3 s
	cutoff := now.Add(-3 * time.Second)
	var sum int64
	keep := t.speedSamples[:0]
	for _, s := range t.speedSamples {
		if s.at.After(cutoff) {
			keep = append(keep, s)
			sum += s.bytes
		}
	}
	t.speedSamples = keep
	if len(keep) > 0 {
		bps := float64(sum) / 3.0
		t.Speed = formatSpeed(int64(bps))
	}
}

func formatSpeed(bps int64) string {
	if bps < 1024 {
		return fmt.Sprintf("%d B/s", bps)
	}
	if bps < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bps)/1024)
	}
	return fmt.Sprintf("%.1f MB/s", float64(bps)/(1024*1024))
}

func genTaskID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("dl_%x", b)
}

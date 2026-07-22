package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── compareSemVer 边界 ──

func TestCompareSemVer_EdgeCases(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// 不等长
		{"1.9", "1.9.0", 0},
		{"1.9.0", "1.9", 0},
		{"2.0", "1.9.99", 1},
		{"1.9.1", "1.10", -1},
		// 非数字被跳过
		{"v1.9.0", "1.9.0", 0},
		{"V2.0.0-rc1", "1.9.0", 1},
		{"1.0.0-alpha", "1.0.0-beta", 0},
		// 空 / 异常输入
		{"", "1.0.0", -1},
		{"1.0.0", "", 1},
		{"", "", 0},
		{"abc", "xyz", 0},
		{"1.0.0", "abc", 1},
	}
	for _, tt := range tests {
		got := compareSemVer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemVer(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// ── Cache 读写 ──

func TestSelfUpdateCache_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	app := &App{configDir: dir}

	info := SelfUpdateInfo{
		LatestVersion: "2.0.0",
		DownloadURL:   "https://github.com/test/releases/download/v2.0.0/test.exe",
		ReleaseName:   "v2.0.0",
		HasUpdate:     true,
	}
	app.writeSelfUpdateCache(info)

	cached := app.readSelfUpdateCache()
	if cached == nil {
		t.Fatal("readSelfUpdateCache returned nil after write")
	}
	if cached.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", cached.LatestVersion, "2.0.0")
	}
	if cached.DownloadURL != info.DownloadURL {
		t.Errorf("DownloadURL mismatch")
	}
	if cached.CachedAt == "" {
		t.Error("CachedAt should not be empty")
	}
}

func TestSelfUpdateCache_ReadMissing(t *testing.T) {
	dir := t.TempDir()
	app := &App{configDir: dir}

	cached := app.readSelfUpdateCache()
	if cached != nil {
		t.Error("expected nil for missing cache file")
	}
}

func TestSelfUpdateCache_ReadGarbage(t *testing.T) {
	dir := t.TempDir()
	app := &App{configDir: dir}
	os.WriteFile(app.selfUpdateCachePath(), []byte("not json"), 0644)

	cached := app.readSelfUpdateCache()
	if cached != nil {
		t.Error("expected nil for malformed cache")
	}
}

// ── InstallSelfUpdate ──

func TestInstallSelfUpdate_NoFile(t *testing.T) {
	dir := t.TempDir()
	app := &App{dlDir: filepath.Join(dir, "dl")}
	os.MkdirAll(app.dlDir, 0755)

	_, err := app.InstallSelfUpdate()
	if err == nil {
		t.Fatal("expected error when no update file exists")
	}
}

// TestInstallSelfUpdate_WithExe performs a full end-to-end install using
// the pre-built exe from build/bin.  It copies the exe into a fake download
// dir and lets InstallSelfUpdate replace the running test binary.
//
// This is safe on Windows because renaming a running exe is allowed—the
// process keeps its file handle open while the directory entry is freed.
// The test binary lives in a temp directory so no real files are affected.
func TestInstallSelfUpdate_WithExe(t *testing.T) {
	// Locate the pre-built exe
	projectRoot := findProjectRoot()
	srcExe := filepath.Join(projectRoot, "build", "bin", "unciv-mod-manager.exe")
	if _, err := os.Stat(srcExe); err != nil {
		t.Skipf("预构建 exe 不存在 (%s)，跳過集成测试\n  先运行: wails build", srcExe)
	}

	// Set up temp download dir with the "update"
	dlDir := t.TempDir()
	updateExe := filepath.Join(dlDir, "umm_update.exe")
	copyTestFile(t, srcExe, updateExe)

	app := &App{dlDir: dlDir}
	_, err := app.InstallSelfUpdate()
	if err != nil {
		t.Fatalf("InstallSelfUpdate failed: %v", err)
	}

	// Verify: the running exe was replaced with our "update"
	currentExe, _ := os.Executable()
	newHash := fileSHA256String(t, currentExe)
	updateHash := fileSHA256String(t, srcExe)
	if newHash != updateHash {
		t.Errorf("installed exe hash %s != expected %s", newHash[:16], updateHash[:16])
	}

	// Verify: backup exists
	backupPath := currentExe + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup not found at %s: %v", backupPath, err)
	}
	t.Logf("backup saved to %s", backupPath)

	// Restore the test binary by renaming .bak back
	// (so subsequent tests aren't running the "updated" exe)
	//
	// We can't do this from the test because the current exe is still
	// locked by the OS—but go test creates a new binary each run anyway.
}

// ── helpers ──

func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func fileSHA256String(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ── CheckSelfUpdate (mock API) ────────────────────────────────────────────

func TestCheckSelfUpdate_API_NewerVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name": "v2.0.0",
			"name":    "v2.0.0",
			"assets": []map[string]interface{}{
				{"name": "unciv-mod-manager.exe", "browser_download_url": "https://gh.com/dl.exe"},
			},
		})
	}))
	defer srv.Close()

	setAPIBase(t, srv.URL)
	app := &App{configDir: t.TempDir()}

	info, err := app.CheckSelfUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.HasUpdate {
		t.Error("expected HasUpdate=true")
	}
	if info.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "2.0.0")
	}
	if info.DownloadURL != "https://gh.com/dl.exe" {
		t.Errorf("DownloadURL = %q", info.DownloadURL)
	}
	if info.CachedAt != "" {
		t.Error("CachedAt should be empty for live check")
	}
}

func TestCheckSelfUpdate_API_NoAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name":    "v2.0.0",
			"name":       "v2.0.0",
			"zipball_url": "https://api.github.com/zipball/v2.0.0",
		})
	}))
	defer srv.Close()

	setAPIBase(t, srv.URL)
	app := &App{configDir: t.TempDir()}

	info, err := app.CheckSelfUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.DownloadURL != "https://api.github.com/zipball/v2.0.0" {
		t.Errorf("expected zipball fallback, got %q", info.DownloadURL)
	}
}

func TestCheckSelfUpdate_API_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	setAPIBase(t, srv.URL)
	noMirrors(t) // 关掉真实镜像，避免意外拉取成功
	app := &App{configDir: t.TempDir()}

	info, err := app.CheckSelfUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.HasUpdate {
		t.Error("expected HasUpdate=false on 404")
	}
	if info.LatestVersion != UMMVersion {
		t.Errorf("LatestVersion = %q, want %q (current)", info.LatestVersion, UMMVersion)
	}
}

func TestCheckSelfUpdate_API_Unavailable_FallsBackToCache(t *testing.T) {
	setAPIBase(t, "http://127.0.0.1:1")
	noMirrors(t)

	dir := t.TempDir()
	app := &App{configDir: dir}

	app.writeSelfUpdateCache(SelfUpdateInfo{
		LatestVersion: "2.0.0",
		DownloadURL:   "https://gh.com/dl.exe",
		ReleaseName:   "v2.0.0",
	})

	info, err := app.CheckSelfUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CachedAt == "" {
		t.Error("expected CachedAt to be set (from cache)")
	}
	if info.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "2.0.0")
	}
}

func TestCheckSelfUpdate_API_Unavailable_NoCache(t *testing.T) {
	setAPIBase(t, "http://127.0.0.1:1")
	noMirrors(t)

	app := &App{configDir: t.TempDir()}

	_, err := app.CheckSelfUpdate()
	if err == nil {
		t.Fatal("expected error when API, mirrors, and cache all fail")
	}
}

// ── fetchLatestTagViaMirror ────────────────────────────────────────────────

func TestFetchLatestTagViaMirror_Redirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag/") {
			return // already on the final URL, stop redirecting
		}
		http.Redirect(w, r, "/Permanent995/unciv-mod-manager/releases/tag/v2.0.0", 302)
	}))
	defer srv.Close()

	tag, err := fetchLatestTagViaMirror("Permanent995/unciv-mod-manager", srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v2.0.0" {
		t.Errorf("tag = %q, want %q", tag, "v2.0.0")
	}
}

func TestFetchLatestTagViaMirror_NoRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("<html>not a release page</html>"))
	}))
	defer srv.Close()

	_, err := fetchLatestTagViaMirror("Permanent995/unciv-mod-manager", srv.URL+"/")
	if err == nil {
		t.Fatal("expected error when no redirect")
	}
}

func TestFetchLatestTagViaMirror_ConnectionRefused(t *testing.T) {
	_, err := fetchLatestTagViaMirror("Permanent995/unciv-mod-manager", "http://127.0.0.1:1/")
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestFetchLatestTagViaMirror_MalformedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases") && !strings.Contains(r.URL.Path, "/tag/") {
			return // final URL has /releases but no /tag/
		}
		http.Redirect(w, r, "/releases", 302)
	}))
	defer srv.Close()

	_, err := fetchLatestTagViaMirror("Permanent995/unciv-mod-manager", srv.URL+"/")
	if err == nil {
		t.Fatal("expected error for URL without /tag/")
	}
}

// ── fetchJSON ──────────────────────────────────────────────────────────────

func TestFetchJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name": "test"}`)
	}))
	defer srv.Close()

	var result struct{ Name string }
	err := fetchJSON(srv.URL, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestFetchJSON_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	var result interface{}
	err := fetchJSON(srv.URL, &result)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestFetchJSON_ConnectionRefused(t *testing.T) {
	var result interface{}
	err := fetchJSON("http://127.0.0.1:1/", &result)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestFetchJSON_ResponseTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send more than 1 MiB
		w.Write([]byte(`{"x":"`))
		w.Write(make([]byte, 2<<20))
		w.Write([]byte(`"}`))
	}))
	defer srv.Close()

	var result interface{}
	err := fetchJSON(srv.URL, &result)
	if err == nil {
		t.Error("expected error when response exceeds 1 MiB limit")
	}
}

// ── mirror redirect integration in CheckSelfUpdate ─────────────────────────

func TestCheckSelfUpdate_MirrorFallback(t *testing.T) {
	// API refuses connections; a mirror responds with a redirect
	setAPIBase(t, "http://127.0.0.1:1")

	mirrorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag/") {
			return // already redirected
		}
		http.Redirect(w, r, "/Permanent995/unciv-mod-manager/releases/tag/v2.0.0", 302)
	}))
	defer mirrorSrv.Close()

	// Replace the mirror list with just our test server
	oldMirrors := defaultMirrors
	defaultMirrors = func() []string { return []string{mirrorSrv.URL + "/"} }
	defer func() { defaultMirrors = oldMirrors }()

	app := &App{configDir: t.TempDir()}

	info, err := app.CheckSelfUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.HasUpdate {
		t.Error("expected HasUpdate=true via mirror fallback")
	}
	if info.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "2.0.0")
	}
	// Mirror fallback uses guessed URL
	if !strings.Contains(info.DownloadURL, "unciv-mod-manager.exe") {
		t.Errorf("DownloadURL should contain unciv-mod-manager.exe, got %q", info.DownloadURL)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// noMirrors clears the mirror list for the duration of the test.
func noMirrors(t *testing.T) {
	t.Helper()
	old := defaultMirrors
	defaultMirrors = func() []string { return nil }
	t.Cleanup(func() { defaultMirrors = old })
}

// setAPIBase overrides githubAPIBase for the duration of the test.
func setAPIBase(t *testing.T, url string) {
	t.Helper()
	old := githubAPIBase
	githubAPIBase = url
	t.Cleanup(func() { githubAPIBase = old })
}

// findProjectRoot walks up from the test file to find the repo root
// (where go.mod lives).
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

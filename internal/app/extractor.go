package app

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// dangerousExts lists file extensions that should never be extracted from a mod ZIP.
var dangerousExts = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".ps1": true,
	".vbs": true, ".sh": true, ".dll": true, ".scr": true,
	".msi": true, ".com": true, ".pif": true, ".reg": true,
	".js": true, ".vbe": true, ".wsf": true, ".msc": true,
}

// CleanupTempFile deletes a temporary file (e.g. a downloaded zip after extraction).
func (a *App) CleanupTempFile(path string) {
	if path != "" {
		os.Remove(path)
	}
}

// ExtractMod unpacks a ZIP file into the mods directory, automatically
// stripping the GitHub-style single root directory (e.g. "ModName-main/").
// Returns the final folder name inside modsDir.
func (a *App) ExtractMod(zipPath, modsDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("无法打开 ZIP: %w", err)
	}
	defer r.Close()

	if len(r.File) == 0 {
		return "", fmt.Errorf("ZIP 文件为空")
	}

	// Detect single root directory (GitHub archive pattern: Repo-main/...)
	rootName := ""
	firstParts := strings.SplitN(r.File[0].Name, "/", 2)
	hasSingleRoot := true
	candidate := firstParts[0] + "/"
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, candidate) && f.Name != firstParts[0] {
			hasSingleRoot = false
			break
		}
	}
	if hasSingleRoot {
		rootName = candidate
	}

	// Determine target folder name: strip root prefix to get the real
	// mod folder name, or use ZIP filename without extension.
	var modFolder string
	if hasSingleRoot {
		modFolder = firstParts[0]
	} else {
		modFolder = strings.TrimSuffix(filepath.Base(zipPath), ".zip")
	}
	target := filepath.Join(modsDir, modFolder)

	for _, f := range r.File {
		// Strip root prefix
		relPath := strings.TrimPrefix(f.Name, rootName)
		if relPath == "" || relPath == "/" {
			continue
		}

		outPath := filepath.Join(target, filepath.FromSlash(relPath))

		// Security: prevent zip-slip
		if !strings.HasPrefix(filepath.Clean(outPath), filepath.Clean(target)+string(os.PathSeparator)) &&
			filepath.Clean(outPath) != filepath.Clean(target) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(outPath, 0755)
			continue
		}

		// Skip files with dangerous extensions
		ext := strings.ToLower(filepath.Ext(relPath))
		if dangerousExts[ext] {
			continue
		}

		os.MkdirAll(filepath.Dir(outPath), 0755)

		src, err := f.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(outPath)
		if err != nil {
			src.Close()
			continue
		}
		io.Copy(dst, src)
		src.Close()
		dst.Close()
	}

	return modFolder, nil
}

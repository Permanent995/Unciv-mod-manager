package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"unciv-mod-manager/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed all:docs
var docsFS embed.FS

// DocInfo describes a document file in the docs/ directory.
type DocInfo struct {
	Name    string `json:"name"`    // filename (e.g. "umm-review-report.md")
	Title   string `json:"title"`   // human-readable title (derived from first # heading or filename)
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

// DocReader serves development documents to the frontend from the embedded docs/ directory.
type DocReader struct{}

// ListDocs returns all .md files in the embedded docs/ directory.
func (d *DocReader) ListDocs() ([]DocInfo, error) {
	entries, err := fs.ReadDir(docsFS, "docs")
	if err != nil {
		return nil, fmt.Errorf("无法读取开发文档目录: %w", err)
	}
	var docs []DocInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		docs = append(docs, DocInfo{
			Name:    e.Name(),
			Title:   docTitle(e.Name()),
			Size:    fi.Size(),
			ModTime: fi.ModTime().Format("2006-01-02 15:04"),
		})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	return docs, nil
}

// ReadDoc returns the raw content of a document by filename from the embedded docs/ directory.
func (d *DocReader) ReadDoc(name string) (string, error) {
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("非法文件名")
	}
	data, err := fs.ReadFile(docsFS, path.Join("docs", name))
	if err != nil {
		return "", fmt.Errorf("无法读取文档: %w", err)
	}
	return string(data), nil
}

// docTitle extracts a human-readable title from a filename.
func docTitle(name string) string {
	title := strings.TrimSuffix(name, ".md")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	// Try to extract Chinese title parts
	if idx := strings.Index(title, "设计"); idx >= 0 {
		return title[idx:]
	}
	// Capitalize first letter and clean up
	parts := strings.Fields(title)
	for i, p := range parts {
		switch p {
		case "umm":
			parts[i] = "UMM"
		case "unciv":
			parts[i] = "Unciv"
		case "vs":
			parts[i] = "vs"
		default:
			if len(p) > 0 && p[0] >= 'a' && p[0] <= 'z' {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return strings.Join(parts, " ")
}

func main() {
	// Create an instance of the app structure
	backend := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Unciv Mod Manager",
		Width:     1400,
		Height:    900,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        backend.Startup,
		Bind: []interface{}{
			backend,
			&DocReader{},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

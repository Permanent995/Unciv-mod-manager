package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// SaveInfo describes a single Unciv save file.
type SaveInfo struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	FileSize   int64    `json:"fileSize"`
	ModifiedAt string   `json:"modifiedAt"`
	CivName    string   `json:"civName,omitempty"`
	Turn       int      `json:"turn,omitempty"`
	Version    string   `json:"version,omitempty"`
	Mods       []string `json:"mods,omitempty"`
}

// SaveArchive describes one save backup in AppData/umm_backups/saves/.
type SaveArchive struct {
	Name       string `json:"name"`
	OrigName   string `json:"origName"`
	Timestamp  string `json:"timestamp"`
	Path       string `json:"path"`
	FileSize   int64  `json:"fileSize"`
	ModifiedAt string `json:"modifiedAt"`
}

// ScanSaves reads the SaveFiles/ directory and returns save metadata.
func (a *App) ScanSaves() ([]SaveInfo, error) {
	uncivPath := a.config.UncivPath
	if uncivPath == "" {
		return nil, fmt.Errorf("未设置 Unciv 路径")
	}
	saveDir := filepath.Join(uncivPath, "SaveFiles")
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		return nil, fmt.Errorf("无法读取 SaveFiles/ 目录: %w", err)
	}

	var saves []SaveInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fp := filepath.Join(saveDir, e.Name())
		fi, err := e.Info()
		if err != nil {
			continue
		}

		info := SaveInfo{
			Name:       e.Name(),
			Path:       fp,
			FileSize:   fi.Size(),
			ModifiedAt: fi.ModTime().Format("2006-01-02 15:04"),
		}

		info.tryParseMetadata(fp)
		saves = append(saves, info)
	}

	sort.Slice(saves, func(i, j int) bool {
		return saves[i].ModifiedAt > saves[j].ModifiedAt
	})

	if saves == nil {
		saves = []SaveInfo{}
	}
	return saves, nil
}

// DeleteSave removes a save file by path.
func (a *App) DeleteSave(path string) error {
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	// Safety: only allow deleting files inside SaveFiles/
	saveDir := filepath.Join(a.config.UncivPath, "SaveFiles")
	absDir, _ := filepath.Abs(saveDir)
	absPath, _ := filepath.Abs(path)
	if len(absPath) < len(absDir) || absPath[:len(absDir)] != absDir {
		return fmt.Errorf("只能删除 SaveFiles/ 目录下的文件")
	}
	return os.Remove(absPath)
}

// ── Save Backup / Archive ──

// ArchiveSave copies a save file to AppData/umm_backups/saves/.
func (a *App) ArchiveSave(savePath string) (string, error) {
	if savePath == "" {
		return "", fmt.Errorf("路径为空")
	}
	fi, err := os.Stat(savePath)
	if err != nil {
		return "", fmt.Errorf("存档文件不存在: %w", err)
	}

	backupRoot := filepath.Join(a.configDir, "umm_backups", "saves")
	os.MkdirAll(backupRoot, 0755)

	ts := time.Now().Format("2006-01-02_150405")
	dst := filepath.Join(backupRoot, fi.Name()+"_"+ts)
	if err := copyFile(savePath, dst); err != nil {
		return "", fmt.Errorf("备份存档失败: %w", err)
	}
	return dst, nil
}

// ListSaveArchives returns all save backups grouped by original filename.
func (a *App) ListSaveArchives() ([]SaveArchive, error) {
	backupRoot := filepath.Join(a.configDir, "umm_backups", "saves")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return []SaveArchive{}, nil
	}
	var archives []SaveArchive
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		// Parse original name: "filename_2026-07-17_150405"
		orig := e.Name()
		ts := ""
		if idx := strings.LastIndex(e.Name(), "_20"); idx != -1 && len(e.Name()) > idx+1+15 {
			orig = e.Name()[:idx]
			ts = e.Name()[idx+1:]
		}
		archives = append(archives, SaveArchive{
			Name:       e.Name(),
			OrigName:   orig,
			Timestamp:  ts,
			Path:       filepath.Join(backupRoot, e.Name()),
			FileSize:   fi.Size(),
			ModifiedAt: fi.ModTime().Format("2006-01-02 15:04"),
		})
	}
	return archives, nil
}

// RestoreSaveArchive copies a save backup back to SaveFiles/.
func (a *App) RestoreSaveArchive(backupPath string) (string, error) {
	if backupPath == "" {
		return "", fmt.Errorf("路径为空")
	}
	_, err := os.Stat(backupPath)
	if err != nil {
		return "", fmt.Errorf("备份文件不存在: %w", err)
	}
	target := filepath.Join(a.config.UncivPath, "SaveFiles", filepath.Base(backupPath))
	if err := copyFile(backupPath, target); err != nil {
		return "", fmt.Errorf("恢复存档失败: %w", err)
	}
	return target, nil
}

// DeleteSaveArchive removes a save backup.
func (a *App) DeleteSaveArchive(backupPath string) error {
	return os.Remove(backupPath)
}

// ── Save Metadata Parsing ──

func (s *SaveInfo) tryParseMetadata(fp string) {
	data, err := os.ReadFile(fp)
	if err != nil || len(data) < 20 {
		return
	}
	// Read up to 96KB to find nested gameParameters
	if len(data) > 2097152 {
		data = data[:2097152]
	}
	content := preprocessUncivJSON(string(data))

	// Version
	if v := gjson.Get(content, "version.createdWith.text"); v.Exists() {
		s.Version = v.String()
	} else if v := gjson.Get(content, "version.number"); v.Exists() {
		s.Version = fmt.Sprintf("v%d", v.Int())
	}

	// Turns
	if t := gjson.Get(content, "turns"); t.Exists() {
		s.Turn = int(t.Int())
	}

	// Civ — find the human player
	civs := gjson.Get(content, "civilizations")
	civs.ForEach(func(_, v gjson.Result) bool {
		if v.Get("playerType").String() == "Human" {
			s.CivName = v.Get("civName").String()
			return false
		}
		return true
	})
	if s.CivName == "" {
		if first := civs.Get("0.civName"); first.Exists() {
			s.CivName = first.String()
		}
	}

	// Mods — search in gameParameters at any nesting level
	raw := gjson.Get(content, "mods")
	if !raw.Exists() || !raw.IsArray() {
		raw = gjson.Get(content, "gameParameters.mods")
	}
	if !raw.Exists() || !raw.IsArray() {
		// Fallback: raw text search for "mods":[...
		mods := gjson.Get(content, "lastGameSetup.gameParameters.mods")
		if mods.Exists() && mods.IsArray() {
			raw = mods
		}
	}
	if raw.Exists() && raw.IsArray() {
		raw.ForEach(func(_, v gjson.Result) bool {
			s.Mods = append(s.Mods, v.String())
			return true
		})
	}
}

package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Log Level ─────────────────────────────────────────────

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ── File Logger ──────────────────────────────────────────

// Logger handles writing structured log messages to rotating files.
type Logger struct {
	mu         sync.Mutex
	logDir     string
	maxSize    int64 // bytes before rotation
	maxBackups int   // old files to keep
	file       *os.File
	size       int64
}

// LogConfig holds initialization parameters for the Logger.
type LogConfig struct {
	LogDir     string
	MaxSize    int64
	MaxBackups int
}

// NewLogger creates and initialises a Logger.
func NewLogger(cfg LogConfig) (*Logger, error) {
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}
	l := &Logger{
		logDir:     cfg.LogDir,
		maxSize:    cfg.MaxSize,
		maxBackups: cfg.MaxBackups,
	}
	if l.maxSize <= 0 {
		l.maxSize = 10 * 1024 * 1024 // 10 MB
	}
	if l.maxBackups <= 0 {
		l.maxBackups = 3
	}
	if err := l.openFile(); err != nil {
		return nil, err
	}
	return l, nil
}

// openFile opens (or creates) today's log file and seeks to the end.
func (l *Logger) openFile() error {
	name := filepath.Join(l.logDir, "umm-"+time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	l.file = f
	l.size = fi.Size()
	return nil
}

// rotate checks file size and rotates if necessary.
func (l *Logger) rotate() error {
	if l.size < l.maxSize {
		return nil
	}
	l.file.Close()
	l.file = nil

	base := filepath.Join(l.logDir, "umm-"+time.Now().Format("2006-01-02"))

	// Shift backups: .N → .N+1
	for i := l.maxBackups - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d.log", base, i)
		older := fmt.Sprintf("%s.%d.log", base, i+1)
		if _, err := os.Stat(old); err == nil {
			os.Rename(old, older)
		}
	}
	// Current → .1
	os.Rename(base+".log", base+".1.log")

	return l.openFile()
}

// write writes a single log line to the file.
func (l *Logger) write(level LogLevel, tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] [%s] [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		level.String(),
		tag,
		msg)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}
	n, _ := io.WriteString(l.file, line)
	l.size += int64(n)
	l.rotate()
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ── Global singleton ──────────────────────────────────────

var globalLog *Logger

// InitLogger initialises the global logger. Must be called once at startup.
func InitLogger(cfg LogConfig) error {
	l, err := NewLogger(cfg)
	if err != nil {
		return err
	}
	globalLog = l
	return nil
}

// Log writes a formatted message at the given level.
func Log(level LogLevel, tag, format string, args ...interface{}) {
	if globalLog != nil {
		globalLog.write(level, tag, format, args...)
	}
}

// LogInfo is a convenience wrapper for INFO-level messages.
func LogInfo(tag, format string, args ...interface{}) {
	Log(INFO, tag, format, args...)
}

// LogWarn is a convenience wrapper for WARN-level messages.
func LogWarn(tag, format string, args ...interface{}) {
	Log(WARN, tag, format, args...)
}

// LogError is a convenience wrapper for ERROR-level messages.
func LogError(tag, format string, args ...interface{}) {
	Log(ERROR, tag, format, args...)
}

// RecoverLog catches a panic and logs it. Use as:  defer RecoverLog("xxx")
func RecoverLog(goroutineName string) {
	if r := recover(); r != nil {
		LogError(goroutineName, "PANIC: %v", r)
	}
}

// ── Wails bindings ───────────────────────────────────────

// GetLogFiles returns all log file paths (newest first).
func (a *App) GetLogFiles() ([]string, error) {
	logDir := filepath.Join(a.configDir, "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "umm-") && strings.HasSuffix(e.Name(), ".log") {
			files = append(files, filepath.Join(logDir, e.Name()))
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i] > files[j] }) // newest first
	return files, nil
}

// ReadLogFile returns the full content of a log file.
func (a *App) ReadLogFile(path string) (string, error) {
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("非法路径")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取日志文件失败: %w", err)
	}
	return string(data), nil
}

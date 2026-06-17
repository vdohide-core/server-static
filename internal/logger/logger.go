package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultMaxSize = 25 * 1024 * 1024 // 25 MB

// RotatingWriter is an io.Writer that rotates log files at a given size.
type RotatingWriter struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	size    int64
	maxSize int64
}

// NewRotatingWriter opens (or creates) the log file at path.
// When the file reaches maxSizeBytes it is renamed with a timestamp suffix
// and a fresh file is opened.
func NewRotatingWriter(path string, maxSizeBytes int64) (*RotatingWriter, error) {
	rw := &RotatingWriter{
		path:    path,
		maxSize: maxSizeBytes,
	}
	if err := rw.openOrCreate(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *RotatingWriter) openOrCreate() error {
	f, err := os.OpenFile(rw.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rw.file = f
	rw.size = info.Size()
	return nil
}

func (rw *RotatingWriter) rotate() error {
	if rw.file != nil {
		rw.file.Close()
		rw.file = nil
	}
	// Rename with timestamp: server-static_20060102_150405.log
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(rw.path)
	base := rw.path[:len(rw.path)-len(ext)]
	newPath := fmt.Sprintf("%s_%s%s", base, ts, ext)
	_ = os.Rename(rw.path, newPath)
	return rw.openOrCreate()
}

// Write implements io.Writer with automatic rotation.
func (rw *RotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.size+int64(len(p)) >= rw.maxSize {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	rw.size += int64(n)
	return n, err
}

// Close closes the underlying file.
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// Init configures the global logger to write ONLY to a rotating log file.
// Returns a Closer that should be deferred in main().
//
// logPath defaults to "./logs/server-static.log" if empty.
func Init(logPath string) (io.Closer, error) {
	if logPath == "" {
		logPath = "logs/server-static.log"
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	rw, err := NewRotatingWriter(logPath, defaultMaxSize)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	// Write to file only (no stdout)
	log.SetOutput(rw)
	log.SetFlags(log.LstdFlags)

	return rw, nil
}

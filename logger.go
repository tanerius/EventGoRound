package eventgoround

import (
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	// DefaultMaxBytes is the default maximum size of log file before rotation (10MB)
	DefaultMaxBytes = 10 * 1024 * 1024 // 10 megabytes
)

// LogConfig holds configuration for event loop logging
type LogConfig struct {
	Enabled     bool   // Whether logging is enabled
	FilePath    string // Path to the log file
	IncludeInfo bool   // Whether to include INFO level logs (ERROR always logged when enabled)
}

// RotatingFileWriter implements io.Writer with automatic file rotation
// when the file size exceeds a maximum threshold
type RotatingFileWriter struct {
	filepath    string
	maxBytes    int64
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
}

// NewRotatingFileWriter creates a new rotating file writer
func NewRotatingFileWriter(filepath string, maxBytes int64) (*RotatingFileWriter, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}

	rfw := &RotatingFileWriter{
		filepath: filepath,
		maxBytes: maxBytes,
	}

	// Open initial file
	if err := rfw.openFile(); err != nil {
		return nil, err
	}

	return rfw, nil
}

// openFile opens or creates the log file
func (rfw *RotatingFileWriter) openFile() error {
	file, err := os.OpenFile(rfw.filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	rfw.currentFile = file
	rfw.currentSize = info.Size()
	return nil
}

// Write writes data to the file, rotating if necessary
func (rfw *RotatingFileWriter) Write(p []byte) (n int, err error) {
	rfw.mu.Lock()
	defer rfw.mu.Unlock()

	// Check if we need to rotate
	if rfw.currentSize+int64(len(p)) > rfw.maxBytes {
		if err := rfw.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to current file
	n, err = rfw.currentFile.Write(p)
	if err != nil {
		return n, err
	}

	rfw.currentSize += int64(n)
	return n, nil
}

// rotate closes the current file, renames it, and opens a new one
func (rfw *RotatingFileWriter) rotate() error {
	// Close current file
	if rfw.currentFile != nil {
		if err := rfw.currentFile.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}
	}

	// Rotate existing backup files (.1 -> .2, .2 -> .3, etc.)
	// We'll keep up to 5 rotated files
	maxBackups := 5
	for i := maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", rfw.filepath, i)
		newPath := fmt.Sprintf("%s.%d", rfw.filepath, i+1)

		// Check if old file exists
		if _, err := os.Stat(oldPath); err == nil {
			// Remove the destination if it exists
			os.Remove(newPath)
			// Rename old to new
			if err := os.Rename(oldPath, newPath); err != nil {
				// Log error but continue
				continue
			}
		}
	}

	// Rename current file to .1
	backupPath := fmt.Sprintf("%s.%d", rfw.filepath, 1)
	if _, err := os.Stat(rfw.filepath); err == nil {
		os.Remove(backupPath)
		if err := os.Rename(rfw.filepath, backupPath); err != nil {
			// If rename fails, just remove the old file
			os.Remove(rfw.filepath)
		}
	}

	// Open new file
	return rfw.openFile()
}

// Close closes the underlying file
func (rfw *RotatingFileWriter) Close() error {
	rfw.mu.Lock()
	defer rfw.mu.Unlock()

	if rfw.currentFile != nil {
		err := rfw.currentFile.Close()
		rfw.currentFile = nil
		return err
	}
	return nil
}

// Ensure RotatingFileWriter implements io.WriteCloser
var _ io.WriteCloser = (*RotatingFileWriter)(nil)

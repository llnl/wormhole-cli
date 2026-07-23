package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logPermissions = 0600
	logFlags       = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

type LogFile struct {
	file   string
	handle *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

func newLogFile(path string) (*LogFile, error) {
	f, err := os.OpenFile(filepath.Clean(path), logFlags, logPermissions)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	lf := &LogFile{
		file:   path,
		handle: f,
		writer: bufio.NewWriterSize(f, 64*1024), // 64KB buffer
	}

	// Start periodic flush
	go lf.periodicFlush()

	return lf, nil
}

func (l *LogFile) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.writer.Write(p)
}

func (l *LogFile) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer != nil {
		_ = l.writer.Flush()
	}

	if l.handle != nil {
		err := l.handle.Close()
		l.handle = nil

		return err
	}

	return nil
}

//

func (l *LogFile) periodicFlush() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		_ = l.writer.Flush()
		l.mu.Unlock()
	}
}

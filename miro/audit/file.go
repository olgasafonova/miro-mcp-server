package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileLogger writes audit events to JSON Lines files with rotation.
type FileLogger struct {
	mu       sync.Mutex
	config   Config
	file     *os.File
	writer   *bufio.Writer
	fileSize int64
	filePath string
	buffer   []Event
}

// NewFileLogger creates a new file-based audit logger.
func NewFileLogger(config Config) (*FileLogger, error) {
	if config.Path == "" {
		return nil, fmt.Errorf("audit log path is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(config.Path, 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logger := &FileLogger{
		config: config,
		buffer: make([]Event, 0, config.BufferSize),
	}

	// Open initial log file
	if err := logger.rotateFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// Log records an audit event.
func (l *FileLogger) Log(ctx context.Context, event Event) error {
	if !l.config.Enabled {
		return nil
	}

	// Sanitize input if configured
	if l.config.SanitizeInput && event.Input != nil {
		event.Input = SanitizeInput(event.Input)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Buffer the event
	if l.config.BufferSize > 0 {
		l.buffer = append(l.buffer, event)
		if len(l.buffer) >= l.config.BufferSize {
			return l.flushLocked()
		}
		return nil
	}

	// Write immediately if not buffering
	return l.writeEvent(event)
}

// writeEvent writes a single event to the current log file.
func (l *FileLogger) writeEvent(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Check if rotation is needed
	if l.fileSize+int64(len(data)+1) > l.config.MaxSizeBytes {
		if err := l.rotateFile(); err != nil {
			return err
		}
	}

	// Write event as JSON line
	n, err := l.writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	l.writer.WriteByte('\n')
	l.fileSize += int64(n + 1)

	return nil
}

// rotateFile closes the current file and opens a new one.
func (l *FileLogger) rotateFile() error {
	// Close existing file
	if l.file != nil {
		l.writer.Flush()
		l.file.Close()
	}

	// Generate new file name with timestamp
	filename := fmt.Sprintf("audit-%s.jsonl", time.Now().Format("2006-01-02T15-04-05"))
	l.filePath = filepath.Join(l.config.Path, filename)

	// Open new file
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat audit log file: %w", err)
	}

	l.file = file
	l.writer = bufio.NewWriter(file)
	l.fileSize = info.Size()

	// Clean up old log files
	go l.cleanupOldFiles()

	return nil
}

// cleanupOldFiles removes log files older than the retention period.
func (l *FileLogger) cleanupOldFiles() {
	if l.config.RetentionDays <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -l.config.RetentionDays)

	entries, err := os.ReadDir(l.config.Path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(l.config.Path, entry.Name()))
		}
	}
}

// Query retrieves audit events matching the specified criteria.
func (l *FileLogger) Query(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	l.mu.Lock()
	// Flush buffer first to ensure all events are written
	if len(l.buffer) > 0 {
		l.flushLocked()
	}
	l.mu.Unlock()

	// Find all log files
	entries, err := os.ReadDir(l.config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit log directory: %w", err)
	}

	// Sort files by name (timestamp order)
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, filepath.Join(l.config.Path, entry.Name()))
		}
	}
	sort.Strings(files)

	// Read and filter events from all files
	var matches []Event
	for _, filePath := range files {
		events, err := l.readEventsFromFile(ctx, filePath, opts)
		if err != nil {
			continue // Skip files that can't be read
		}
		matches = append(matches, events...)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Timestamp.After(matches[j].Timestamp)
	})

	total := len(matches)

	// Apply offset
	if opts.Offset > 0 {
		if opts.Offset >= len(matches) {
			matches = nil
		} else {
			matches = matches[opts.Offset:]
		}
	}

	// Apply limit
	hasMore := false
	if opts.Limit > 0 && len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
		hasMore = true
	}

	return &QueryResult{
		Events:  matches,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// readEventsFromFile reads and filters events from a single log file.
func (l *FileLogger) readEventsFromFile(ctx context.Context, filePath string, opts QueryOptions) ([]Event, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	const maxLineSize = 1024 * 1024 // 1MB
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return events, ctx.Err()
		default:
		}

		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue // Skip malformed lines
		}

		if matchesQuery(event, opts) {
			events = append(events, event)
		}
	}

	return events, scanner.Err()
}

// Flush writes all buffered events to disk.
func (l *FileLogger) Flush(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flushLocked()
}

// flushLocked writes buffered events (caller must hold mutex).
func (l *FileLogger) flushLocked() error {
	for _, event := range l.buffer {
		if err := l.writeEvent(event); err != nil {
			return err
		}
	}
	l.buffer = l.buffer[:0]

	if l.writer != nil {
		return l.writer.Flush()
	}
	return nil
}

// Close flushes pending events and closes the log file.
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Flush buffer
	l.flushLocked()

	// Close file
	if l.file != nil {
		l.writer.Flush()
		return l.file.Close()
	}
	return nil
}

// CurrentFilePath returns the path of the current log file.
func (l *FileLogger) CurrentFilePath() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.filePath
}

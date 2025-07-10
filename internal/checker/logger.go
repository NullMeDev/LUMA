package checker

import (
	"sync"
	"universal-checker/pkg/types"
)

// Logger manages log entries for real-time display
type Logger struct {
	entries []types.LogEntry
	mutex   sync.RWMutex
	maxSize int
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		entries: make([]types.LogEntry, 0),
		maxSize: 1000, // Keep last 1000 log entries
	}
}

// Add adds a new log entry
func (l *Logger) Add(entry types.LogEntry) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.entries = append(l.entries, entry)
	
	// Keep only the most recent entries
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

// GetRecent returns the most recent log entries
func (l *Logger) GetRecent(count int) []types.LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	if count <= 0 || count >= len(l.entries) {
		return append([]types.LogEntry(nil), l.entries...)
	}
	
	start := len(l.entries) - count
	return append([]types.LogEntry(nil), l.entries[start:]...)
}

// GetAll returns all log entries
func (l *Logger) GetAll() []types.LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	return append([]types.LogEntry(nil), l.entries...)
}

// Clear clears all log entries
func (l *Logger) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.entries = l.entries[:0]
}

// GetByLevel returns log entries filtered by level
func (l *Logger) GetByLevel(level string) []types.LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	var filtered []types.LogEntry
	for _, entry := range l.entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	
	return filtered
}

// Count returns the total number of log entries
func (l *Logger) Count() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	return len(l.entries)
}

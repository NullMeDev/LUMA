package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"universal-checker/pkg/types"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of LogLevel
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
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Session   string                 `json:"session,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// StructuredLogger provides structured logging capabilities
type StructuredLogger struct {
	level       LogLevel
	output      io.Writer
	fileOutput  *os.File
	jsonFormat  bool
	sessionID   string
	component   string
	mutex       sync.Mutex
	bufferSize  int
	buffer      []LogEntry
	bufferMutex sync.Mutex
}

// Config for StructuredLogger
type LoggerConfig struct {
	Level      LogLevel `json:"level"`
	JSONFormat bool     `json:"json_format"`
	OutputFile string   `json:"output_file"`
	BufferSize int      `json:"buffer_size"`
	Component  string   `json:"component"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	logger := &StructuredLogger{
		level:      config.Level,
		output:     os.Stdout,
		jsonFormat: config.JSONFormat,
		component:  config.Component,
		sessionID:  generateSessionID(),
		bufferSize: config.BufferSize,
		buffer:     make([]LogEntry, 0, config.BufferSize),
	}

	// Set up file output if specified
	if config.OutputFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(config.OutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}

		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}
		logger.fileOutput = file
		logger.output = file
	}

	return logger, nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().Unix())
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	sl.log(DEBUG, message, "", fields...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	sl.log(INFO, message, "", fields...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	sl.log(WARN, message, "", fields...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	sl.log(ERROR, message, errorStr, fields...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	sl.log(FATAL, message, errorStr, fields...)
	os.Exit(1)
}

// log is the internal logging method
func (sl *StructuredLogger) log(level LogLevel, message string, errorStr string, fields ...map[string]interface{}) {
	if level < sl.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Component: sl.component,
		Session:   sl.sessionID,
		Error:     errorStr,
	}

	// Merge fields if provided
	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for _, fieldMap := range fields {
			for k, v := range fieldMap {
				entry.Fields[k] = v
			}
		}
	}

	sl.writeEntry(entry)
	
	// Add to buffer if buffering is enabled
	if sl.bufferSize > 0 {
		sl.addToBuffer(entry)
	}
}

// writeEntry writes a log entry to the output
func (sl *StructuredLogger) writeEntry(entry LogEntry) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	var output string
	if sl.jsonFormat {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			// Fallback to simple format if JSON marshaling fails
			output = fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
		} else {
			output = string(jsonData) + "\n"
		}
	} else {
		// Human-readable format
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		if entry.Error != "" {
			output = fmt.Sprintf("[%s] %s [%s] %s - Error: %s\n", timestamp, entry.Level, entry.Component, entry.Message, entry.Error)
		} else {
			output = fmt.Sprintf("[%s] %s [%s] %s\n", timestamp, entry.Level, entry.Component, entry.Message)
		}
		
		// Add fields if present
		if entry.Fields != nil && len(entry.Fields) > 0 {
			output += fmt.Sprintf("  Fields: %+v\n", entry.Fields)
		}
	}

	sl.output.Write([]byte(output))
}

// addToBuffer adds an entry to the internal buffer
func (sl *StructuredLogger) addToBuffer(entry LogEntry) {
	sl.bufferMutex.Lock()
	defer sl.bufferMutex.Unlock()

	sl.buffer = append(sl.buffer, entry)
	
	// Keep buffer within size limit
	if len(sl.buffer) > sl.bufferSize {
		sl.buffer = sl.buffer[len(sl.buffer)-sl.bufferSize:]
	}
}

// GetRecentLogs returns recent log entries from the buffer
func (sl *StructuredLogger) GetRecentLogs(limit int) []LogEntry {
	sl.bufferMutex.Lock()
	defer sl.bufferMutex.Unlock()

	if limit <= 0 || limit > len(sl.buffer) {
		limit = len(sl.buffer)
	}

	// Return the most recent entries
	start := len(sl.buffer) - limit
	if start < 0 {
		start = 0
	}

	result := make([]LogEntry, limit)
	copy(result, sl.buffer[start:])
	return result
}

// SetLevel changes the logging level
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.level = level
}

// SetComponent changes the component name
func (sl *StructuredLogger) SetComponent(component string) {
	sl.component = component
}

// Close closes the logger and any file handles
func (sl *StructuredLogger) Close() error {
	if sl.fileOutput != nil {
		return sl.fileOutput.Close()
	}
	return nil
}

// LogCheckerEvent logs checker-specific events
func (sl *StructuredLogger) LogCheckerEvent(eventType string, result types.CheckResult, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	fields["event_type"] = eventType
	fields["combo"] = result.Combo.Username
	fields["config"] = result.Config
	fields["status"] = string(result.Status)
	fields["latency"] = result.Latency
	
	if result.Proxy != nil {
		fields["proxy"] = fmt.Sprintf("%s:%d", result.Proxy.Host, result.Proxy.Port)
	}

	sl.Info(fmt.Sprintf("Checker event: %s", eventType), fields)
}

// LogProxyEvent logs proxy-related events
func (sl *StructuredLogger) LogProxyEvent(eventType string, proxy types.Proxy, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	fields["event_type"] = eventType
	fields["proxy_host"] = proxy.Host
	fields["proxy_port"] = proxy.Port
	fields["proxy_type"] = string(proxy.Type)
	fields["proxy_score"] = proxy.Score
	fields["proxy_quality"] = string(proxy.Quality)
	
	if proxy.Location != nil {
		fields["proxy_country"] = proxy.Location.Country
	}

	sl.Info(fmt.Sprintf("Proxy event: %s", eventType), fields)
}

// ExportLogs exports recent logs to a file
func (sl *StructuredLogger) ExportLogs(filename string, limit int) error {
	logs := sl.GetRecentLogs(limit)
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(map[string]interface{}{
		"exported_at": time.Now(),
		"session_id":  sl.sessionID,
		"total_logs":  len(logs),
		"logs":        logs,
	})
}

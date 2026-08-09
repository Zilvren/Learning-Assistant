package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct {
	info  *log.Logger
	error *log.Logger
	audit *log.Logger
}

type AuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	ClientIP   string    `json:"client_ip,omitempty"`
	UserID     int64     `json:"user_id,omitempty"`
}

// New 在日志层中创建所需对象并完成初始化。
func New() *Logger {
	return &Logger{
		info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		audit: log.New(os.Stdout, "[AUDIT] ", 0),
	}
}

// Infof 在日志层中完成本文件定义的局部处理。
func (l *Logger) Infof(format string, args ...interface{}) {
	l.info.Printf(format, args...)
}

// Errorf 在日志层中完成本文件定义的局部处理。
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.error.Printf(format, args...)
}

// Audit 在日志层中完成本文件定义的局部处理。
func (l *Logger) Audit(entry AuditEntry) {
	if l == nil || l.audit == nil {
		return
	}
	entry.Timestamp = entry.Timestamp.UTC()
	payload, err := json.Marshal(entry)
	if err != nil {
		l.Errorf("failed to encode audit entry: %v", err)
		return
	}
	l.audit.Print(string(payload))
}

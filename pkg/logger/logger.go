package logger

import (
	"log"
	"os"
)

type Logger struct {
	info  *log.Logger
	error *log.Logger
}

func New() *Logger {
	return &Logger{
		info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.info.Printf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.error.Printf(format, args...)
}

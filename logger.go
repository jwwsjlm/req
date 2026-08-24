package req

import (
	"io"
	"log"
	"os"
)

// Logger is the abstract logging interface, gives control to
// the Req users, choice of the logger.
type Logger interface {
	// Errorf logs a formatted error message.
	// Errorf 记录格式化的错误消息。
	Errorf(format string, v ...any)
	// Warnf logs a formatted warning message.
	// Warnf 记录格式化的警告消息。
	Warnf(format string, v ...any)
	// Debugf logs a formatted debug message.
	// Debugf 记录格式化的调试消息。
	Debugf(format string, v ...any)
}

// NewLogger creates a Logger backed by a new standard-library log.Logger.
// NewLogger 使用给定输出、前缀和标志创建由标准库 log.Logger 支持的 Logger。
func NewLogger(output io.Writer, prefix string, flag int) Logger {
	return &logger{l: log.New(output, prefix, flag)}
}

// NewLoggerFromStandardLogger wraps an existing standard-library log.Logger.
// NewLoggerFromStandardLogger 将现有的标准库 log.Logger 包装为 Logger。
func NewLoggerFromStandardLogger(l *log.Logger) Logger {
	return &logger{l: l}
}

func createDefaultLogger() Logger {
	return NewLogger(os.Stdout, "", log.Ldate|log.Lmicroseconds)
}

var _ Logger = (*logger)(nil)

type disableLogger struct{}

func (l *disableLogger) Errorf(format string, v ...any) {}
func (l *disableLogger) Warnf(format string, v ...any)  {}
func (l *disableLogger) Debugf(format string, v ...any) {}

type logger struct {
	l *log.Logger
}

func (l *logger) Errorf(format string, v ...any) {
	l.output("ERROR", format, v...)
}

func (l *logger) Warnf(format string, v ...any) {
	l.output("WARN", format, v...)
}

func (l *logger) Debugf(format string, v ...any) {
	l.output("DEBUG", format, v...)
}

func (l *logger) output(level, format string, v ...any) {
	format = level + " [req] " + format
	if len(v) == 0 {
		l.l.Print(format)
		return
	}
	l.l.Printf(format, v...)
}

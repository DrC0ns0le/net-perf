package logging

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

var (
	loggingLevel = flag.String("logging.level", "info", "Enable debug logging")
)

type Logger interface {
	Debug(a ...any)
	Debugf(format string, v ...interface{})
	Info(a ...any)
	Infof(format string, v ...interface{})
	Warn(a ...any)
	Warnf(format string, v ...interface{})
	Error(a ...any)
	Errorf(format string, v ...interface{})
	Fatal(a ...any)
	Fatalf(format string, v ...interface{})
	With(args ...any) Logger
}

type defaultLogger struct {
	logger              *slog.Logger
	addtionalAttributes []any
	offset              int
}

func NewDefaultLogger() Logger {
	programLevel := new(slog.LevelVar)
	level := strings.ToLower(*loggingLevel)

	switch level {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "info":
		programLevel.Set(slog.LevelInfo)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		programLevel.Set(slog.LevelInfo)
		fmt.Fprintf(os.Stderr, "invalid logging level %q, defaulting to info\n", *loggingLevel)
	}

	return &defaultLogger{
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: programLevel,
		})),
		addtionalAttributes: []any{},
		offset:              0,
	}
}

func (l *defaultLogger) With(args ...any) Logger {
	return &defaultLogger{
		logger:              l.logger.With(args...),
		addtionalAttributes: append(l.addtionalAttributes, args...),
		offset:              l.offset + 0,
	}
}

// getCaller returns the calling function's package, file, and line number
func getCaller(offset int) (string, string, int) {
	pc, file, line, ok := runtime.Caller(3 + offset) // Skip four frames to get the actual caller
	if !ok {
		return "unknown", "unknown", 0
	}

	// Get the function name
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", file, line
	}

	// Extract package name from full function name
	// Format is "package/path.function"
	fullName := fn.Name()
	if lastDot := strings.LastIndexByte(fullName, '.'); lastDot >= 0 {
		if lastSlash := strings.LastIndexByte(fullName[:lastDot], '/'); lastSlash >= 0 {
			return fullName[lastSlash+1 : lastDot], file, line
		}
		return fullName[:lastDot], file, line
	}
	return fullName, file, line
}

func (l *defaultLogger) additionalArgs() []any {
	pkg, file, line := getCaller(l.offset)
	if strings.ToLower(*loggingLevel) == "debug" {
		return []any{
			"package", pkg,
			"file", file,
			"line", line,
		}
	}
	return []any{
		"package", pkg,
	}
}

func (l *defaultLogger) Info(a ...any) {
	l.logger.Info(fmt.Sprint(a...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Warn(a ...any) {
	l.logger.Warn(fmt.Sprint(a...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, v...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Error(a ...any) {
	l.logger.Error(fmt.Sprint(a...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Debug(a ...any) {
	l.logger.Debug(fmt.Sprint(a...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, v...),
		l.additionalArgs()...,
	)
}

func (l *defaultLogger) Fatal(a ...any) {
	l.logger.Error(fmt.Sprint(a...),
		l.additionalArgs()...,
	)
	os.Exit(1)
}

func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...),
		l.additionalArgs()...,
	)
	os.Exit(1)
}

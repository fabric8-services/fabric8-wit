package log

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"runtime"

	"github.com/almighty/almighty-core/configuration"

	"github.com/goadesign/goa/middleware"
	"golang.org/x/net/context"
)

var (
	logger = NewLogger()
)

func NewLogger() *log.Logger {
	logger := log.New()

	if err := configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	// TODO: This part should rely on a persistence configuration without having
	// to call configuration.Setup(...)
	if os.Getenv("ALMIGHTY_DEVELOPER_MODE_ENABLED") == "true" || configuration.IsPostgresDeveloperModeEnabled() {
		customFormatter := new(log.TextFormatter)
		customFormatter.FullTimestamp = true
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		log.SetFormatter(customFormatter)

		log.SetLevel(log.DebugLevel)
		logger.Level = log.DebugLevel
		logger.Formatter = customFormatter
	} else {
		customFormatter := new(log.JSONFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"

		log.SetFormatter(customFormatter)
		customFormatter.DisableTimestamp = false

		log.SetLevel(log.InfoLevel)
		logger.Level = log.InfoLevel
		logger.Formatter = customFormatter
	}

	logger.Out = os.Stdout

	return logger
}

func NewCustomizedLogger(level string, defaultFormatter bool) (*log.Logger, error) {
	logger := log.New()

	lv, err := log.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.Level = lv

	customFormatter := new(log.TextFormatter)
	if !defaultFormatter {
		customFormatter := new(log.JSONFormatter)
		customFormatter.DisableTimestamp = false
	}
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logger.Formatter = customFormatter

	return logger, nil
}

func Logger() *log.Logger {
	return logger
}

func LogError(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		if pc, file, line, ok := runtime.Caller(1); ok {
			fName := runtime.FuncForPC(pc).Name()
			entry = entry.WithField("file", file).WithField("line", line).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("requestID", middleware.ContextRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Errorf(format, args)
		} else {
			entry.WithFields(fields).Errorln(format)
		}
	}
}

func LogWarn(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.WarnLevel {
		entry := log.NewEntry(logger)
		if pc, file, _, ok := runtime.Caller(1); ok {
			fName := runtime.FuncForPC(pc).Name()
			entry = log.WithField("file", file).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("requestID", middleware.ContextRequestID(ctx))
		}
		if len(args) > 0 {
			entry.WithFields(fields).Warnf(format, args...)
		} else {
			entry.WithFields(fields).Warnln(format)
		}
	}
}

func LogInfo(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.InfoLevel {
		entry := log.NewEntry(logger)

		if ctx != nil {
			entry = entry.WithField("requestID", middleware.ContextRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Infof(format, args...)
		} else {
			entry.WithFields(fields).Infoln(format)
		}
	}
}

func LogPanic(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		if ctx != nil {
			entry = entry.WithField("requestID", middleware.ContextRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Panicf(format, args)
		} else {
			entry.WithFields(fields).Panicln(format)
		}
	}
}

func LogDebug(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.DebugLevel {
		entry := log.NewEntry(logger)

		if ctx != nil {
			entry = entry.WithField("requestID", middleware.ContextRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Debugf(format, args...)
		} else {
			entry.WithFields(fields).Debugln(format)
		}
	}
}

func LoggerRuntimeContext() *log.Entry {
	entry := log.WithField("pid", os.Getpid())

	if pc, file, line, ok := runtime.Caller(1); ok {
		fName := runtime.FuncForPC(pc).Name()
		return entry.WithField("file", file).WithField("line", line).WithField("func", fName)
	} else {
		return entry
	}
}

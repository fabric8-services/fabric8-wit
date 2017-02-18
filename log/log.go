package log

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"os"
	"runtime"

	"github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
	"golang.org/x/net/context"
)

var (
	logger = log.New()
)

func InitializeLogger(developerModeFlag bool) {
	logger = log.New()

	if developerModeFlag {
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

}

func NewCustomizedLogger(level string, developerModeFlag bool) (*log.Logger, error) {
	logger := log.New()

	lv, err := log.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.Level = lv

	if developerModeFlag {
		customFormatter := new(log.TextFormatter)
		customFormatter.FullTimestamp = true
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		log.SetFormatter(customFormatter)

		log.SetLevel(log.DebugLevel)
		logger.Level = lv
		logger.Formatter = customFormatter
	} else {
		customFormatter := new(log.JSONFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"

		log.SetFormatter(customFormatter)
		customFormatter.DisableTimestamp = false

		log.SetLevel(log.InfoLevel)
		logger.Level = lv
		logger.Formatter = customFormatter
	}

	logger.Out = os.Stdout

	return logger, nil
}

func Logger() *log.Logger {
	return logger
}

func Error(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		file, line, fName, err := extractCallerDetails()
		if err != nil {
			entry = entry.WithField("file", file).WithField("line", line).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Errorf(format, args)
		} else {
			entry.WithFields(fields).Errorln(format)
		}
	}
}

func Warn(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.WarnLevel {
		entry := log.NewEntry(logger)

		file, _, fName, err := extractCallerDetails()
		if err != nil {
			entry = log.WithField("file", file).WithField("func", fName)
		}

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Warnf(format, args...)
		} else {
			entry.WithFields(fields).Warnln(format)
		}
	}
}

func Info(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.InfoLevel {
		entry := log.NewEntry(logger)

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Infof(format, args...)
		} else {
			entry.WithFields(fields).Infoln(format)
		}
	}
}

func Panic(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.ErrorLevel {
		entry := log.WithField("pid", os.Getpid())

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Panicf(format, args)
		} else {
			entry.WithFields(fields).Panicln(format)
		}
	}
}

func Debug(ctx context.Context, fields map[string]interface{}, format string, args ...interface{}) {
	if logger.Level >= log.DebugLevel {
		entry := log.NewEntry(logger)

		if ctx != nil {
			entry = entry.WithField("req_id", extractRequestID(ctx))
		}

		if len(args) > 0 {
			entry.WithFields(fields).Debugf(format, args...)
		} else {
			entry.WithFields(fields).Debugln(format)
		}
	}
}

// extractRequestID obtains the request ID either from a goa client or middleware
func extractRequestID(ctx context.Context) string {
	reqID := middleware.ContextRequestID(ctx)
	if reqID == "" {
		return client.ContextRequestID(ctx)
	}

	return reqID
}

func extractCallerDetails() (string, int, string, error) {
	if pc, file, line, ok := runtime.Caller(2); ok {
		fName := runtime.FuncForPC(pc).Name()
		return file, line, fName, nil
	}

	return "", 0, "", errors.New("Unable to extract the caller details")
}

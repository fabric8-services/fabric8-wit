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

// InitializeLogger creates a default logger whose ouput format, log level differs
// depending of whether the developer mode flag is enable/disabled.
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

// NewCustomizedLogger creates a custom logger specifying the desired log level
// and the developer mode flag. Returns the logger object and the error.
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

// Logger returns the current logger object.
func Logger() *log.Logger {
	return logger
}

// Error logs an error message that might contain the following attributes: pid,
// request id if provided by the context, file location of the caller, line that
// called the log Error function and the function name. Moreover, we can use the
// parameter fields to add additional attributes to the output message. Likewise
// format and args are used to print a detailed message with the reasons of the
// error log.
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

// Warn logs a warning message that might contain the following attributes:
// request id if provided by the context, the file and the
// function name that invoked the Warn() function. In this function, we can use
// the parameter fields to add additional attributes to the output of this
// message. Likewise format and args are used to print a detailed message with
// the reasons of the warning log.
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

// Info logs an info message that might contain the request id if provided by
// the context. In this function, the parameter fields enables to additional
// attributes to the message. The format and args input arguments are used to
// print a detailed information about the reasons of this log.
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

// Panic logs a panic message that might contain the following attributes:
// the request id if provided by the context and the pid. In this function, the
// parameter fields enables to additional attributes to the message. The format
// and args input arguments are used to print a detailed information about the
// reasons of this log.
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

// Debug logs a debug message that might specifies the request id if provided by
// the context. In this function, the parameter fields enables to additional
// attributes to the message. The format and args input arguments are used to
// print a detailed information about the reasons of this log.
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

// extractCallerDetails gets information about the file, line and function that
// called a certain logging method such as Error, Info, Debug, Warn and Panic.
func extractCallerDetails() (string, int, string, error) {
	if pc, file, line, ok := runtime.Caller(2); ok {
		fName := runtime.FuncForPC(pc).Name()
		return file, line, fName, nil
	}

	return "", 0, "", errors.New("Unable to extract the caller details")
}

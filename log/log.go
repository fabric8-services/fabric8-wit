package log

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"runtime"
)

var (
	logger = NewLogger()
)

func init() {
	customFormatter := new(log.JSONFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.DisableTimestamp = false

	log.SetOutput(os.Stdout)

	log.SetLevel(log.WarnLevel)
}

func NewLogger() *log.Logger {
	logger := log.New()
	customFormatter := new(log.JSONFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.DisableTimestamp = false

	logger.Out = os.Stdout
	logger.Formatter = customFormatter

	logger.Level = log.InfoLevel

	return logger
}

func Logger() *log.Logger {
	return logger
}

func SetFields(fields map[string]interface{}) log.Fields {
	f := log.Fields{}
	f = fields
	return f
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

func NewCustomizedLogger(level string, defaultFormatter bool) (*log.Logger, error) {
	logger := log.New()

	lv, err := log.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	log.SetLevel(lv)

	customFormatter := new(log.TextFormatter)
	if !defaultFormatter {
		customFormatter := new(log.JSONFormatter)
		customFormatter.DisableTimestamp = false
	}
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	return logger, nil
}

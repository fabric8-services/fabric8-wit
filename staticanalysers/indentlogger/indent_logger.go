package indentlogger

import (
	"io"
	"log"
	"strings"
)

// IndentLogger is just a normal log.Logger but with the option to indent and
// outdent the output after the original prefix of the embedded logger.
type IndentLogger struct {
	*log.Logger

	originalPrefix string
	indentation    []string
	indentBy       string
}

// New creates a logger that can indent and outdent its output.
func New(out io.Writer, prefix string, flag int, indentBy string) *IndentLogger {
	l := &IndentLogger{
		Logger:         log.New(out, prefix, flag),
		originalPrefix: prefix,
		indentation:    make([]string, 0, 100),
		indentBy:       indentBy,
	}
	return l
}

// Indent indents the log output after the original prefix with a space.
func (l *IndentLogger) Indent() {
	l.indentation = append(l.indentation, l.indentBy)
	l.Logger.SetPrefix(l.originalPrefix + strings.Join(l.indentation, ""))
}

// Outdent outdents the log output after the original by one space.
func (l *IndentLogger) Outdent() {
	l.indentation = l.indentation[1:]
	l.Logger.SetPrefix(l.originalPrefix + strings.Join(l.indentation, ""))
}

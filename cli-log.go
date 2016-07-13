package clilog

import "fmt"
import "io"
import "os"

// Log levels go in order: Debug, Info, Warn, Error, Fatal
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var logLevel = LevelInfo
var outWriter io.Writer = os.Stdout
var errWriter io.Writer = os.Stderr

// SetLogLevel allows you to set the level to log, by default LevelInfo is set
func SetLogLevel(level int) {
	logLevel = level
}

// SetOutputWriter allows you to set the output file for debug, info, and warn messges
func SetOutputWriter(w io.Writer) {
	outWriter = w
}

// SetErrorWriter allows you to set the output writer for error and fatal messages
func SetErrorWriter(w io.Writer) {
	errWriter = w
}

// Debug logs a debug level message
func Debug(a ...interface{}) {
	logMessage(LevelDebug, outWriter, fmt.Sprint(a...))
}

// Info logs an info level message
func Info(a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprint(a...))
}

// Warn logs a warning level message
func Warn(a ...interface{}) {
	logMessage(LevelWarn, outWriter, fmt.Sprint(a...))
}

// Error logs an error level message
func Error(a ...interface{}) {
	logMessage(LevelError, errWriter, fmt.Sprint(a...))
}

// Fatal logs a fatal level message
func Fatal(a ...interface{}) {
	logMessage(LevelFatal, errWriter, fmt.Sprint(a...))
}

// Debugf logs a formatted debug level message
func Debugf(format string, a ...interface{}) {
	logMessage(LevelDebug, outWriter, fmt.Sprintf(format, a...))
}

// Infof logs a formatted info level message
func Infof(format string, a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprintf(format, a...))
}

// Warnf logs a formatted warning level message
func Warnf(format string, a ...interface{}) {
	logMessage(LevelWarn, outWriter, fmt.Sprintf(format, a...))
}

// Errorf logs a formatted error level message
func Errorf(format string, a ...interface{}) {
	logMessage(LevelError, errWriter, fmt.Sprintf(format, a...))
}

// Fatalf logs a formatted fatal level message
func Fatalf(format string, a ...interface{}) {
	logMessage(LevelFatal, errWriter, fmt.Sprintf(format, a...))
}

// Debugln logs a debug level message with a newline appended
func Debugln(a ...interface{}) {
	logMessage(LevelDebug, outWriter, fmt.Sprintln(a...))
}

// Infoln logs an info level message with a newline appended
func Infoln(a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprintln(a...))
}

// Warnln logs a warning level message with a newline appended
func Warnln(a ...interface{}) {
	logMessage(LevelWarn, outWriter, fmt.Sprintln(a...))
}

// Errorln logs an error level message with a newline appended
func Errorln(a ...interface{}) {
	logMessage(LevelError, errWriter, fmt.Sprintln(a...))
}

// Fatalln logs a fatal level message with a newline appended
func Fatalln(a ...interface{}) {
	logMessage(LevelFatal, errWriter, fmt.Sprintln(a...))
}

func logMessage(level int, writer io.Writer, message string) {
	if level < logLevel {
		return
	}
	fmt.Fprint(writer, message)
}

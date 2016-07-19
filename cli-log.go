package clilog

import "fmt"
import "io"
import "os"
import "github.com/fatih/color"

// SprintfYellow creates a yellow formatted string
var SprintfYellow = color.New(color.FgHiYellow).SprintfFunc()

// SprintfGreen creates a green formatted string
var SprintfGreen = color.New(color.FgHiGreen).SprintfFunc()

// SprintfBlue creates a blue formatted string
var SprintfBlue = color.New(color.FgHiBlue).SprintfFunc()

// SprintfRed creates a red formatted string
var SprintfRed = color.New(color.FgHiRed).SprintfFunc()

// Yellow creates a yellow string
var Yellow = SprintfYellow

// Green creates a green string
var Green = SprintfGreen

// Blue creates a blue string
var Blue = SprintfBlue

// Red creates a red string
var Red = SprintfRed

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

// SetColor sets the color status. True for color, False for no color
func SetColor(colorOn bool) {
	color.NoColor = !colorOn
}

// Debug logs a formatted debug level message with a newline appended
func Debug(format string, a ...interface{}) {
	logMessage(LevelDebug, outWriter, fmt.Sprintln(SprintfBlue(format, a...)))
}

// Info logs a formatted info level message with a newline appended
func Info(format string, a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprintln(fmt.Sprintf(format, a...)))
}

// Warn logs a formatted warning level message with a newline appended
func Warn(format string, a ...interface{}) {
	logMessage(LevelWarn, outWriter, fmt.Sprintln(SprintfYellow(format, a...)))
}

// Error logs a formatted error level message with a newline appended
func Error(format string, a ...interface{}) {
	logMessage(LevelError, errWriter, fmt.Sprintln(SprintfRed(format, a...)))
}

// Fatal logs a formatted fatal level message with a newline appended and calls os.Exit(1)
func Fatal(format string, a ...interface{}) {
	logMessage(LevelFatal, errWriter, fmt.Sprintln(SprintfRed(format, a...)))
	os.Exit(1)
}

// Debugf logs a formatted debug level message
func Debugf(format string, a ...interface{}) {
	logMessage(LevelDebug, outWriter, SprintfBlue(format, a...))
}

// Infof logs a formatted info level message
func Infof(format string, a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprintf(format, a...))
}

// Warnf logs a formatted warning level message
func Warnf(format string, a ...interface{}) {
	logMessage(LevelWarn, outWriter, SprintfYellow(format, a...))
}

// Errorf logs a formatted error level message
func Errorf(format string, a ...interface{}) {
	logMessage(LevelError, errWriter, SprintfRed(format, a...))
}

// Fatalf logs a formatted fatal level message and calls os.Exit(1)
func Fatalf(format string, a ...interface{}) {
	logMessage(LevelFatal, errWriter, SprintfRed(format, a...))
	os.Exit(1)
}

// Debugln logs a debug level message with a newline appended
func Debugln(a ...interface{}) {
	logMessage(LevelDebug, outWriter, SprintfBlue(fmt.Sprintln(a...)))
}

// Infoln logs an info level message with a newline appended
func Infoln(a ...interface{}) {
	logMessage(LevelInfo, outWriter, fmt.Sprintln(a...))
}

// Warnln logs a warning level message with a newline appended
func Warnln(a ...interface{}) {
	logMessage(LevelWarn, outWriter, SprintfYellow(fmt.Sprintln(a...)))
}

// Errorln logs an error level message with a newline appended
func Errorln(a ...interface{}) {
	logMessage(LevelError, errWriter, SprintfRed(fmt.Sprintln(a...)))
}

// Fatalln logs a fatal level message with a newline appended and calls os.Exit(1)
func Fatalln(a ...interface{}) {
	logMessage(LevelFatal, errWriter, SprintfRed(fmt.Sprintln(a...)))
	os.Exit(1)
}

func logMessage(level int, writer io.Writer, message string) {
	if level < logLevel {
		return
	}
	fmt.Fprint(writer, message)
}

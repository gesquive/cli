package cli

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

// Print levels go in order: Debug, Info, Warn, Error, Fatal
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var printLevel = LevelInfo
var outWriter io.Writer = os.Stdout
var errWriter io.Writer = os.Stderr

// SetPrintLevel allows you to set the level to print, by default LevelInfo is set
func SetPrintLevel(level int) {
	printLevel = level
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

// Debug prints a formatted debug level message with a newline appended
func Debug(format string, a ...interface{}) {
	printMessage(LevelDebug, outWriter, fmt.Sprintln(SprintfBlue(format, a...)))
}

// Info prints a formatted info level message with a newline appended
func Info(format string, a ...interface{}) {
	printMessage(LevelInfo, outWriter, fmt.Sprintln(fmt.Sprintf(format, a...)))
}

// Warn prints a formatted warning level message with a newline appended
func Warn(format string, a ...interface{}) {
	printMessage(LevelWarn, outWriter, fmt.Sprintln(SprintfYellow(format, a...)))
}

// Error prints a formatted error level message with a newline appended
func Error(format string, a ...interface{}) {
	printMessage(LevelError, errWriter, fmt.Sprintln(SprintfRed(format, a...)))
}

// Fatal prints a formatted fatal level message with a newline appended and calls os.Exit(1)
func Fatal(format string, a ...interface{}) {
	printMessage(LevelFatal, errWriter, fmt.Sprintln(SprintfRed(format, a...)))
	os.Exit(1)
}

// Debugf prints a formatted debug level message
func Debugf(format string, a ...interface{}) {
	printMessage(LevelDebug, outWriter, SprintfBlue(format, a...))
}

// Infof prints a formatted info level message
func Infof(format string, a ...interface{}) {
	printMessage(LevelInfo, outWriter, fmt.Sprintf(format, a...))
}

// Warnf prints a formatted warning level message
func Warnf(format string, a ...interface{}) {
	printMessage(LevelWarn, outWriter, SprintfYellow(format, a...))
}

// Errorf prints a formatted error level message
func Errorf(format string, a ...interface{}) {
	printMessage(LevelError, errWriter, SprintfRed(format, a...))
}

// Fatalf prints a formatted fatal level message and calls os.Exit(1)
func Fatalf(format string, a ...interface{}) {
	printMessage(LevelFatal, errWriter, SprintfRed(format, a...))
	os.Exit(1)
}

// Debugln prints a debug level message with a newline appended
func Debugln(a ...interface{}) {
	printMessage(LevelDebug, outWriter, SprintfBlue(fmt.Sprintln(a...)))
}

// Infoln prints an info level message with a newline appended
func Infoln(a ...interface{}) {
	printMessage(LevelInfo, outWriter, fmt.Sprintln(a...))
}

// Warnln prints a warning level message with a newline appended
func Warnln(a ...interface{}) {
	printMessage(LevelWarn, outWriter, SprintfYellow(fmt.Sprintln(a...)))
}

// Errorln prints an error level message with a newline appended
func Errorln(a ...interface{}) {
	printMessage(LevelError, errWriter, SprintfRed(fmt.Sprintln(a...)))
}

// Fatalln prints a fatal level message with a newline appended and calls os.Exit(1)
func Fatalln(a ...interface{}) {
	printMessage(LevelFatal, errWriter, SprintfRed(fmt.Sprintln(a...)))
	os.Exit(1)
}

func printMessage(level int, writer io.Writer, message string) {
	if level < printLevel {
		return
	}
	fmt.Fprint(writer, message)
}

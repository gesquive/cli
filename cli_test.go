package cli

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorPrint(t *testing.T) {
	SetPrintLevel(LevelError)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(true)

	Error("error")

	assert.NotEqual(t, "error\n", stdErr.String(), "Error is incorrect")
	assert.True(t, strings.Contains(stdErr.String(), "error"), "Error is not found in output")
	assert.True(t, len(stdErr.String()) > 6, "Error length is incorrect")

}

func TestDebugPrintLevel(t *testing.T) {
	SetPrintLevel(LevelDebug)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	assert.Equal(t, "debug\ninfo\nwarn\n", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
}

func TestInfoPrintLevel(t *testing.T) {
	SetPrintLevel(LevelInfo)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	assert.Equal(t, "info\nwarn\n", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
}

func TestWarnPrintLevel(t *testing.T) {
	SetPrintLevel(LevelWarn)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	assert.Equal(t, "warn\n", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
}

func TestErrorPrintLevel(t *testing.T) {
	SetPrintLevel(LevelError)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	assert.Equal(t, "", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
}

func TestFatalPrintLevel(t *testing.T) {
	SetPrintLevel(LevelError)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	assert.Equal(t, "", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
	if os.Getenv("TEST_FATAL") == "1" {
		Fatal("fatal")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalPrintLevel")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")
	err := cmd.Run()
	assert.Error(t, err, "Fatal did not exit")
	assert.IsType(t, &exec.ExitError{}, err, "Unexptected error")
	assert.False(t, err.(*exec.ExitError).Success(), "Fatal exited successfully")
}

func TestDebugfPrintLevel(t *testing.T) {
	SetPrintLevel(LevelDebug)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debugf("debug ")
	Infof("info ")
	Warnf("warn ")
	Errorf("error")

	assert.Equal(t, "debug info warn ", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error", stdErr.String(), "Error is incorrect")
}

func TestDebuglPrintLevel(t *testing.T) {
	SetPrintLevel(LevelDebug)
	stdOut := new(bytes.Buffer)
	SetOutputWriter(stdOut)
	stdErr := new(bytes.Buffer)
	SetErrorWriter(stdErr)
	SetColor(false)

	Debugln("debug")
	Infoln("info")
	Warnln("warn")
	Errorln("error")

	assert.Equal(t, "debug\ninfo\nwarn\n", stdOut.String(), "Output is incorrect")
	assert.Equal(t, "error\n", stdErr.String(), "Error is incorrect")
}

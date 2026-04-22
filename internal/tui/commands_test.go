package tui

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOpenURLCommandWindowsUsesEmptyStartTitle(t *testing.T) {
	origExecCommand := openURLExecCommand
	defer func() { openURLExecCommand = origExecCommand }()

	var gotName string
	var gotArgs []string
	openURLExecCommand = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return exec.Command("sh", "-c", "exit 0")
	}

	cmd := buildOpenURLCommand("windows", "https://example.com")
	require.NotNil(t, cmd)
	require.Equal(t, "cmd", gotName)
	require.Equal(t, []string{"/c", "start", "", "https://example.com"}, gotArgs)
}

func TestOpenURLCmdReturnsLauncherError(t *testing.T) {
	origGOOS := openURLGOOS
	origExecCommand := openURLExecCommand
	origStart := openURLStart
	defer func() {
		openURLGOOS = origGOOS
		openURLExecCommand = origExecCommand
		openURLStart = origStart
	}()

	openURLGOOS = "linux"
	openURLExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	openURLStart = func(cmd *exec.Cmd) error {
		require.NotNil(t, cmd)
		return errors.New("launcher failed")
	}

	msg := openURLCmd("https://example.com")().(openURLMsg)
	require.Equal(t, "https://example.com", msg.url)
	require.EqualError(t, msg.err, "launcher failed")
}

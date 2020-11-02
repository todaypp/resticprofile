package lock

import (
	"bytes"
	"os"
	"os/signal"
	"runtime"
	"testing"

	"github.com/creativeprojects/resticprofile/shell"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
)

// These tests are using the shell package. This is just a convenient wrapper around cmd.exe and sh

func TestProcessFinished(t *testing.T) {
	childPID := 0
	buffer := &bytes.Buffer{}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Reset(os.Interrupt)

	cmd := shell.NewSignalledCommand("echo", []string{"Hello World!"}, c)
	cmd.Stdout = buffer
	cmd.SetPID = func(pid int) {
		childPID = pid
	}
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	// at that point, the child process should be finished
	running, err := process.PidExists(int32(childPID))
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestProcessNotFinished(t *testing.T) {
	childPID := 0
	buffer := &bytes.Buffer{}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Reset(os.Interrupt)

	// use ping to make sure the process is running for long enough to check its existence
	var parameters []string
	if runtime.GOOS == "windows" {
		// it will run for 1 second
		parameters = []string{"-n", "2", "127.0.0.1"}
	} else {
		// run for 200ms (don't need a whole second)
		// 0.2 is the minimum in linux, 0.1 in darwin
		parameters = []string{"-c", "2", "-i", "0.2", "127.0.0.1"}
	}

	cmd := shell.NewSignalledCommand("ping", parameters, c)
	cmd.Stdout = buffer
	// SetPID method is called right after we forked and have a PID available
	cmd.SetPID = func(pid int) {
		childPID = pid
		running, err := process.PidExists(int32(childPID))
		assert.NoError(t, err)
		assert.True(t, running)
	}
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	// at that point, the child process should be finished
	running, err := PidExists(childPID)
	assert.NoError(t, err)
	assert.False(t, running)
}

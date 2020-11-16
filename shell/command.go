package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SetPID is a callback to send the PID of the current child process
type SetPID func(pid int)

// Command holds the configuration to run a shell command
type Command struct {
	Command               string
	Arguments             []string
	Environ               []string
	Dir                   string
	Stdin                 io.Reader
	Stdout                io.Writer
	Stderr                io.Writer
	SetPID                SetPID
	PropagateSignalScript bool
	sigChan               chan os.Signal
	done                  chan interface{}
}

// NewCommand instantiate a default Command without receiving OS signals (SIGTERM, etc.)
func NewCommand(command string, args []string) *Command {
	return &Command{
		Command:   command,
		Arguments: args,
		Environ:   []string{},
	}
}

// NewSignalledCommand instantiate a default Command receiving OS signals (SIGTERM, etc.)
func NewSignalledCommand(command string, args []string, c chan os.Signal) *Command {
	return &Command{
		Command:   command,
		Arguments: args,
		Environ:   []string{},
		sigChan:   c,
		done:      make(chan interface{}),
	}
}

// Run the command
func (c *Command) Run() error {
	var err error

	command, args, err := c.getShellCommand()
	if err != nil {
		return err
	}

	cmd := exec.Command(command, args...)

	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin

	cmd.Env = os.Environ()
	if c.Environ != nil && len(c.Environ) > 0 {
		cmd.Env = append(cmd.Env, c.Environ...)
	}

	// spawn the child process
	if err = cmd.Start(); err != nil {
		return err
	}
	if c.SetPID != nil {
		// send the PID back (to write down in a lockfile)
		c.SetPID(cmd.Process.Pid)
	}
	// setup the OS signalling if we need it (typically used for unixes but not windows)
	if c.sigChan != nil {
		defer func() {
			close(c.done)
		}()
		go c.propagateSignal(cmd.Process)
	}
	return cmd.Wait()
}

// getShellCommand transforms the command line and arguments to be launched via a shell (sh or cmd.exe)
func (c *Command) getShellCommand() (string, []string, error) {
	return getShellCommand(c.Command, c.Arguments, c.PropagateSignalScript)
}

// getShellCommand transforms the command line and arguments to be launched via a shell (sh or cmd.exe)
func getShellCommand(command string, args []string, propagateSignalScript bool) (string, []string, error) {

	if runtime.GOOS == "windows" {
		shell, err := exec.LookPath("cmd.exe")
		if err != nil {
			return "", nil, fmt.Errorf("cannot find shell executable (cmd.exe) in path")
		}
		// cmd.exe accepts that all arguments are sent one by one
		args := append([]string{"/C", command}, removeQuotes(args)...)
		return shell, args, nil
	}

	shell, err := exec.LookPath("sh")
	if err != nil {
		return "", nil, fmt.Errorf("cannot find shell executable (sh) in path")
	}

	if propagateSignalScript {
		return getSignalPropagationShellScript(shell, command, args)
	}
	// Flatten all arguments into one string, sh expects one big string
	flatCommand := append([]string{command}, args...)
	return shell, []string{"-c", strings.Join(flatCommand, " ")}, nil
}

// removeQuotes removes single and double quotes when the whole string is quoted
func removeQuotes(args []string) []string {
	if args == nil {
		return nil
	}

	singleQuote := `'`
	doubleQuote := `"`

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], doubleQuote) && strings.HasSuffix(args[i], doubleQuote) {
			args[i] = strings.Trim(args[i], doubleQuote)

		} else if strings.HasPrefix(args[i], singleQuote) && strings.HasSuffix(args[i], singleQuote) {
			args[i] = strings.Trim(args[i], singleQuote)
		}
	}
	return args
}

// getSignalPropagationShellScript returns a shell script to make sure all signals are sent to
// the commands started by the shell.
// See http://veithen.io/2014/11/16/sigterm-propagation.html
func getSignalPropagationShellScript(shell, command string, args []string) (string, []string, error) {
	template := `
trap 'kill -TERM $PID' TERM INT
%s %s &
PID=$!
wait $PID
trap - TERM INT
wait $PID
EXIT_STATUS=$?
`
	return shell, []string{"-c", fmt.Sprintf(template, command, strings.Join(args, " "))}, nil
}

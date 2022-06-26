// Package shelley runs commands with behavior similar to a command line shell.
package shelley

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
)

// ExitError is the type of error returned by commands that completed with a
// non-zero exit code.
type ExitError = *exec.ExitError

// ExitIfError exits the current process with a non-zero code if err is non-nil.
//
// If err is an ExitError, the process will exit silently with the same code as
// the command that generated the error. Otherwise, the error will be logged
// with the log package and the process will exit with code 1.
//
// This enables an extremely limited but easy to use form of error handling,
// roughly analogous to "set -e" in a shell script, but without the complex
// rules and exceptions that many "set -e" users (particularly this author) do
// not actually understand.
func ExitIfError(err error) {
	if err == nil {
		return
	}

	var exitErr ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	log.Fatal(err)
}

// DefaultContext is the Context for commands created by the top level Command
// function.
var DefaultContext = &Context{
	Stdin:       os.Stdin,
	Stdout:      os.Stdout,
	Stderr:      os.Stderr,
	DebugLogger: nil,
}

// Context provides default settings that affect the execution of commands.
type Context struct {
	// Stdin is the default source for stdin.
	Stdin io.Reader
	// Stdout is the default destination for stdout.
	Stdout io.Writer
	// Stderr is the default destination for stderr.
	Stderr io.Writer
	// DebugLogger logs all commands as they are executed, approximating the
	// behavior of "set -x" in a shell. Debug lines include environment variables
	// along with the exact arguments that a command was built with, with shell
	// quoting for all values. Aliases are not expanded.
	DebugLogger *log.Logger
}

// Command initializes a new command that will run with the provided arguments.
//
// The first argument is the name of the command to be run. If it contains no
// path separators, it will be resolved to a complete name using a PATH lookup.
func (c *Context) Command(args ...string) *Cmd {
	return &Cmd{context: c, args: args}
}

// Cmd represents a runnable command.
type Cmd struct {
	context *Context
	cmd     *exec.Cmd

	args  []string
	envs  []string
	stdin io.Reader
}

// Command initializes a new command using DefaultContext.
func Command(args ...string) *Cmd {
	return DefaultContext.Command(args...)
}

// Stdin overrides the command's stdin to come from the provided reader, rather
// than the context's stdin.
func (c *Cmd) Stdin(r io.Reader) *Cmd {
	c.stdin = r
	return c
}

// Env appends an environment value to the command.
//
// The appended value overrides any value inherited from the current process or
// set by a previous Env call.
func (c *Cmd) Env(name, value string) *Cmd {
	c.envs = append(c.envs, name+"="+value)
	return c
}

// Run runs the command and waits for it to complete.
func (c *Cmd) Run() error {
	if c.context.DebugLogger != nil {
		var envString string
		for _, env := range c.envs {
			split := strings.SplitN(env, "=", 2)
			envString += split[0] + "=" + shellquote.Join(split[1]) + " "
		}
		c.context.DebugLogger.Print(envString + shellquote.Join(c.args...))
	}

	c.cmd = exec.Command(c.args[0], c.args[1:]...)
	c.cmd.Env = append(os.Environ(), c.envs...)
	c.cmd.Stdout = c.context.Stdout
	c.cmd.Stderr = c.context.Stderr

	c.cmd.Stdin = c.context.Stdin
	if c.stdin != nil {
		c.cmd.Stdin = c.stdin
	}

	return c.cmd.Run()
}

// Test runs the command, waits for it to complete, and returns whether it
// exited with a status code of 0. It returns a non-nil error only if the
// command failed to start, not if it finished with a non-zero status.
func (c *Cmd) Test() (bool, error) {
	err := c.Run()
	if err == nil {
		return true, nil
	}

	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, err
}

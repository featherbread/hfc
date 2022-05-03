// Package shelley runs commands with behavior similar to a command line shell.
package shelley

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
	"golang.org/x/exp/slices"
)

// ExitError is the type of error returned by commands that completed with a
// non-zero exit code.
type ExitError struct {
	*exec.ExitError

	// Args holds the original arguments of the command that generated this
	// ExitError.
	Args []string
}

func (e ExitError) Unwrap() error {
	return e.ExitError
}

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

// GetOrExit returns result after checking err with ExitIfError.
func GetOrExit[T any](result T, err error) T {
	ExitIfError(err)
	return result
}

// DefaultContext is the Context for commands created by the top level Command
// function.
var DefaultContext = &Context{
	Stdin:       os.Stdin,
	Stdout:      os.Stdout,
	Stderr:      os.Stderr,
	Aliases:     make(map[string][]string),
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
	// Aliases is a mapping from alias names to expanded arguments. When a command
	// is built whose first argument matches a defined alias, the alias will be
	// replaced with the associated arguments before executing the command.
	Aliases map[string][]string
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

	sibling  *Cmd
	args     []string
	envs     []string
	nostdout bool
	nostderr bool
}

// Command initializes a new command using DefaultContext.
func Command(args ...string) *Cmd {
	return DefaultContext.Command(args...)
}

// Pipe initializes a new command whose stdin will be connected to the stdout of
// the original command.
//
// The new command will start its sibling when run, but will not inherit other
// settings (e.g. environment) from it. If multiple commands in a pipeline
// should have these settings, they must be specified for each command in the
// pipeline.
//
// Piped commands approximate the behavior of "set -o pipefail" in a shell.
// That is, if this command does not produce an error but its sibling does, this
// command will take on the sibling's error. Unlike with "set -o pipefail", it
// is possible to determine which command in a pipeline failed with a non-zero
// status by unwrapping the error as an ExitError and reading the contained
// Args. This partially works around the usual limitation of this setting in a
// real shell.
func (c *Cmd) Pipe(args ...string) *Cmd {
	return &Cmd{
		context: c.context,
		sibling: c,
		args:    args,
	}
}

// Env appends an environment value to the command.
//
// The appended value overrides any value inherited from the current process or
// set by a previous Env call.
func (c *Cmd) Env(name, value string) *Cmd {
	c.envs = append(c.envs, name+"="+value)
	return c
}

// NoStdout suppresses the command writing its stdout stream to the context's
// stderr.
func (c *Cmd) NoStdout() *Cmd {
	c.nostdout = true
	return c
}

// NoStderr suppresses the command writing its stderr stream to the context's
// stderr.
func (c *Cmd) NoStderr() *Cmd {
	c.nostderr = true
	return c
}

// NoOutput combines NoStdout and NoStderr.
func (c *Cmd) NoOutput() *Cmd {
	return c.NoStdout().NoStderr()
}

// Run runs the command and waits for it to complete.
func (c *Cmd) Run() error {
	c.logDebug()
	c.initCmd()
	return c.run()
}

// Text runs the command, waits for it to complete, and returns its standard
// output as a string with whitespace trimmed from both ends. This overrides the
// NoStdout setting as well as the Stdout of the command's Context, and captures
// the command's output exclusively into an in-memory buffer.
func (c *Cmd) Text() (string, error) {
	c.logDebug()
	c.initCmd()

	var stdout bytes.Buffer
	c.cmd.Stdout = &stdout

	err := c.run()
	return strings.TrimSpace(stdout.String()), err
}

// Successful runs the command, waits for it to complete, and returns whether it
// exited with a status code of 0. It returns a non-nil error only if the
// command failed to start, not if it finished with a non-zero status.
func (c *Cmd) Successful() (bool, error) {
	c.logDebug()
	c.initCmd()

	err := c.run()
	if err == nil {
		return true, nil
	}

	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, err
}

func (c *Cmd) initCmd() {
	args := c.expandedArgs()
	c.cmd = exec.Command(args[0], args[1:]...)
	c.cmd.Env = append(os.Environ(), c.envs...)
}

func (c *Cmd) expandedArgs() []string {
	if alias, ok := c.context.Aliases[c.args[0]]; ok {
		return append(slices.Clone(alias), c.args[1:]...)
	}
	return c.args
}

func (c *Cmd) run() error {
	if c.cmd.Stdin == nil {
		c.cmd.Stdin = c.context.Stdin
	}
	if c.cmd.Stdout == nil && !c.nostdout {
		c.cmd.Stdout = c.context.Stdout
	}
	if c.cmd.Stderr == nil && !c.nostderr {
		c.cmd.Stderr = c.context.Stderr
	}

	siblingErr, err := c.startSibling()
	if err != nil {
		return err
	}

	err = c.cmd.Run()

	if serr := <-siblingErr; serr != nil && err == nil {
		return serr
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ExitError{exitErr, c.args}
	}
	return err
}

func (c *Cmd) startSibling() (siblingErr chan error, err error) {
	siblingErr = make(chan error, 1)
	if c.sibling == nil {
		close(siblingErr)
		return siblingErr, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		close(siblingErr)
		return siblingErr, err
	}

	c.sibling.initCmd()
	c.sibling.cmd.Stdout = pw
	c.cmd.Stdin = pr

	go func() {
		// We need to clean up our references to both ends of the pipe, but only
		// after we have started the processes and allowed them to duplicate those
		// references. We especially have to close our write side, otherwise the
		// child will never get the EOF from the read side even after the sibling is
		// done writing.
		//
		// In theory we can close the appropriate side of the pipe right after each
		// process starts, but it's easier to implement things this way. If shelley
		// is hitting open file limits or something because of this behavior, it
		// might be time to reconsider whether shelley is the right solution.
		defer pr.Close()
		defer pw.Close()
		defer close(siblingErr)
		siblingErr <- c.sibling.run()
	}()

	return siblingErr, nil
}

func (c *Cmd) logDebug() {
	if c.context.DebugLogger == nil {
		return
	}
	if c.sibling != nil {
		c.sibling.logDebug()
	}

	var envString string
	for _, env := range c.envs {
		split := strings.SplitN(env, "=", 2)
		envString += split[0] + "=" + shellquote.Join(split[1]) + " "
	}

	c.context.DebugLogger.Print(envString + shellquote.Join(c.args...))
}

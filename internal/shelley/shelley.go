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
// rules and exceptions that many "set -e" users do not actually understand.
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

// GetOrExit calls ExitIfError if err is non-nil, and otherwise returns result.
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
	DebugLogger: log.Default(),
}

// Context provides default settings that affect the execution of commands.
type Context struct {
	// Stdin is the default source of stdin for commands.
	Stdin io.Reader
	// Stdout is the default destination for the stdout of commands.
	Stdout io.Writer
	// Stderr is the default destination for the stderr of commands.
	Stderr io.Writer
	// Aliases is a mapping from alias names to expanded arguments. When a command
	// is built whose first argument matches a defined alias, the alias will be
	// replaced with the associated arguments before executing the command.
	Aliases map[string][]string
	// DebugLogger is the logger that receives debug lines written by Cmd.Debug.
	DebugLogger *log.Logger
}

// Command initializes a new command that will run with the provided arguments.
//
// The first argument is the name of the command to be run. If it contains no
// path separators, it will be resolved to a complete name using a PATH lookup.
func (c *Context) Command(args ...string) *Cmd {
	return &Cmd{context: c, args: args}
}

// Cmd is a builder for a command.
//
// By default, a command will inherit the environment and standard streams of
// the current process, and will return an error to indicate whether the command
// exited with a non-zero status. Methods on Cmd can override this default
// behavior where appropriate.
type Cmd struct {
	context *Context
	cmd     *exec.Cmd
	started chan struct{}

	parent   *Cmd
	args     []string
	envs     []string
	debug    bool
	nostdout bool
	nostderr bool
}

// Command initializes a new command using DefaultContext.
func Command(args ...string) *Cmd {
	return DefaultContext.Command(args...)
}

// Pipe initializes a new command whose stdin will be connected to the stdout of
// its parent.
//
// The new child command will start its parent when run, but will not inherit
// other settings (e.g. environment) from the parent. If multiple commands in a
// pipeline should have these settings, they must be specified for each command
// in the pipeline.
//
// Piped commands approximate the behavior of "set -o pipefail" in a shell.
// That is, if the child does not produce an error but the parent does, the
// child will return the parent's error. Unlike "set -o pipefail", it is
// possible to determine which command in a pipeline failed with a non-zero
// status by unwrapping the error as an ExitError and reading the contained
// Args. This partially works around the usual limitation of "set -o pipefail".
func (c *Cmd) Pipe(args ...string) *Cmd {
	return &Cmd{
		context: c.context,
		parent:  c,
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

// Debug prints a representation of the command with the log package before
// running it, roughly approximating the behavior of "set -x" in a shell.
//
// The command will be logged with any environment values explicitly set by Env,
// followed by the exact arguments that the command was constructed with, with
// shell quoting applied. Aliases will not be expanded.
func (c *Cmd) Debug() *Cmd {
	c.debug = true
	return c
}

// NoStdout suppresses the command writing its stdout stream to the stdout of
// the current process.
func (c *Cmd) NoStdout() *Cmd {
	c.nostdout = true
	return c
}

// NoStderr suppresses the command writing its stderr stream to the stderr of
// the current process.
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
	c.initCmd()
	return c.run()
}

// Text runs the command, waits for it to complete, and returns its standard
// output as a string with whitespace trimmed from both ends.
func (c *Cmd) Text() (string, error) {
	c.initCmd()

	var stdout bytes.Buffer
	c.cmd.Stdout = &stdout

	err := c.run()
	return strings.TrimSpace(stdout.String()), err
}

// Successful runs the command, waits for it to complete, and returns whether it
// exited with a status code of 0.
//
// Successful returns a non-nil error if the command failed to start, but not if
// it finished with a non-zero status.
func (c *Cmd) Successful() (bool, error) {
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
	c.started = make(chan struct{})
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

	parentErr, err := c.startParent()
	if err != nil {
		return err
	}

	if c.debug {
		c.logDebug()
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}
	close(c.started)

	err = c.cmd.Wait()

	if perr := <-parentErr; perr != nil && err == nil {
		return perr
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ExitError{exitErr, c.args}
	}
	return err
}

func (c *Cmd) startParent() (parentErr chan error, err error) {
	parentErr = make(chan error, 1)
	if c.parent == nil {
		parentErr <- nil
		return
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return
	}

	// Initialize the parent's state and set up the pipe before calling run(), so
	// the parent will see that it shouldn't just redirect to os.Stdout.
	c.parent.initCmd()
	c.parent.cmd.Stdout = pw
	c.cmd.Stdin = pr

	go func() {
		// We need to clean up our references to both ends of the pipe, but only
		// after we have started the parent process and allowed it to duplicate
		// those references. In theory we could do this as soon as the parent
		// starts, but instead we just leave our handles open until the parent is
		// done, since it's easier to implement this way.
		defer pr.Close()
		defer pw.Close()

		// Now, we can finally start the parent.
		parentErr <- c.parent.run()
	}()

	return
}

func (c *Cmd) logDebug() {
	var envString string
	for _, env := range c.envs {
		split := strings.SplitN(env, "=", 2)
		envString += split[0] + "=" + shellquote.Join(split[1]) + " "
	}

	if c.parent != nil {
		<-c.parent.started
	}

	c.context.DebugLogger.Print("+ " + envString + shellquote.Join(c.args...))
}

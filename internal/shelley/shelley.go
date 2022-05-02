// Package shelley runs commands with behavior similar to a command line shell.
package shelley

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
)

// ExitError is the type of error returned by commands that completed with a
// non-zero exit code.
type ExitError = *exec.ExitError

// Cmd is a builder for a command.
//
// By default, a command will inherit the environment and standard streams of
// the current process, and will return an error to indicate whether the command
// exited with a non-zero status. Methods on Cmd can override this default
// behavior where appropriate.
type Cmd struct {
	cmd *exec.Cmd

	parent   *Cmd
	args     []string
	envs     []string
	debug    bool
	errexit  bool
	nostdout bool
	nostderr bool
}

// Command initializes a new command that will run with the provided arguments.
//
// The first argument is the name of the command to be run. If it contains no
// path separators, it will be resolved to a complete name using a PATH lookup.
func Command(args ...string) *Cmd {
	return &Cmd{args: args}
}

// Pipe initializes a new command whose stdin will be connected to the stdout of
// its parent.
//
// The new child command will start its parent when run, but will not inherit
// any other settings from the parent (environment, Debug, ErrExit, etc.). If
// multiple commands in a pipeline should have these settings, they must be
// specified for each command in the pipeline.
func (c *Cmd) Pipe(args ...string) *Cmd {
	return &Cmd{parent: c, args: args}
}

// Env appends an environment value to the command.
//
// The appended value overrides any value inherited from the current process or
// set by a previous Env call.
func (c *Cmd) Env(name, value string) *Cmd {
	c.envs = append(c.envs, name+"="+value)
	return c
}

// Debug causes the full command to be printed with the log package before it is
// run, approximating the behavior of "set -x" in a shell.
func (c *Cmd) Debug() *Cmd {
	c.debug = true
	return c
}

// ErrExit causes the current process to exit if the command fails to start, or
// if regular execution of the command exits with a non-zero status.
//
// ErrExit approximates the behavior of "set -e" in a shell, with many of the
// same caveats and dangerous pitfalls. Some methods modify the typical behavior
// of ErrExit, see the documentation of those functions for details.
//
// For commands exiting with a non-zero status, the current process will exit
// with the same code as the command. For commands that fail to start, the error
// will be logged with the log package and the current process will exit with
// status 1.
func (c *Cmd) ErrExit() *Cmd {
	c.errexit = true
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
	err := c.run()
	if err != nil && c.errexit {
		c.exitForError(err)
	}
	return err
}

// Text runs the command, waits for it to complete, and returns its standard
// output as a string with whitespace trimmed from both ends.
func (c *Cmd) Text() (string, error) {
	c.initCmd()

	var stdout bytes.Buffer
	c.cmd.Stdout = &stdout

	err := c.run()
	if err != nil && c.errexit {
		c.exitForError(err)
	}

	return strings.TrimSpace(stdout.String()), err
}

// Successful runs the command, waits for it to complete, and returns whether it
// exited with a status code of 0.
//
// Successful returns a non-nil error if the command failed to start, but not if
// it finished with a non-zero status. With ErrExit enabled, Successful will
// only exit the current process if the command failed to start.
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

	if c.errexit {
		c.exitForError(err)
	}
	return false, err
}

func (c *Cmd) initCmd() {
	c.cmd = exec.Command(c.args[0], c.args[1:]...)
	c.cmd.Env = append(os.Environ(), c.envs...)
}

func (c *Cmd) run() error {
	if c.cmd.Stdin == nil {
		c.cmd.Stdin = os.Stdin
	}
	if c.cmd.Stdout == nil && !c.nostdout {
		c.cmd.Stdout = os.Stdout
	}
	if c.cmd.Stderr == nil && !c.nostderr {
		c.cmd.Stderr = os.Stderr
	}

	parentErr := make(chan error, 1)
	if c.parent != nil {
		pr, pw, err := os.Pipe()
		if err != nil {
			return err
		}
		defer pr.Close()
		defer pw.Close()

		c.parent.initCmd()
		c.parent.cmd.Stdout = pw
		c.cmd.Stdin = pr
		go func() {
			defer pw.Close()
			parentErr <- c.parent.run()
		}()
	} else {
		parentErr <- nil
	}

	if c.debug {
		c.logDebug()
	}

	err := c.cmd.Run()
	if perr := <-parentErr; perr != nil {
		return perr
	}
	return err
}

func (c *Cmd) exitForError(err error) {
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	log.Fatal(err)
}

func (c *Cmd) logDebug() {
	var envString string
	for _, env := range c.envs {
		split := strings.SplitN(env, "=", 2)
		envString += split[0] + "=" + shellquote.Join(split[1]) + " "
	}
	log.Print("+ " + envString + shellquote.Join(c.args...))
}

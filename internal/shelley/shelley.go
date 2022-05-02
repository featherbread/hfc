// Package shelley runs commands with behavior similar to a command line shell.
package shelley

import (
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
	args    []string
	envs    []string
	debug   bool
	errexit bool
}

// Command initializes a new command that will run with the provided arguments.
//
// The first argument is the name of the command to be run. If it contains no
// path separators, it will be resolved to a complete name using a PATH lookup.
func Command(args ...string) *Cmd {
	return &Cmd{args: args}
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

// ErrExit causes the current process to exit if the command fails to start or
// exits with a non-zero status, approximating the behavior of "set -e" in a
// shell. When ErrExit is used, Run will never return a non-nil error.
//
// For commands exiting with a non-zero status, the current process will exit
// with the same code as the command. For commands that fail to start, the error
// will be logged with the log package and the current process will exit with
// status 1.
func (c *Cmd) ErrExit() *Cmd {
	c.errexit = true
	return c
}

// Run runs the command and waits for it to complete.
func (c *Cmd) Run() error {
	cmd := exec.Command(c.args[0], c.args[1:]...)
	cmd.Env = append(os.Environ(), c.envs...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if c.debug {
		c.logDebug()
	}

	err := cmd.Run()

	if err != nil && c.errexit {
		var exitErr ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal(err)
	}

	return err
}

func (c *Cmd) logDebug() {
	var envString string
	for _, env := range c.envs {
		split := strings.SplitN(env, "=", 2)
		envString += split[0] + "=" + shellquote.Join(split[1]) + " "
	}
	log.Print("+ " + envString + shellquote.Join(c.args...))
}

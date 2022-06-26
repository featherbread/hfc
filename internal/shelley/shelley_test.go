package shelley

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// It's a Unix system! I know this!
	//
	// Or, maybe it's not, and I don't. This is sort of a hack to globally skip
	// these tests if we can't assume that a reasonable baseline set of commands
	// is available.
	requiredCommands := []string{"sh", "cat", "false", "sort"}
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			return
		}
	}

	os.Exit(m.Run())
}

func TestRunWithoutOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	context := &Context{
		Stdin:  strings.NewReader("stdin\n"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err := context.Command("sh", "-c", "cat; echo stdout; echo stderr 1>&2").Run()
	if err != nil {
		t.Fatal(err)
	}

	const wantStdout = "stdin\nstdout\n"
	if stdout.String() != wantStdout {
		t.Errorf("unexpected stdout; got %q, want %q", stdout.String(), wantStdout)
	}

	const wantStderr = "stderr\n"
	if stderr.String() != wantStderr {
		t.Errorf("unexpected stderr; got %q, want %q", stderr.String(), wantStderr)
	}
}

func TestExitError(t *testing.T) {
	err := Command("false").Run()
	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("error was not an ExitError: %v", err)
	}
}

func TestDebug(t *testing.T) {
	var stdout, debug bytes.Buffer
	context := &Context{
		Stdin:       strings.NewReader("one\ntwo\nthree\n"),
		Stdout:      &stdout,
		DebugLogger: log.New(&debug, "", 0),
	}

	err := context.Command("sort").Env("LC_ALL", "C").Run()
	if err != nil {
		t.Fatal(err)
	}

	const wantStdout = "one\nthree\ntwo\n"
	if stdout.String() != wantStdout {
		t.Errorf("unexpected output; got %q, want %q", stdout.String(), wantStdout)
	}

	const wantDebug = "LC_ALL=C sort\n"
	if debug.String() != wantDebug {
		t.Errorf("unexpected debug; got %q, want %q", debug.String(), wantDebug)
	}
}

func TestEnv(t *testing.T) {
	var stdout bytes.Buffer
	context := &Context{Stdout: &stdout}

	err := context.Command("sh", "-c", `echo "$SHELLEY"`).Env("SHELLEY", "shelley").Run()
	if err != nil {
		t.Fatal(err)
	}

	const wantStdout = "shelley\n"
	if stdout.String() != wantStdout {
		t.Errorf("unexpected output; got %q, want %q", stdout.String(), wantStdout)
	}
}

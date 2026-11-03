// Package gitx runs the system git binary.
//
// mestiere shells out instead of using a Go git library so that private registries
// work with whatever authentication the user's git already has (SSH agent,
// credential helpers, gh auth, proxies).
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Error struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		msg = e.Err.Error()
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), msg)
}

func (e *Error) Unwrap() error { return e.Err }

// Run executes git in dir and returns trimmed stdout.
func Run(dir string, args ...string) (string, error) {
	out, err := RunInput(dir, nil, args...)
	return strings.TrimSpace(string(out)), err
}

// RunInput executes git in dir with stdin and returns raw stdout.
func RunInput(dir string, stdin io.Reader, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), &Error{Args: args, Stderr: stderr.String(), Err: err}
	}
	return stdout.Bytes(), nil
}

// ExitCode returns the git process exit code, or -1 if git did not run.
func ExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func Available() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is required but was not found in PATH")
	}
	return nil
}

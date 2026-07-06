// Demonstrates running external processes with os/exec: capturing output,
// feeding stdin, reading exit codes with errors.As, controlling the child's
// environment, and killing a runaway process via context timeout — the
// plumbing behind every Go tool that shells out.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := captureOutput(ctx); err != nil {
		return err
	}
	if err := feedStdin(ctx); err != nil {
		return err
	}
	if err := readExitCode(ctx); err != nil {
		return err
	}
	if err := controlEnvironment(ctx); err != nil {
		return err
	}
	return killOnTimeout(ctx)
}

// captureOutput runs a command and collects its stdout. Output (as opposed to
// CombinedOutput) keeps stderr separate — mixing them makes output unparseable
// the day the tool prints a warning.
func captureOutput(ctx context.Context) error {
	fmt.Println("--- Output: run and capture stdout ---")
	out, err := exec.CommandContext(ctx, "echo", "hello from a child process").Output()
	if err != nil {
		return fmt.Errorf("running echo: %w", err)
	}
	fmt.Printf("captured: %q\n", strings.TrimSpace(string(out)))
	return nil
}

// feedStdin pipes data into the child's stdin — here `tr` upper-cases it.
// Any io.Reader works: a file, a network stream, another command's output.
func feedStdin(ctx context.Context) error {
	fmt.Println("\n--- Stdin: pipe data into the child ---")
	cmd := exec.CommandContext(ctx, "tr", "a-z", "A-Z")
	cmd.Stdin = strings.NewReader("shout this\n")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running tr: %w", err)
	}
	fmt.Printf("tr said: %q\n", strings.TrimSpace(string(out)))
	return nil
}

// readExitCode distinguishes "the command ran and failed" (*exec.ExitError,
// carries the exit code) from "the command never ran" (bad path, permissions).
// Tools speak through exit codes; conflating the two cases loses that signal.
func readExitCode(ctx context.Context) error {
	fmt.Println("\n--- ExitError: read the child's exit code ---")
	err := exec.CommandContext(ctx, "sh", "-c", "exit 3").Run()

	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		fmt.Printf("command failed as expected with exit code %d\n", exitErr.ExitCode())
	case err != nil:
		return fmt.Errorf("command could not start: %w", err)
	default:
		return errors.New("expected a non-zero exit, got success")
	}
	return nil
}

// controlEnvironment sets the child's env explicitly. Setting cmd.Env replaces
// the inherited environment entirely — start from os.Environ() and append,
// or the child loses PATH, HOME, and friends.
func controlEnvironment(ctx context.Context) error {
	fmt.Println("\n--- Env: control what the child sees ---")
	cmd := exec.CommandContext(ctx, "sh", "-c", `echo "greeting is: $GREETING"`)
	cmd.Env = append(os.Environ(), "GREETING=hola")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running env demo: %w", err)
	}
	fmt.Printf("child saw: %q\n", strings.TrimSpace(string(out)))
	return nil
}

// killOnTimeout shows CommandContext's reason for existing: when the context
// expires, the child is killed instead of hanging the parent forever.
func killOnTimeout(ctx context.Context) error {
	fmt.Println("\n--- CommandContext: kill a runaway child ---")
	timeoutCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := exec.CommandContext(timeoutCtx, "sleep", "30").Run()
	elapsed := time.Since(start)

	if err == nil {
		return errors.New("expected the 30s sleep to be killed, but it finished")
	}
	if elapsed >= 30*time.Second {
		return fmt.Errorf("child was not killed by the deadline (took %v)", elapsed)
	}
	fmt.Printf("30s sleep killed after the 200ms deadline (context: %v, child: %v)\n",
		context.Cause(timeoutCtx), err)
	return nil
}

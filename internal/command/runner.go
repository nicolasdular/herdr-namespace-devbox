package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Runner executes external commands. Keeping this boundary small makes command
// construction testable without replacing binaries on PATH.
type Runner interface {
	CombinedOutput(ctx context.Context, executable string, args ...string) ([]byte, error)
	Run(ctx context.Context, executable string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type OSRunner struct{}

func (OSRunner) CombinedOutput(ctx context.Context, executable string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, executable, args...).CombinedOutput()
	if err != nil && ctx.Err() != nil {
		return output, ctx.Err()
	}
	return output, err
}

func (OSRunner) Run(
	ctx context.Context,
	executable string,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// Command binds an executable to its runner and terminal streams.
// Domain clients can therefore focus on arguments and results rather than the
// mechanics of starting subprocesses.
type Command struct {
	executable string
	runner     Runner
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
}

func New(executable string) Command {
	return NewWithRunner(executable, OSRunner{})
}

func NewWithRunner(executable string, runner Runner) Command {
	return Command{
		executable: executable,
		runner:     runner,
	}
}

func (c Command) WithStreams(stdin io.Reader, stdout, stderr io.Writer) Command {
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	return c
}

func (c Command) Output(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	output, err := c.runner.CombinedOutput(ctx, c.executable, args...)
	if err != nil {
		return output, failure(c.executable, args, output, err)
	}
	return output, nil
}

func (c Command) Run(ctx context.Context, args ...string) error {
	return c.RunWithStdin(ctx, c.stdin, args...)
}

func (c Command) RunWithStdin(ctx context.Context, stdin io.Reader, args ...string) error {
	if err := c.runner.Run(ctx, c.executable, args, stdin, c.stdout, c.stderr); err != nil {
		return failure(c.executable, args, nil, err)
	}
	return nil
}

// failure adds captured command output to an execution error while retaining
// the original error for errors.Is/errors.As and exit-code inspection.
func failure(executable string, args []string, output []byte, err error) error {
	invocation := make([]string, 0, len(args)+1)
	for _, arg := range append([]string{executable}, args...) {
		invocation = append(invocation, strconv.Quote(arg))
	}
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return fmt.Errorf("run %s: %w", strings.Join(invocation, " "), err)
	}
	return fmt.Errorf("run %s: %w: %s", strings.Join(invocation, " "), err, detail)
}

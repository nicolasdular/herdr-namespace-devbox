package namespace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"herdr-namespace/internal/command"
)

const (
	probeTimeout     = 15 * time.Second
	operationTimeout = 2 * time.Minute
)

type Client struct {
	cmd command.Command
}

func New() Client {
	return Client{
		cmd: command.New("devbox").WithStreams(os.Stdin, os.Stdout, os.Stderr),
	}
}

func (c Client) combinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	return c.cmd.Output(ctx, probeTimeout, args...)
}

func (c Client) Preflight(ctx context.Context) error {
	_, err := c.combinedOutput(ctx, "version")
	if err != nil {
		return fmt.Errorf("Namespace Devbox CLI is unavailable or not working: %w", err)
	}
	return nil
}

func (c Client) IsAuthenticated(ctx context.Context) (bool, error) {
	_, err := c.combinedOutput(ctx, "auth", "check-login")
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return false, nil
	}
	return false, fmt.Errorf("could not check Namespace authentication: %w", err)
}

func (c Client) Login(ctx context.Context) error {
	if err := c.cmd.Run(ctx, "login"); err != nil {
		return fmt.Errorf("Namespace login did not complete successfully: %w", err)
	}
	return nil
}

func (c Client) Exists(ctx context.Context, name string) (bool, error) {
	devboxes, err := c.List(ctx)
	if err != nil {
		return false, err
	}
	for _, devbox := range devboxes {
		if devbox.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (c Client) List(ctx context.Context) ([]Devbox, error) {
	output, err := c.combinedOutput(ctx, "list", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("could not list Namespace Devboxes: %w", err)
	}
	devboxes, err := parseDevboxList(output)
	if err != nil {
		return nil, err
	}
	return devboxes, nil
}

type Devbox struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	CreatedAt     string        `json:"created_at"`
	LastUsedAt    string        `json:"last_used_at"`
	Repository    string        `json:"repository"`
	Site          string        `json:"site"`
	DefaultDir    string        `json:"default_dir"`
	InstanceShape InstanceShape `json:"instance_shape"`
}

type InstanceShape struct {
	VirtualCPU      int    `json:"virtual_cpu"`
	MemoryMegabytes int    `json:"memory_megabytes"`
	MachineArch     string `json:"machine_arch"`
	OS              string `json:"os"`
}

func parseDevboxList(output []byte) ([]Devbox, error) {
	start := bytes.IndexByte(output, '[')
	end := bytes.LastIndexByte(output, ']')
	if start < 0 || end < start {
		return nil, errors.New("Namespace returned an invalid Devbox list")
	}
	var devboxes []Devbox
	if err := json.Unmarshal(output[start:end+1], &devboxes); err != nil {
		return nil, fmt.Errorf("parse Namespace Devbox list: %w", err)
	}
	return devboxes, nil
}

func (c Client) Create(ctx context.Context, spec Spec, options CreateOptions) error {
	contents, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	args := []string{"create", "--from", "-", "--from_format", "json"}
	if options.Dotfiles != "" {
		args = append(args, "--dotfiles", options.Dotfiles)
	}
	if err := c.cmd.RunWithStdin(ctx, bytes.NewReader(contents), args...); err != nil {
		return fmt.Errorf("Namespace could not create Devbox %s: %w", spec.Name, err)
	}
	return nil
}

func (c Client) Upload(ctx context.Context, name, localPath, remotePath string) error {
	if err := c.cmd.Run(ctx, "upload", name, localPath, remotePath); err != nil {
		return fmt.Errorf("upload local changes to Devbox %s: %w", name, err)
	}
	return nil
}

func (c Client) Exec(ctx context.Context, name string, args ...string) error {
	commandArgs := []string{"exec", name, "--"}
	commandArgs = append(commandArgs, args...)
	if err := c.cmd.Run(ctx, commandArgs...); err != nil {
		return fmt.Errorf("run command in Devbox %s: %w", name, err)
	}
	return nil
}

func (c Client) Stop(ctx context.Context, name string) error {
	if _, err := c.cmd.Output(ctx, operationTimeout, "stop", name, "--force"); err != nil {
		return fmt.Errorf("Namespace could not stop Devbox %s: %w", name, err)
	}
	return nil
}

func (c Client) Delete(ctx context.Context, name string) error {
	if _, err := c.cmd.Output(ctx, operationTimeout, "expire", name, "--force"); err != nil {
		return fmt.Errorf("Namespace could not delete Devbox %s: %w", name, err)
	}
	return nil
}

func (c Client) Connect(ctx context.Context, name, sessionName string) (int, error) {
	args := []string{"session", "connect", name}
	if sessionName != "" {
		args = append(args, "--session", sessionName)
	}
	err := c.cmd.Run(ctx, args...)
	if err == nil {
		return 0, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 1, fmt.Errorf("Devbox terminal connection ended with signal %s", status.Signal())
		}
		return exitError.ExitCode(), nil
	}
	return 1, fmt.Errorf("could not connect to Namespace Devbox session: %w", err)
}

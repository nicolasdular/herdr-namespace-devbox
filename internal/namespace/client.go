package namespace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"herdr-namespace/internal/config"
)

type devboxSpec struct {
	Name                string        `json:"name"`
	Image               string        `json:"image"`
	Size                string        `json:"size"`
	AccessMode          string        `json:"access_mode"`
	AutoStopIdleTimeout string        `json:"auto_stop_idle_timeout"`
	Repository          repository    `json:"repository"`
	Sessions            []session     `json:"sessions"`
	Integrations        *integrations `json:"integrations,omitempty"`
	VolumeSizeGB        *int          `json:"volume_size_gb,omitempty"`
	Site                *string       `json:"site,omitempty"`
}

type repository struct {
	Disabled bool `json:"disabled"`
}
type session struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}
type integrations struct {
	GitHub githubIntegration `json:"github"`
}
type githubIntegration struct {
	ShareAuth bool `json:"share_auth"`
}

func makeDevboxSpec(name string, cfg config.Config) devboxSpec {
	spec := devboxSpec{
		Name: name, Image: cfg.Image, Size: cfg.Size, AccessMode: cfg.AccessMode,
		AutoStopIdleTimeout: cfg.AutoStopIdleTimeout,
		Repository:          repository{Disabled: true},
		Sessions:            []session{{Name: cfg.SessionName, Command: cfg.Shell}},
		VolumeSizeGB:        cfg.VolumeSizeGB, Site: cfg.Site,
	}
	if cfg.SetupGitHub {
		spec.Integrations = &integrations{GitHub: githubIntegration{ShareAuth: true}}
	}
	return spec
}

type Client struct {
	executable string
}

func New(executable string) Client {
	return Client{executable: executable}
}

func (c Client) command(args ...string) *exec.Cmd {
	return exec.Command(c.executable, args...)
}

func (c Client) Preflight() error {
	if err := c.command("version").Run(); err != nil {
		return errors.New("Namespace Devbox CLI is unavailable or not working")
	}
	return nil
}

func (c Client) IsAuthenticated() (bool, error) {
	err := c.command("auth", "check-login").Run()
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return false, nil
	}
	return false, fmt.Errorf("could not check Namespace authentication: %w", err)
}

func (c Client) Login() error {
	command := c.command("login")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return errors.New("Namespace login did not complete successfully")
	}
	return nil
}

func (c Client) Create(name string, cfg config.Config) error {
	spec, err := json.Marshal(makeDevboxSpec(name, cfg))
	if err != nil {
		return err
	}
	command := c.command("create", "--from", "-", "--from_format", "json")
	command.Stdin, command.Stdout, command.Stderr = bytes.NewReader(spec), os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("Namespace could not create Devbox %s", name)
	}
	return nil
}

func (c Client) Connect(name, sessionName string) (int, error) {
	command := c.command("session", "connect", name, "--session", sessionName)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := command.Run()
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
	return 1, fmt.Errorf("could not run %s: %w", c.executable, err)
}

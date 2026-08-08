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
	Disabled bool   `json:"disabled,omitempty"`
	URL      string `json:"url,omitempty"`
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

func makeDevboxSpec(name string, cfg config.Config, repositoryURL string) devboxSpec {
	repo := repository{Disabled: true}
	if repositoryURL != "" {
		repo = repository{URL: repositoryURL}
	}
	spec := devboxSpec{
		Name: name, Image: cfg.Image, Size: cfg.Size, AccessMode: cfg.AccessMode,
		AutoStopIdleTimeout: cfg.AutoStopIdleTimeout,
		Repository:          repo,
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

func New() Client {
	return Client{executable: "devbox"}
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

func (c Client) Exists(name string) (bool, error) {
	output, err := c.command("list", "-o", "json").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("could not list Namespace Devboxes: %w", err)
	}
	devboxes, err := parseDevboxList(output)
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

type devboxSummary struct {
	Name string `json:"name"`
}

func parseDevboxList(output []byte) ([]devboxSummary, error) {
	start := bytes.IndexByte(output, '[')
	end := bytes.LastIndexByte(output, ']')
	if start < 0 || end < start {
		return nil, errors.New("Namespace returned an invalid Devbox list")
	}
	var devboxes []devboxSummary
	if err := json.Unmarshal(output[start:end+1], &devboxes); err != nil {
		return nil, fmt.Errorf("parse Namespace Devbox list: %w", err)
	}
	return devboxes, nil
}

func (c Client) Create(name string, cfg config.Config, repositoryURL string) error {
	spec, err := json.Marshal(makeDevboxSpec(name, cfg, repositoryURL))
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

func (c Client) CreateFromSpec(name, path string) error {
	command := c.command("create", "--from", path, "--name", name)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
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

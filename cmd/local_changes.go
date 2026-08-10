package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/namespace"
)

const gitPatchTimeout = 30 * time.Second

type LocalChangesInfo struct {
	BaseCommit string
	FileCount  int
}

type LocalChangesService interface {
	Inspect(context.Context, string, namespace.Repository) (LocalChangesInfo, error)
	GeneratePatch(context.Context, string, namespace.Repository) ([]byte, error)
}

type gitLocalChangesService struct {
	runner command.Runner
}

func newGitLocalChangesService(runner command.Runner) LocalChangesService {
	return gitLocalChangesService{runner: runner}
}

func (s gitLocalChangesService) Inspect(
	ctx context.Context,
	workspace string,
	repository namespace.Repository,
) (LocalChangesInfo, error) {
	base, err := s.resolveBase(ctx, workspace, repository)
	if err != nil {
		return LocalChangesInfo{}, err
	}
	git := command.NewWithRunner("git", s.runner)
	output, err := git.Output(
		ctx, gitPatchTimeout,
		"-C", workspace, "diff", "--name-only", "-z", "--no-ext-diff", base.commit, "--",
	)
	if err != nil {
		return LocalChangesInfo{}, fmt.Errorf("inspect tracked local changes: %w", err)
	}
	return LocalChangesInfo{
		BaseCommit: base.commit,
		FileCount:  strings.Count(string(output), "\x00"),
	}, nil
}

func (s gitLocalChangesService) GeneratePatch(
	ctx context.Context,
	workspace string,
	repository namespace.Repository,
) ([]byte, error) {
	base, err := s.resolveBase(ctx, workspace, repository)
	if err != nil {
		return nil, err
	}
	git := command.NewWithRunner("git", s.runner)
	patch, err := git.Output(
		ctx, gitPatchTimeout,
		"-C", workspace, "diff", "--binary", "--full-index", "--no-ext-diff", base.commit, "--",
	)
	if err != nil {
		return nil, fmt.Errorf("generate tracked local changes: %w", err)
	}
	return patch, nil
}

type localChangesBase struct {
	commit string
}

func (s gitLocalChangesService) resolveBase(
	ctx context.Context,
	workspace string,
	repository namespace.Repository,
) (localChangesBase, error) {
	git := command.NewWithRunner("git", s.runner)
	origin, err := git.Output(ctx, gitPatchTimeout, "-C", workspace, "remote", "get-url", "origin")
	if err != nil {
		return localChangesBase{}, fmt.Errorf("verify local changes repository: %w", err)
	}
	localRepository := normalizeRepositoryURL(strings.TrimSpace(string(origin)))
	effectiveRepository := normalizeRepositoryURL(repository.URL)
	if localRepository == "" || localRepository != effectiveRepository {
		return localChangesBase{}, fmt.Errorf(
			"local changes are unavailable because devbox.yaml clones %s, not workspace origin %s",
			repository.URL,
			strings.TrimSpace(string(origin)),
		)
	}

	ref := "HEAD"
	if repository.Ref != "" {
		ref = repository.Ref
	}
	resolved, err := git.Output(
		ctx,
		gitPatchTimeout,
		"-C", workspace, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}",
	)
	if err != nil {
		return localChangesBase{}, fmt.Errorf("resolve Devbox repository ref %q locally: %w", ref, err)
	}
	return localChangesBase{commit: strings.TrimSpace(string(resolved))}, nil
}

func formatByteSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f KiB", float64(size)/1024)
}

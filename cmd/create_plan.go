package main

import (
	"context"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/namespace"
)

// DevboxCreatePlan is the resolved, immutable configuration shown by the
// creation form. Transient UI state intentionally lives elsewhere.
type DevboxCreatePlan struct {
	Name       string
	Repository *namespace.Repository
	Image      string
	Size       string
	Site       string
}

func resolveCreatePlan(
	ctx context.Context,
	inputs ActionInputs,
	runner command.Runner,
) (DevboxCreatePlan, error) {
	name := namespace.NewDevboxName(inputs.Workspace)
	repository := repositoryURL(ctx, runner, inputs.Workspace)
	spec, err := namespace.NewSpec(inputs.Workspace, name, repository)
	if err != nil {
		return DevboxCreatePlan{}, err
	}

	plan := DevboxCreatePlan{
		Name:  name,
		Image: spec.Image,
		Size:  spec.Size,
		Site:  "automatic",
	}
	if spec.Repository != nil && !spec.Repository.Disabled && spec.Repository.URL != "" {
		repository := *spec.Repository
		plan.Repository = &repository
	}
	if spec.Site != nil && *spec.Site != "" {
		plan.Site = *spec.Site
	}
	return plan, nil
}

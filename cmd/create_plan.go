package main

import (
	"context"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/namespace"
)

// DevboxCreatePlan is the resolved configuration edited by the creation form.
// Transient UI state intentionally lives elsewhere.
type DevboxCreatePlan struct {
	Name       string
	Repository *namespace.Repository
	Image      string
	Size       string
	Site       string
}

func (p DevboxCreatePlan) apply(spec namespace.Spec) namespace.Spec {
	spec.Name = p.Name
	spec.Image = p.Image
	spec.Size = p.Size
	if p.Repository == nil || p.Repository.URL == "" {
		spec.Repository = &namespace.Repository{Disabled: true}
	} else {
		repository := *p.Repository
		spec.Repository = &repository
	}
	if p.Site == "" || p.Site == "automatic" {
		spec.Site = nil
	} else {
		site := p.Site
		spec.Site = &site
	}
	return spec
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

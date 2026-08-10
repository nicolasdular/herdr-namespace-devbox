package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type changesState int

const (
	changesNotLoaded changesState = iota
	changesLoading
	changesAvailable
	changesUnavailable
)

type createFormField int

const (
	createFormName createFormField = iota
	createFormRepository
	createFormImage
	createFormSize
	createFormLocation
	createFormUpload
	createFormSubmit
	createFormFieldCount
)

type devboxCreateForm struct {
	Plan               DevboxCreatePlan
	ChangesState       changesState
	TrackedChanges     int
	UploadLocalChanges bool
}

func newDevboxCreateForm(plan DevboxCreatePlan) devboxCreateForm {
	return devboxCreateForm{Plan: plan}
}

func createPlanRepository(plan DevboxCreatePlan) string {
	if plan.Repository == nil {
		return "none"
	}
	return plan.Repository.URL
}

func (f devboxCreateForm) canUploadLocalChanges() bool {
	return f.ChangesState == changesAvailable && f.TrackedChanges > 0
}

func (f *devboxCreateForm) beginChangesInspection() bool {
	if f.Plan.Repository == nil || f.ChangesState == changesLoading {
		return false
	}
	f.ChangesState = changesLoading
	f.TrackedChanges = 0
	f.UploadLocalChanges = false
	return true
}

func (f *devboxCreateForm) finishChangesInspection(info LocalChangesInfo, err error) {
	f.TrackedChanges = info.FileCount
	if err != nil {
		f.ChangesState = changesUnavailable
		f.UploadLocalChanges = false
		return
	}
	f.ChangesState = changesAvailable
}

func (m devboxManager) createFormView() tea.View {
	contentWidth := max(m.width-4, 26)
	form := m.createForm
	location := form.Plan.Site
	if location == "automatic" {
		location = "Closest available"
	}
	upload := "[ No ]   Yes"
	if form.UploadLocalChanges {
		upload = "No   [ Yes ]"
	}

	fileSuffix := "s"
	if form.TrackedChanges == 1 {
		fileSuffix = ""
	}
	changeSummary := fmt.Sprintf("%d tracked file%s changed", form.TrackedChanges, fileSuffix)
	switch form.ChangesState {
	case changesLoading:
		upload = "Inspecting…"
		changeSummary = m.spinner.View() + " Inspecting tracked local changes"
	case changesUnavailable:
		upload = "Unavailable"
		changeSummary = "Could not inspect tracked local changes"
	default:
		if form.TrackedChanges == 0 {
			upload = "No tracked changes"
			changeSummary = "Only tracked modifications can be uploaded"
		}
	}

	row := func(field createFormField, label, value string) string {
		const labelWidth = 15
		text := sanitizeText(value)
		if label != "" {
			text = fmt.Sprintf("%-*s%s", labelWidth, label, text)
		}
		text = "  " + ansi.Truncate(text, max(contentWidth-2, 1), "…")
		if m.createField == field {
			text = "› " + strings.TrimPrefix(text, "  ")
			return lipgloss.NewStyle().Reverse(true).Render(text)
		}
		return text
	}

	hint := "Editing is not available yet"
	switch m.createField {
	case createFormName:
		hint = "Generated from the workspace · editing is not available yet"
	case createFormRepository:
		hint = "Derived from devbox.yaml or Git origin · editing is not available yet"
	case createFormUpload:
		switch form.ChangesState {
		case changesLoading:
			hint = "Inspecting tracked local changes"
		case changesUnavailable:
			hint = "Could not inspect tracked local changes"
		default:
			if form.TrackedChanges == 0 {
				hint = "There are no tracked local changes to upload"
			} else {
				hint = "space/enter/←/→ toggle"
			}
		}
	case createFormSubmit:
		hint = "enter create"
	}
	hint = ansi.Truncate(hint, contentWidth, "…")
	footer := ansi.Truncate("↑/↓ or j/k select  tab next  esc back  q close", contentWidth, "…")
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Render("Create Namespace Devbox"),
		"",
		row(createFormName, "Name", form.Plan.Name),
		row(createFormRepository, "Repository", createPlanRepository(form.Plan)),
		row(createFormImage, "Image", form.Plan.Image),
		row(createFormSize, "Size", form.Plan.Size),
		row(createFormLocation, "Location", location),
		"",
		row(createFormUpload, "Local changes", upload),
		"  " + ansi.Truncate(changeSummary, max(contentWidth-2, 1), "…"),
		"",
		row(createFormSubmit, "", "[ Create Devbox ]"),
		"",
		hint,
		footer,
	}, "\n")
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

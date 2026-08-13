package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
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
	Input              textinput.Model
	Editing            bool
	Error              string
}

func newDevboxCreateForm(plan DevboxCreatePlan) devboxCreateForm {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 200
	return devboxCreateForm{Plan: plan, Input: input}
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

func (f *devboxCreateForm) beginEditing(field createFormField) tea.Cmd {
	value := ""
	switch field {
	case createFormName:
		value = f.Plan.Name
	case createFormImage:
		value = f.Plan.Image
	case createFormSize:
		value = f.Plan.Size
	case createFormLocation:
		if f.Plan.Site != "automatic" {
			value = f.Plan.Site
		}
	default:
		return nil
	}
	f.Input.SetValue(value)
	f.Input.CursorEnd()
	f.Editing = true
	f.Error = ""
	return f.Input.Focus()
}

func (f *devboxCreateForm) cancelEditing() {
	f.Editing = false
	f.Error = ""
	f.Input.Blur()
}

func (f *devboxCreateForm) commitEditing(field createFormField) bool {
	value := strings.TrimSpace(f.Input.Value())
	if field == createFormName && value == "" {
		f.Error = "Name cannot be empty"
		return false
	}

	switch field {
	case createFormName:
		f.Plan.Name = value
	case createFormImage:
		f.Plan.Image = value
	case createFormSize:
		f.Plan.Size = value
	case createFormLocation:
		if value == "" || strings.EqualFold(value, "automatic") {
			f.Plan.Site = "automatic"
		} else {
			f.Plan.Site = value
		}
	default:
		return false
	}

	f.Editing = false
	f.Error = ""
	f.Input.Blur()
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
	upload := "◉ No   ○ Yes"
	if form.UploadLocalChanges {
		upload = "○ No   ◉ Yes"
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
		if m.createField == field && form.Editing {
			form.Input.SetWidth(max(contentWidth-labelWidth-2, 4))
			return fmt.Sprintf("› %-*s%s", labelWidth, label, form.Input.View())
		}
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

	hint := "enter edit"
	switch m.createField {
	case createFormName:
		hint = "enter edit · generated from the workspace"
	case createFormRepository:
		hint = "Derived from devbox.yaml or Git origin"
	case createFormLocation:
		hint = "enter edit · leave empty for the closest available location"
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
	if form.Editing {
		hint = "enter save · esc cancel"
	}
	if form.Error != "" {
		hint = form.Error
	}
	hint = ansi.Truncate(hint, contentWidth, "…")
	footer := "↑/↓ or j/k select  tab next  esc back  q close"
	if form.Editing {
		footer = "enter save  tab save/next  esc cancel  ctrl+c close"
	}
	footer = ansi.Truncate(footer, contentWidth, "…")
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

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"herdr-namespace/internal/namespace"
)

type managerClient interface {
	List(context.Context) ([]namespace.Devbox, error)
	Stop(context.Context, string) error
	Delete(context.Context, string) error
}

type managerOperation int

const (
	managerOperationNone managerOperation = iota
	managerOperationRefresh
	managerOperationStop
	managerOperationDelete
)

type managerOperationResult struct {
	operation managerOperation
	target    string
	devboxes  []namespace.Devbox
	err       error
}

type devboxItem struct {
	namespace.Devbox
}

func (i devboxItem) FilterValue() string {
	return i.Name + " " + i.Repository
}

type devboxDelegate struct {
	now func() time.Time
}

func (devboxDelegate) Height() int  { return 3 }
func (devboxDelegate) Spacing() int { return 1 }
func (devboxDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d devboxDelegate) Render(output io.Writer, model list.Model, index int, item list.Item) {
	devbox, ok := item.(devboxItem)
	if !ok {
		return
	}
	contentWidth := max(model.Width()-4, 20)
	name := ansi.Truncate(sanitizeText(devbox.Name), contentWidth-2, "…")
	marker := "  "
	if index == model.Index() {
		marker = "› "
		name = lipgloss.NewStyle().Reverse(true).Render(marker + name)
	} else {
		name = marker + name
	}

	metadata, repository := managerDevboxDetails(devbox.Devbox, d.now())
	fmt.Fprintf(
		output,
		"%s\n  %s\n  %s",
		name,
		ansi.Truncate(metadata, contentWidth-2, "…"),
		ansi.Truncate(repository, contentWidth-2, "…"),
	)
}

type devboxManager struct {
	ctx          context.Context
	client       managerClient
	createInputs *ActionInputs
	create       bool
	list         list.Model
	spinner      spinner.Model
	width        int
	height       int
	confirmation managerOperation
	operation    managerOperation
	target       string
	message      string
}

func manageDevboxes(ctx context.Context) error {
	client, err := prepareSession(ctx)
	if err != nil {
		showManagerError(err, os.Stdin, os.Stderr)
		return nil
	}

	manager := newDevboxManager(ctx, client)
	if inputs, inputErr := loadActionInputs(); inputErr == nil {
		manager.createInputs = &inputs
	}
	finalModel, err := tea.NewProgram(manager, tea.WithContext(ctx)).Run()
	if err != nil {
		return err
	}
	manager = finalModel.(devboxManager)
	if manager.create && manager.createInputs != nil {
		devboxName := namespace.NewDevboxName(manager.createInputs.Workspace)
		return openDevbox(ctx, *manager.createInputs, "new-devbox", devboxName)
	}
	return nil
}

func newDevboxManager(ctx context.Context, client managerClient) devboxManager {
	const defaultWidth, defaultHeight = 80, 24
	delegate := devboxDelegate{now: time.Now}
	devboxes := list.New(nil, delegate, defaultWidth-4, defaultHeight-6)
	devboxes.SetFilteringEnabled(false)
	devboxes.SetShowTitle(false)
	devboxes.SetShowStatusBar(false)
	devboxes.SetShowPagination(false)
	devboxes.SetShowHelp(false)
	devboxes.DisableQuitKeybindings()
	devboxes.KeyMap.NextPage.SetKeys("right", "l", "pgdown", "f")

	return devboxManager{
		ctx:       ctx,
		client:    client,
		list:      devboxes,
		spinner:   spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		width:     defaultWidth,
		height:    defaultHeight,
		operation: managerOperationRefresh,
		message:   "Loading Devboxes…",
	}
}

func (m devboxManager) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.operationCmd(managerOperationRefresh, ""))
}

func (m devboxManager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = max(message.Width, 30)
		m.height = max(message.Height, 10)
		m.resizeList()
		return m, nil

	case tea.KeyPressMsg:
		return m.updateKey(message)

	case managerOperationResult:
		m.finishOperation(message)
		return m, nil

	case spinner.TickMsg:
		if m.operation == managerOperationNone {
			return m, nil
		}
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(message)
		return m, command
	}

	var command tea.Cmd
	m.list, command = m.list.Update(message)
	return m, command
}

func (m devboxManager) updateKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := message.String()
	if m.operation != managerOperationNone {
		if key == "q" || key == "esc" || key == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	if m.confirmation != managerOperationNone {
		switch key {
		case "y":
			operation := m.confirmation
			m.confirmation = managerOperationNone
			m.operation = operation
			if selected, ok := m.selectedDevbox(); ok {
				m.target = selected.Name
			}
			if operation == managerOperationDelete {
				m.message = "Deleting " + sanitizeText(m.target) + " permanently…"
			} else {
				m.message = "Stopping " + sanitizeText(m.target) + "… This may take up to a minute."
			}
			return m, tea.Batch(m.spinner.Tick, m.operationCmd(operation, m.target))
		case "n", "q", "esc":
			m.confirmation = managerOperationNone
			m.message = "Operation cancelled"
		}
		return m, nil
	}

	switch key {
	case "c":
		if m.createInputs == nil {
			m.message = "Create a Devbox from a Herdr workspace."
			return m, nil
		}
		m.create = true
		return m, tea.Quit
	case "s":
		if m.list.SelectedItem() != nil {
			m.confirmation = managerOperationStop
			m.message = ""
		}
		return m, nil
	case "d":
		if m.list.SelectedItem() != nil {
			m.confirmation = managerOperationDelete
			m.message = ""
		}
		return m, nil
	case "r":
		m.operation = managerOperationRefresh
		m.message = "Refreshing…"
		return m, tea.Batch(m.spinner.Tick, m.operationCmd(managerOperationRefresh, ""))
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}

	var command tea.Cmd
	m.list, command = m.list.Update(message)
	return m, command
}

func (m devboxManager) operationCmd(operation managerOperation, target string) tea.Cmd {
	return func() tea.Msg {
		result := managerOperationResult{operation: operation, target: target}
		switch operation {
		case managerOperationRefresh:
			result.devboxes, result.err = m.client.List(m.ctx)
		case managerOperationStop:
			result.err = m.client.Stop(m.ctx, target)
		case managerOperationDelete:
			result.err = m.client.Delete(m.ctx, target)
		}
		return result
	}
}

func (m *devboxManager) finishOperation(result managerOperationResult) {
	m.operation = managerOperationNone
	m.target = ""
	if result.err != nil {
		m.message = result.err.Error()
		return
	}
	switch result.operation {
	case managerOperationRefresh:
		m.setDevboxes(result.devboxes)
		m.message = fmt.Sprintf("Loaded %d Devbox%s", len(result.devboxes), plural(len(result.devboxes)))
	case managerOperationStop:
		m.message = "Stopped " + sanitizeText(result.target)
	case managerOperationDelete:
		m.removeDevbox(result.target)
		m.message = "Deleted " + sanitizeText(result.target)
	}
}

func (m *devboxManager) setDevboxes(devboxes []namespace.Devbox) {
	selectedName := ""
	if selected, ok := m.selectedDevbox(); ok {
		selectedName = selected.Name
	}
	sort.SliceStable(devboxes, func(i, j int) bool {
		return devboxes[i].LastUsedAt > devboxes[j].LastUsedAt
	})
	items := make([]list.Item, len(devboxes))
	for index, devbox := range devboxes {
		items[index] = devboxItem{Devbox: devbox}
	}
	m.list.SetItems(items)
	if selectedName != "" {
		for index, item := range items {
			if item.(devboxItem).Name == selectedName {
				m.list.Select(index)
				break
			}
		}
	}
}

func (m *devboxManager) removeDevbox(name string) {
	items := m.list.Items()
	for index, item := range items {
		if item.(devboxItem).Name == name {
			m.list.RemoveItem(index)
			break
		}
	}
}

func (m devboxManager) selectedDevbox() (namespace.Devbox, bool) {
	item, ok := m.list.SelectedItem().(devboxItem)
	return item.Devbox, ok
}

func (m *devboxManager) resizeList() {
	m.list.SetSize(max(m.width-4, 26), max(m.height-6, 4))
}

func (m devboxManager) View() tea.View {
	contentWidth := max(m.width-4, 26)
	count := len(m.list.Items())
	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d Devbox%s", count, plural(count)))
	status := ansi.Truncate(sanitizeText(m.message), contentWidth, "…")
	if m.operation != managerOperationNone {
		status = m.spinner.View() + " " + status
	}
	if m.confirmation != managerOperationNone {
		if selected, ok := m.selectedDevbox(); ok {
			prompt := "Stop " + sanitizeText(selected.Name) + "? [y/N]"
			if m.confirmation == managerOperationDelete {
				prompt = "Delete " + sanitizeText(selected.Name) + " permanently? [y/N]"
			}
			status = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")).Render(
				ansi.Truncate(prompt, contentWidth, "…"),
			)
		}
	}

	footer := "↑/k up  ↓/j down  c create  s stop  d delete  r refresh  q/esc close"
	if m.operation != managerOperationNone {
		footer = "Operation in progress · q/esc close"
	} else if ansi.StringWidth(footer) > contentWidth {
		footer = "↑/k ↓/j  c create  s stop  d delete  r refresh  q close"
	}
	footer = ansi.Truncate(footer, contentWidth, "…")

	content := strings.Join([]string{
		header,
		"",
		m.list.View(),
		status,
		"",
		footer,
	}, "\n")
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func managerDevboxDetails(devbox namespace.Devbox, now time.Time) (string, string) {
	parts := make([]string, 0, 4)
	if devbox.InstanceShape.VirtualCPU > 0 {
		parts = append(parts, fmt.Sprintf("%d vCPU", devbox.InstanceShape.VirtualCPU))
	}
	if memory := devbox.InstanceShape.MemoryMegabytes; memory > 0 {
		parts = append(parts, fmt.Sprintf("%g GiB", float64(memory)/1024))
	}
	if devbox.Site != "" {
		parts = append(parts, sanitizeText(devbox.Site))
	}
	if lastUsed := relativeTime(devbox.LastUsedAt, now); lastUsed != "" {
		parts = append(parts, "used "+lastUsed)
	}
	repository := sanitizeText(devbox.Repository)
	if repository == "" && devbox.ID != "" {
		repository = "ID " + sanitizeText(devbox.ID)
	}
	return strings.Join(parts, " · "), repository
}

func relativeTime(value string, now time.Time) string {
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return ""
	}
	duration := now.Sub(timestamp)
	if duration < 0 {
		duration = 0
	}
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}

func showManagerError(err error, input io.Reader, output io.Writer) {
	fmt.Fprintf(output, "Namespace Devboxes\n\n%s\n\nPress Enter to close.", sanitizeText(err.Error()))
	_, _ = bufio.NewReader(input).ReadString('\n')
}

func sanitizeText(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return -1
		}
		return character
	}, value)
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "es"
}

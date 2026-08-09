package main

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/namespace"
)

type managerTestClient struct {
	devboxes  []namespace.Devbox
	listErr   error
	stopErr   error
	stopped   []string
	deleteErr error
	deleted   []string
}

func (c *managerTestClient) List(context.Context) ([]namespace.Devbox, error) {
	return append([]namespace.Devbox(nil), c.devboxes...), c.listErr
}

func (c *managerTestClient) Stop(_ context.Context, name string) error {
	c.stopped = append(c.stopped, name)
	return c.stopErr
}

func (c *managerTestClient) Delete(_ context.Context, name string) error {
	c.deleted = append(c.deleted, name)
	return c.deleteErr
}

func managerWithDevboxes(t *testing.T, client *managerTestClient) devboxManager {
	t.Helper()
	manager := newDevboxManager(context.Background(), client)
	manager.finishOperation(managerOperationResult{
		operation: managerOperationRefresh,
		devboxes:  client.devboxes,
		err:       client.listErr,
	})
	return manager
}

func pressManagerKey(t *testing.T, manager devboxManager, code rune) (devboxManager, tea.Cmd) {
	t.Helper()
	message := tea.KeyPressMsg{Code: code}
	if code >= ' ' && code <= '~' {
		message.Text = string(code)
	}
	updated, command := manager.Update(message)
	return updated.(devboxManager), command
}

func TestManagerRefreshSortsByLastUsed(t *testing.T) {
	client := &managerTestClient{devboxes: []namespace.Devbox{
		{Name: "older", LastUsedAt: "2026-08-08T12:00:00Z"},
		{Name: "newer", LastUsedAt: "2026-08-09T12:00:00Z"},
	}}
	manager := managerWithDevboxes(t, client)

	require.Equal(t, "newer", manager.list.Items()[0].(devboxItem).Name)
	require.Equal(t, "older", manager.list.Items()[1].(devboxItem).Name)
	require.Equal(t, "Loaded 2 Devboxes", manager.message)
}

func TestManagerUsesBubblesNavigationAndStopsWithConfirmation(t *testing.T) {
	client := &managerTestClient{devboxes: []namespace.Devbox{
		{Name: "first"},
		{Name: "second"},
	}}
	manager := managerWithDevboxes(t, client)

	updated, _ := manager.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	manager = updated.(devboxManager)
	require.Equal(t, "second", manager.list.SelectedItem().(devboxItem).Name)

	manager, _ = pressManagerKey(t, manager, 's')
	require.Equal(t, managerOperationStop, manager.confirmation)
	manager, command := pressManagerKey(t, manager, 'y')
	require.NotNil(t, command)
	require.Equal(t, managerOperationStop, manager.operation)
	require.Equal(t, "Stopping second… This may take up to a minute.", manager.message)

	result := manager.operationCmd(manager.operation, manager.target)().(managerOperationResult)
	manager.finishOperation(result)
	require.Equal(t, []string{"second"}, client.stopped)
	require.Equal(t, "Stopped second", manager.message)
}

func TestManagerReportsStopFailure(t *testing.T) {
	client := &managerTestClient{
		devboxes: []namespace.Devbox{{Name: "demo"}},
		stopErr:  errors.New("stop failed"),
	}
	manager := managerWithDevboxes(t, client)
	manager.finishOperation(managerOperationResult{
		operation: managerOperationStop,
		target:    "demo",
		err:       client.stopErr,
	})

	require.Equal(t, "stop failed", manager.message)
}

func TestManagerDeletesWithPermanentConfirmation(t *testing.T) {
	client := &managerTestClient{devboxes: []namespace.Devbox{
		{Name: "first"},
		{Name: "second"},
	}}
	manager := managerWithDevboxes(t, client)
	manager.list.Select(1)

	manager, _ = pressManagerKey(t, manager, 'd')
	require.Equal(t, managerOperationDelete, manager.confirmation)
	require.Contains(t, manager.View().Content, "Delete second permanently? [y/N]")

	manager, _ = pressManagerKey(t, manager, 'y')
	result := manager.operationCmd(manager.operation, manager.target)().(managerOperationResult)
	manager.finishOperation(result)

	require.Equal(t, []string{"second"}, client.deleted)
	require.Equal(t, "Deleted second", manager.message)
	require.Len(t, manager.list.Items(), 1)
	require.Equal(t, "first", manager.list.Items()[0].(devboxItem).Name)
}

func TestManagerViewShowsSafeBoundedDevboxCards(t *testing.T) {
	client := &managerTestClient{devboxes: []namespace.Devbox{{
		Name:       "demo\x1b[31m" + strings.Repeat("-long", 20),
		Repository: "github.com/acme/demo" + strings.Repeat("/long", 20),
		Site:       "zrh",
		LastUsedAt: "2026-08-09T10:00:00Z",
		InstanceShape: namespace.InstanceShape{
			VirtualCPU:      8,
			MemoryMegabytes: 16384,
		},
	}}}
	manager := managerWithDevboxes(t, client)
	manager.list.SetDelegate(devboxDelegate{now: func() time.Time {
		return time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	}})
	manager.width = 50
	manager.height = 24
	manager.resizeList()

	screen := manager.View().Content
	plain := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`).ReplaceAllString(screen, "")
	require.Contains(t, plain, "demo[31m")
	require.NotContains(t, screen, "demo\x1b[31m")
	require.Contains(t, plain, "8 vCPU · 16 GiB · zrh · used 2h ago")
	require.Contains(t, plain, "Loaded 1 Devbox")
	for _, line := range strings.Split(screen, "\n") {
		require.LessOrEqual(t, ansi.StringWidth(line), 46, line)
	}
}

func TestManagerConfirmationCancelDoesNotQuit(t *testing.T) {
	manager := managerWithDevboxes(t, &managerTestClient{
		devboxes: []namespace.Devbox{{Name: "demo"}},
	})
	manager, _ = pressManagerKey(t, manager, 's')

	manager, command := pressManagerKey(t, manager, 'q')
	require.Nil(t, command)
	require.Equal(t, managerOperationNone, manager.confirmation)
	require.Equal(t, "Operation cancelled", manager.message)

	_, command = pressManagerKey(t, manager, 'q')
	require.NotNil(t, command)
}

func TestManagerCreateClosesPopupAndRequestsNewDevbox(t *testing.T) {
	manager := managerWithDevboxes(t, &managerTestClient{})
	manager.createInputs = &ActionInputs{
		PluginExecutable: "/tmp/herdr-namespace",
		Workspace:        "/tmp/demo",
	}

	manager, command := pressManagerKey(t, manager, 'c')

	require.True(t, manager.create)
	require.NotNil(t, command)
	require.Contains(t, manager.View().Content, "c create")
}

func TestManagerCreateRequiresWorkspaceContext(t *testing.T) {
	manager := managerWithDevboxes(t, &managerTestClient{})

	manager, command := pressManagerKey(t, manager, 'c')

	require.False(t, manager.create)
	require.Nil(t, command)
	require.Equal(t, "Create a Devbox from a Herdr workspace.", manager.message)
}

func TestManagerRemainsResponsiveDuringOperation(t *testing.T) {
	manager := newDevboxManager(context.Background(), &managerTestClient{})
	manager.operation = managerOperationStop
	manager.target = "demo"
	manager.message = "Stopping demo… This may take up to a minute."

	updated, command := manager.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	manager = updated.(devboxManager)
	require.Nil(t, command)
	require.Contains(t, manager.View().Content, "Operation in progress")

	_, command = pressManagerKey(t, manager, 'q')
	require.NotNil(t, command)
}

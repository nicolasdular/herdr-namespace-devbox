package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/namespace"
)

func TestCreateFormChangesStateLifecycle(t *testing.T) {
	form := newDevboxCreateForm(DevboxCreatePlan{
		Repository: &namespace.Repository{URL: "github.com/acme/demo"},
	})

	require.True(t, form.beginChangesInspection())
	require.Equal(t, changesLoading, form.ChangesState)
	require.False(t, form.beginChangesInspection())

	form.finishChangesInspection(LocalChangesInfo{BaseCommit: "abc123", FileCount: 2}, nil)
	require.Equal(t, changesAvailable, form.ChangesState)
	require.Equal(t, 2, form.TrackedChanges)
	require.True(t, form.canUploadLocalChanges())
}

func TestCreateFormDisablesUploadAfterInspectionFailure(t *testing.T) {
	form := devboxCreateForm{
		Plan: DevboxCreatePlan{
			Repository: &namespace.Repository{URL: "github.com/acme/demo"},
		},
		ChangesState:       changesAvailable,
		TrackedChanges:     2,
		UploadLocalChanges: true,
	}

	form.finishChangesInspection(LocalChangesInfo{}, errors.New("Git failed"))
	require.Equal(t, changesUnavailable, form.ChangesState)
	require.False(t, form.UploadLocalChanges)
	require.False(t, form.canUploadLocalChanges())
}

func TestCreateFormSkipsInspectionWithoutRepository(t *testing.T) {
	form := newDevboxCreateForm(DevboxCreatePlan{})

	require.False(t, form.beginChangesInspection())
	require.Equal(t, changesNotLoaded, form.ChangesState)
	require.Equal(t, "none", createPlanRepository(form.Plan))
}

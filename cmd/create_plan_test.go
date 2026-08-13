package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/namespace"
)

func TestCreatePlanAppliesFormValuesToSpec(t *testing.T) {
	spec := namespace.Spec{
		Name:       "original",
		Image:      "original:image",
		Size:       "s",
		Repository: &namespace.Repository{URL: "github.com/acme/original"},
	}
	plan := DevboxCreatePlan{
		Name:       "edited",
		Image:      "custom:image",
		Size:       "xl",
		Site:       "zrh",
		Repository: &namespace.Repository{URL: "github.com/acme/edited", Ref: "feature"},
	}

	got := plan.apply(spec)

	require.Equal(t, "edited", got.Name)
	require.Equal(t, "custom:image", got.Image)
	require.Equal(t, "xl", got.Size)
	require.Equal(t, "zrh", *got.Site)
	require.Equal(t, "github.com/acme/edited", got.Repository.URL)
	require.Equal(t, "feature", got.Repository.Ref)
}

func TestCreatePlanCanDisableRepositoryAndUseAutomaticSite(t *testing.T) {
	site := "zrh"
	spec := namespace.Spec{
		Repository: &namespace.Repository{URL: "github.com/acme/original"},
		Site:       &site,
	}

	got := (DevboxCreatePlan{Name: "edited", Site: "automatic"}).apply(spec)

	require.True(t, got.Repository.Disabled)
	require.Empty(t, got.Repository.URL)
	require.Nil(t, got.Site)
}

func TestCreatePlanCanCrossShellBoundaryAsURLSafeBase64(t *testing.T) {
	plan := DevboxCreatePlan{
		Name:       "herdr-demo-123",
		Repository: &namespace.Repository{URL: "github.com/acme/demo", Ref: "main"},
		Image:      "custom:image",
		Size:       "m",
		Site:       "automatic",
	}
	contents, err := json.Marshal(plan)
	require.NoError(t, err)

	encoded := base64.RawURLEncoding.EncodeToString(contents)
	require.NotContains(t, encoded, "{")
	require.NotContains(t, encoded, "\"")
	require.NotContains(t, encoded, ",")

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var got DevboxCreatePlan
	require.NoError(t, json.Unmarshal(decoded, &got))
	require.Equal(t, plan, got)
}

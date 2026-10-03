package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffKeysReportsAddedAndRemoved(t *testing.T) {
	t.Parallel()

	recorded := map[string]struct{}{"minutes": {}, "totalCost": {}}
	live := map[string]struct{}{"minutes": {}, "newField": {}}
	added, removed := diffKeys(recorded, live)
	require.Equal(t, []string{"newField"}, added)
	require.Equal(t, []string{"totalCost"}, removed)
}

func TestPlainErrorRejectsJSON(t *testing.T) {
	t.Parallel()

	err := checkPlainError(
		400,
		"application/json",
		[]byte(`{"code":400}`),
		[]byte("Invalid 'window' parameter: illegal window: \n"),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plain")
}

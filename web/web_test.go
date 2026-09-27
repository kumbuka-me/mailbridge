package web

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssets(t *testing.T) {
	t.Parallel()

	index, err := fs.ReadFile(Assets, "index.gohtml")
	require.NoError(t, err)
	assert.Contains(t, string(index), "<title>mailbridge</title>")
	assert.Contains(t, string(index), `/assets/favicon.svg`)

	for _, name := range []string{
		"favicon.svg",
		"favicon-16x16.png",
		"favicon-32x32.png",
		"apple-touch-icon.png",
		"mailbridge.png",
	} {
		file, err := Assets.Open(name)
		require.NoError(t, err, name)
		require.NoError(t, file.Close(), name)
	}
}

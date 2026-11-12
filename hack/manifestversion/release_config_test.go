package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type goreleaserConfig struct {
	Before struct {
		Hooks []string `yaml:"hooks"`
	} `yaml:"before"`
	Archives []struct {
		Files []yaml.Node `yaml:"files"`
	} `yaml:"archives"`
}

type archiveFile struct {
	Src         string `yaml:"src"`
	Dst         string `yaml:"dst"`
	StripParent bool   `yaml:"strip_parent"`
}

// Core's plugin validate compares the manifest version with the install
// directory, which a registry install names after the release tag.
func TestGoreleaserArchivesStampedManifest(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", ".goreleaser.yaml"))
	require.NoError(t, err)
	var cfg goreleaserConfig
	require.NoError(t, yaml.Unmarshal(raw, &cfg))

	stampHook := "go run ./hack/manifestversion -in plugin.manifest.json -out " +
		stampedManifest + " -version v{{ .Version }}"
	assert.Contains(t, cfg.Before.Hooks, stampHook)

	require.NotEmpty(t, cfg.Archives)
	for _, archive := range cfg.Archives {
		var stamped int
		for _, node := range archive.Files {
			if node.Kind == yaml.ScalarNode {
				assert.NotEqual(t, "plugin.manifest.json", node.Value, "archive ships the unstamped manifest")
				continue
			}
			var file archiveFile
			require.NoError(t, node.Decode(&file))
			assert.NotEqual(t, "plugin.manifest.json", file.Src, "archive ships the unstamped manifest")
			if file.Src == stampedManifest {
				stamped++
				assert.True(t, file.StripParent, "stamped manifest must sit at the archive root")
				assert.Empty(t, strings.TrimSpace(file.Dst))
			}
		}
		assert.Equal(t, 1, stamped, "archive must ship the stamped manifest once")
	}
}

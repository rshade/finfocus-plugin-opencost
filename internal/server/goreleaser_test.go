package server

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"gopkg.in/yaml.v3"
)

// GoReleaserConfig represents the minimal goreleaser configuration structure needed for testing.
type GoReleaserConfig struct {
	ProjectName string `yaml:"project_name"`
	Archives    []struct {
		NameTemplate string `yaml:"name_template"`
	} `yaml:"archives"`
}

// TestGoReleaserAssetNaming verifies that .goreleaser.yaml produces asset names
// that match the finfocus installer's pattern rules from
// github.com/rshade/finfocus/internal/registry/github.go buildAssetPatterns (lines 488-560).
func TestGoReleaserAssetNaming(t *testing.T) {
	// Find repo root from test source location
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Could not determine test source location")
	}

	// Walk up from this test file to repo root (testfile is at internal/server/goreleaser_test.go)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	configPath := filepath.Join(repoRoot, ".goreleaser.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read .goreleaser.yaml at %s: %v", configPath, err)
	}

	var config GoReleaserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse .goreleaser.yaml: %v", err)
	}

	if config.ProjectName == "" {
		t.Fatal("project_name not found in .goreleaser.yaml")
	}

	if len(config.Archives) == 0 {
		t.Fatal("No archives section found in .goreleaser.yaml")
	}

	nameTemplate := config.Archives[0].NameTemplate
	if nameTemplate == "" {
		t.Fatal("name_template not found in archives section")
	}

	// Parse template with title function
	tmpl, err := template.New("asset").Funcs(template.FuncMap{
		"title": strings.Title,
	}).Parse(nameTemplate)
	if err != nil {
		t.Fatalf("Failed to parse name_template: %v", err)
	}

	// Test cases covering all relevant OS/arch combinations
	testCases := []struct {
		goos        string
		goarch      string
		version     string
		description string
	}{
		{"linux", "amd64", "v0.1.0", "Linux x86_64 with v-prefix"},
		{"linux", "amd64", "0.1.0", "Linux x86_64 without v-prefix"},
		{"linux", "arm64", "v0.1.0", "Linux arm64 with v-prefix"},
		{"linux", "arm64", "0.1.0", "Linux arm64 without v-prefix"},
		{"darwin", "amd64", "v0.1.0", "Darwin x86_64 with v-prefix"},
		{"darwin", "amd64", "0.1.0", "Darwin x86_64 without v-prefix"},
		{"darwin", "arm64", "v0.1.0", "Darwin arm64 with v-prefix"},
		{"darwin", "arm64", "0.1.0", "Darwin arm64 without v-prefix"},
		{"windows", "amd64", "v0.1.0", "Windows amd64 with v-prefix"},
		{"windows", "amd64", "0.1.0", "Windows amd64 without v-prefix"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ext := ".tar.gz"
			if tc.goos == "windows" {
				ext = ".zip"
			}

			values := map[string]string{
				"ProjectName": config.ProjectName,
				"Version":     tc.version,
				"Os":          tc.goos,
				"Arch":        tc.goarch,
			}

			var buf strings.Builder
			if err := tmpl.Execute(&buf, values); err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			renderedName := buf.String() + ext
			t.Logf("Rendered: %s", renderedName)

			if !matchesInstallerPattern(renderedName, config.ProjectName, tc.version, tc.goos, tc.goarch) {
				t.Errorf("Rendered name %q does not match installer's expected patterns for %s/%s",
					renderedName, tc.goos, tc.goarch)
			}
		})
	}
}

// matchesInstallerPattern checks if a rendered asset name matches one of the patterns
// the finfocus installer generates. This mirrors buildAssetPatterns logic from
// github.com/rshade/finfocus/internal/registry/github.go (lines 491-560).
func matchesInstallerPattern(assetName, projectName, version, goos, goarch string) bool {
	// Possible OS names per installer
	osNames := []string{goos, strings.Title(goos)}
	if goos == "darwin" {
		osNames = append(osNames, "Darwin", "macos", "macOS", "MacOS")
	}

	// Possible architecture names per installer
	archNames := []string{goarch}
	if goarch == "amd64" {
		archNames = append(archNames, "x86_64", "X86_64", "AMD64")
	}
	if goarch == "arm64" {
		archNames = append(archNames, "ARM64", "aarch64", "AARCH64")
	}

	// Possible versions (with and without leading v)
	versions := []string{version}
	if strings.HasPrefix(version, "v") {
		versions = append(versions, strings.TrimPrefix(version, "v"))
	} else {
		versions = append(versions, "v"+version)
	}

	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	// Check if asset matches any combination of installer's expected patterns
	for _, ver := range versions {
		for _, osName := range osNames {
			for _, arch := range archNames {
				expectedPattern := fmt.Sprintf("%s_%s_%s_%s%s", projectName, ver, osName, arch, ext)
				if assetName == expectedPattern {
					return true
				}
			}
		}
	}

	return false
}

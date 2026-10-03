package version

import (
	"fmt"
	"runtime"
)

// Version identifies the plugin build. GoReleaser and the Makefile override it with ldflags.
//
//nolint:gochecknoglobals // ldflags -X can only set package-level variables
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
	GitState  = "unknown"
)

// defaultVersionInfo provides default version information.
func defaultVersionInfo() (string, string, string, string, string) {
	return Version, BuildDate, GitCommit, GitBranch, GitState
}

// Info contains version information.
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
	GitBranch string `json:"gitBranch"`
	GitState  string `json:"gitState"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// GetVersionInfo returns the complete version information.
func GetVersionInfo() Info {
	version, buildDate, gitCommit, gitBranch, gitState := defaultVersionInfo()
	return Info{
		Version:   version,
		BuildDate: buildDate,
		GitCommit: gitCommit,
		GitBranch: gitBranch,
		GitState:  gitState,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string.
func String() string {
	info := GetVersionInfo()
	return fmt.Sprintf("v%s (%s, %s, %s)",
		info.Version,
		info.GitCommit,
		info.BuildDate,
		info.Platform)
}

// FullString returns a detailed version string.
func FullString() string {
	info := GetVersionInfo()
	return fmt.Sprintf(
		"Version: %s\nBuild Date: %s\nGit Commit: %s\nGit Branch: %s\nGit State: %s\nGo Version: %s\nPlatform: %s",
		info.Version,
		info.BuildDate,
		info.GitCommit,
		info.GitBranch,
		info.GitState,
		info.GoVersion,
		info.Platform)
}

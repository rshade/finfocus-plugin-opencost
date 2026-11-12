// Command manifestversion writes a copy of plugin.manifest.json whose
// top-level "version" is the release version. FinFocus core's plugin validate
// requires that value to equal the install directory, which a registry install
// names after the release tag (v0.1.2). GoReleaser and make install run it.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const stampedManifest = "build/release/plugin.manifest.json"

var versionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+([-+][0-9A-Za-z.+-]+)?$`)

func main() {
	in := flag.String("in", "plugin.manifest.json", "checked-in manifest to read")
	out := flag.String("out", stampedManifest, "stamped manifest to write")
	version := flag.String("version", "", "release version with its v prefix, for example v0.1.2")
	flag.Parse()
	if err := run(*in, *out, *version); err != nil {
		fmt.Fprintf(os.Stderr, "manifestversion: %v\n", err)
		os.Exit(1)
	}
}

func run(in, out, version string) error {
	manifest, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	stamped, err := stamp(manifest, version)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return err
	}
	//nolint:gosec // G306: the manifest ships in release archives and must stay world-readable
	return os.WriteFile(out, stamped, 0o644)
}

// stamp returns manifest with its top-level "version" string replaced by
// version. Every other byte, including key order and formatting, is kept.
func stamp(manifest []byte, version string) ([]byte, error) {
	if !versionPattern.MatchString(version) {
		return nil, fmt.Errorf("version %q must look like v1.2.3", version)
	}
	start, end, err := versionSpan(manifest)
	if err != nil {
		return nil, err
	}
	quoted, err := json.Marshal(version)
	if err != nil {
		return nil, err
	}
	stamped := make([]byte, 0, len(manifest)+len(quoted))
	stamped = append(stamped, manifest[:start]...)
	stamped = append(stamped, quoted...)
	return append(stamped, manifest[end:]...), nil
}

// versionSpan returns the byte range of the top-level "version" value.
func versionSpan(manifest []byte) (int, int, error) {
	dec := json.NewDecoder(bytes.NewReader(manifest))
	tok, err := dec.Token()
	if err != nil {
		return 0, 0, fmt.Errorf("read manifest: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return 0, 0, errors.New("manifest is not a JSON object")
	}
	start, end := -1, -1
	for dec.More() {
		key, keyErr := dec.Token()
		if keyErr != nil {
			return 0, 0, fmt.Errorf("read manifest: %w", keyErr)
		}
		var value json.RawMessage
		if valueErr := dec.Decode(&value); valueErr != nil {
			return 0, 0, fmt.Errorf("read manifest: %w", valueErr)
		}
		if key != "version" {
			continue
		}
		if start >= 0 {
			return 0, 0, errors.New(`manifest has more than one top-level "version"`)
		}
		if value[0] != '"' {
			return 0, 0, errors.New(`manifest "version" is not a string`)
		}
		end = int(dec.InputOffset())
		start = end - len(value)
	}
	if _, err = dec.Token(); err != nil {
		return 0, 0, fmt.Errorf("read manifest: %w", err)
	}
	if _, err = dec.Token(); !errors.Is(err, io.EOF) {
		return 0, 0, errors.New("manifest has trailing data after the object")
	}
	if start < 0 {
		return 0, 0, errors.New(`manifest has no top-level "version"`)
	}
	return start, end, nil
}

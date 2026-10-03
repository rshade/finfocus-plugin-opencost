// Command livekeyset compares a live OpenCost /allocation key set and its
// HTTP 400 plain-text errors with testdata/opencost-real. It never writes fixtures.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	queryTimeout = 3 * time.Minute
	maxBodyBytes = 32 << 20
)

func main() {
	base := flag.String("base", "", "OpenCost base URL")
	fixtures := flag.String("fixtures", "testdata/opencost-real", "read-only recorded fixtures")
	responses := flag.String("responses", "", "allocation JSON directory to compare instead of -base")
	flag.Parse()
	if err := run(*base, *fixtures, *responses); err != nil {
		fmt.Fprintf(os.Stderr, "live drift: %v\n", err)
		os.Exit(1)
	}
}

func run(base, fixtures, responses string) error {
	if (base == "") == (responses == "") {
		return errors.New("set exactly one of -base or -responses")
	}
	recorded, err := keysFromDir(fixtures)
	if err != nil {
		return err
	}
	var live map[string]struct{}
	var errorErr error
	if responses != "" {
		live, err = keysFromDir(responses)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		live, err = keysFromBase(ctx, base)
		if err == nil {
			errorErr = errorsFromBase(ctx, base, fixtures)
		}
	}
	if err != nil {
		return err
	}
	added, removed := diffKeys(recorded, live)
	fmt.Fprintf(os.Stdout, "added: %s\nremoved: %s\n", formatKeys(added), formatKeys(removed))
	if len(added)+len(removed) > 0 {
		return errors.New("allocation keys differ from the recorded fixtures")
	}
	if errorErr != nil {
		return errorErr
	}
	fmt.Fprintf(os.Stdout, "keys match (%d)\n", len(recorded))
	return nil
}

func keysFromBase(ctx context.Context, base string) (map[string]struct{}, error) {
	paths := []string{
		"/allocation?window=60m&aggregate=namespace&includeIdle=false",
		"/allocation?window=60m&aggregate=namespace&includeIdle=true",
	}
	keys := map[string]struct{}{}
	for _, path := range paths {
		status, _, body, err := get(ctx, base, path)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("%s: status %d", path, status)
		}
		found, err := allocationKeys(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for key := range found {
			keys[key] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("live allocation has no object keys")
	}
	return keys, nil
}

func errorsFromBase(ctx context.Context, base, fixtures string) error {
	probes := []struct {
		path string
		file string
	}{
		{path: "/allocation?window=notawindow", file: "error-bad-window.json"},
		{path: "/allocation", file: "error-no-window.json"},
	}
	for _, probe := range probes {
		status, contentType, body, err := get(ctx, base, probe.path)
		if err != nil {
			return err
		}
		fixture, err := os.ReadFile(filepath.Join(fixtures, probe.file))
		if err != nil {
			return err
		}
		if err = checkPlainError(status, contentType, body, fixture); err != nil {
			return fmt.Errorf("%s: %w", probe.file, err)
		}
	}
	return nil
}

func get(ctx context.Context, base, path string) (int, string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+path, nil)
	if err != nil {
		return 0, "", nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return 0, "", nil, err
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

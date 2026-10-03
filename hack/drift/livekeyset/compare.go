package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func diffKeys(recorded, live map[string]struct{}) ([]string, []string) {
	var added, removed []string
	for key := range live {
		if _, ok := recorded[key]; !ok {
			added = append(added, key)
		}
	}
	for key := range recorded {
		if _, ok := live[key]; !ok {
			removed = append(removed, key)
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	return added, removed
}

func formatKeys(keys []string) string {
	if len(keys) == 0 {
		return "none"
	}
	return strings.Join(keys, ", ")
}

func checkPlainError(status int, contentType string, body, fixture []byte) error {
	if status != http.StatusBadRequest {
		return fmt.Errorf("status %d, want %d", status, http.StatusBadRequest)
	}
	media := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if media != "text/plain" {
		return fmt.Errorf("content type %q is not plain text", contentType)
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return errors.New("error body is not plain text")
	}
	if !bytes.Equal(body, fixture) {
		return errors.New("error body differs from the recorded fixture")
	}
	return nil
}

func keysFromDir(dir string) (map[string]struct{}, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "allocation-*.json"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no allocation files in %s", dir)
	}
	keys := map[string]struct{}{}
	for _, name := range matches {
		body, readErr := os.ReadFile(name)
		if readErr != nil {
			return nil, readErr
		}
		found, keyErr := allocationKeys(body)
		if keyErr != nil {
			return nil, fmt.Errorf("%s: %w", name, keyErr)
		}
		for key := range found {
			keys[key] = struct{}{}
		}
	}
	return keys, nil
}

func allocationKeys(body []byte) (map[string]struct{}, error) {
	var envelope struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	for _, step := range envelope.Data {
		for _, raw := range step {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				return nil, err
			}
			for key := range object {
				keys[key] = struct{}{}
			}
		}
	}
	return keys, nil
}

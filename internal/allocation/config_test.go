package allocation //nolint:testpackage // Package name intentionally matches implementation for simplicity

import (
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		BaseURL:       "http://localhost:9090",
		APIToken:      "test-token",
		DefaultWindow: "30d",
		Timeout:       30 * time.Second,
		TLSSkipVerify: true,
	}

	if cfg.BaseURL != "http://localhost:9090" {
		t.Errorf("Expected BaseURL %s, got %s", "http://localhost:9090", cfg.BaseURL)
	}

	if cfg.APIToken != "test-token" {
		t.Errorf("Expected APIToken %s, got %s", "test-token", cfg.APIToken)
	}

	if cfg.DefaultWindow != "30d" {
		t.Errorf("Expected DefaultWindow %s, got %s", "30d", cfg.DefaultWindow)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout %v, got %v", 30*time.Second, cfg.Timeout)
	}

	if !cfg.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true")
	}
}

func TestLoadConfigFromEnvOrFile_EnvironmentOnly(t *testing.T) {
	// Set environment variables
	t.Setenv("KUBECOST_BASE_URL", "http://env-test:9090")
	t.Setenv("KUBECOST_API_TOKEN", "env-token")
	t.Setenv("KUBECOST_DEFAULT_WINDOW", "7d")
	t.Setenv("KUBECOST_TIMEOUT", "60s")
	t.Setenv("KUBECOST_TLS_SKIP_VERIFY", "true")
	defer func() {
		os.Unsetenv("KUBECOST_BASE_URL")
		os.Unsetenv("KUBECOST_API_TOKEN")
		os.Unsetenv("KUBECOST_DEFAULT_WINDOW")
		os.Unsetenv("KUBECOST_TIMEOUT")
		os.Unsetenv("KUBECOST_TLS_SKIP_VERIFY")
	}()

	cfg, err := LoadConfigFromEnvOrFile("")
	if err != nil {
		t.Fatalf("LoadConfigFromEnvOrFile failed: %v", err)
	}

	if cfg.BaseURL != "http://env-test:9090" {
		t.Errorf("Expected BaseURL %s, got %s", "http://env-test:9090", cfg.BaseURL)
	}

	if cfg.APIToken != "env-token" {
		t.Errorf("Expected APIToken %s, got %s", "env-token", cfg.APIToken)
	}

	if cfg.DefaultWindow != "7d" {
		t.Errorf("Expected DefaultWindow %s, got %s", "7d", cfg.DefaultWindow)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout %v, got %v", 60*time.Second, cfg.Timeout)
	}

	if !cfg.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true")
	}
}

func TestLoadConfigFromEnvOrFile_WithFile(t *testing.T) {
	// Create a temporary config file
	configContent := `
baseUrl: http://file-test:9090
apiToken: file-token
defaultWindow: 14d
timeout: 45s
tlsSkipVerify: false
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "kubecost-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config content: %v", writeErr)
	}
	tmpFile.Close()

	// Set environment variables (should be overridden by file)
	t.Setenv("KUBECOST_BASE_URL", "http://env-test:9090")
	t.Setenv("KUBECOST_API_TOKEN", "env-token")
	defer func() {
		os.Unsetenv("KUBECOST_BASE_URL")
		os.Unsetenv("KUBECOST_API_TOKEN")
	}()

	cfg, err := LoadConfigFromEnvOrFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfigFromEnvOrFile failed: %v", err)
	}

	// File values should override environment values
	if cfg.BaseURL != "http://file-test:9090" {
		t.Errorf("Expected BaseURL %s, got %s", "http://file-test:9090", cfg.BaseURL)
	}

	if cfg.APIToken != "file-token" {
		t.Errorf("Expected APIToken %s, got %s", "file-token", cfg.APIToken)
	}

	if cfg.DefaultWindow != "14d" {
		t.Errorf("Expected DefaultWindow %s, got %s", "14d", cfg.DefaultWindow)
	}

	if cfg.Timeout != 45*time.Second {
		t.Errorf("Expected Timeout %v, got %v", 45*time.Second, cfg.Timeout)
	}

	if cfg.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be false")
	}
}

func TestLoadConfigFromEnvOrFile_FileNotFound(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("KUBECOST_BASE_URL")
	os.Unsetenv("KUBECOST_API_TOKEN")
	os.Unsetenv("KUBECOST_DEFAULT_WINDOW")
	os.Unsetenv("KUBECOST_TIMEOUT")
	os.Unsetenv("KUBECOST_TLS_SKIP_VERIFY")

	cfg, err := LoadConfigFromEnvOrFile("/nonexistent/file.yaml")
	if err != nil {
		t.Fatalf("LoadConfigFromEnvOrFile should not fail for nonexistent file: %v", err)
	}

	// Should use defaults
	if cfg.BaseURL != "" {
		t.Errorf("Expected empty BaseURL, got %s", cfg.BaseURL)
	}

	if cfg.APIToken != "" {
		t.Errorf("Expected empty APIToken, got %s", cfg.APIToken)
	}

	if cfg.DefaultWindow != "30d" {
		t.Errorf("Expected DefaultWindow %s, got %s", "30d", cfg.DefaultWindow)
	}

	if cfg.Timeout != 15*time.Second {
		t.Errorf("Expected Timeout %v, got %v", 15*time.Second, cfg.Timeout)
	}

	if cfg.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be false")
	}
}

func TestGetenvDefault(t *testing.T) {
	// Test with environment variable set
	t.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := getenvDefault("TEST_VAR", "default-value")
	if result != "test-value" {
		t.Errorf("Expected %s, got %s", "test-value", result)
	}

	// Test with environment variable not set
	result = getenvDefault("NONEXISTENT_VAR", "default-value")
	if result != "default-value" {
		t.Errorf("Expected %s, got %s", "default-value", result)
	}

	// Test with empty environment variable
	t.Setenv("EMPTY_VAR", "")
	result = getenvDefault("EMPTY_VAR", "default-value")
	if result != "default-value" {
		t.Errorf("Expected %s, got %s", "default-value", result)
	}
}

func TestGetenvDuration(t *testing.T) {
	// Test with valid duration
	t.Setenv("TEST_DURATION", "30s")
	defer os.Unsetenv("TEST_DURATION")

	result := getenvDuration("TEST_DURATION", 15*time.Second)
	if result != 30*time.Second {
		t.Errorf("Expected %v, got %v", 30*time.Second, result)
	}

	// Test with invalid duration
	t.Setenv("INVALID_DURATION", "invalid")
	result = getenvDuration("INVALID_DURATION", 15*time.Second)
	if result != 15*time.Second {
		t.Errorf("Expected %v, got %v", 15*time.Second, result)
	}

	// Test with environment variable not set
	result = getenvDuration("NONEXISTENT_DURATION", 15*time.Second)
	if result != 15*time.Second {
		t.Errorf("Expected %v, got %v", 15*time.Second, result)
	}

	// Test with complex duration
	t.Setenv("COMPLEX_DURATION", "1h30m")
	result = getenvDuration("COMPLEX_DURATION", 15*time.Second)
	expected := 1*time.Hour + 30*time.Minute
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear all environment variables
	os.Unsetenv("KUBECOST_BASE_URL")
	os.Unsetenv("KUBECOST_API_TOKEN")
	os.Unsetenv("KUBECOST_DEFAULT_WINDOW")
	os.Unsetenv("KUBECOST_TIMEOUT")
	os.Unsetenv("KUBECOST_TLS_SKIP_VERIFY")

	cfg, err := LoadConfigFromEnvOrFile("")
	if err != nil {
		t.Fatalf("LoadConfigFromEnvOrFile failed: %v", err)
	}

	// Check defaults
	if cfg.BaseURL != "" {
		t.Errorf("Expected empty BaseURL, got %s", cfg.BaseURL)
	}

	if cfg.APIToken != "" {
		t.Errorf("Expected empty APIToken, got %s", cfg.APIToken)
	}

	if cfg.DefaultWindow != "30d" {
		t.Errorf("Expected DefaultWindow %s, got %s", "30d", cfg.DefaultWindow)
	}

	if cfg.Timeout != 15*time.Second {
		t.Errorf("Expected Timeout %v, got %v", 15*time.Second, cfg.Timeout)
	}

	if cfg.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be false")
	}
}

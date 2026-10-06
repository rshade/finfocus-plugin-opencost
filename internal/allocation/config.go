package allocation

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultTimeoutDuration = 15 * time.Second

const (
	// ProfileOpenCost calls GET /allocation and does not require a token.
	ProfileOpenCost = "opencost"
	// ProfileKubecost calls GET /model/allocation and sends a bearer token.
	ProfileKubecost = "kubecost"
)

type Config struct {
	BaseURL string `yaml:"baseUrl"`
	// APIToken is read from KUBECOST_API_TOKEN. YAML apiToken is ignored.
	APIToken      string        `yaml:"-"`
	Profile       string        `yaml:"profile"`
	DefaultWindow string        `yaml:"defaultWindow"` // e.g. "30d"
	Timeout       time.Duration `yaml:"timeout"`
	TLSSkipVerify bool          `yaml:"tlsSkipVerify"`
	// Prediction API specific configuration
	ClusterID        string `yaml:"clusterId"`
	DefaultNamespace string `yaml:"defaultNamespace"`
	PredictionWindow string `yaml:"predictionWindow"` // e.g. "2d" (default for prediction API)
	// Currency is the ISO 4217 code from pricing config. Empty is not USD.
	Currency string `yaml:"currency"`
	// CacheTTL is how long a successful allocation URL is reused.
	// Zero uses the default. A negative value disables the cache.
	CacheTTL time.Duration `yaml:"cacheTTL"`
	// RequestsPerSecond limits outbound allocation requests. Zero uses the default.
	RequestsPerSecond float64 `yaml:"requestsPerSecond"`
	// RateBurst is how many outbound requests may run at once. Zero uses the default.
	RateBurst int `yaml:"rateBurst"`
}

func LoadConfigFromEnvOrFile(path string) (Config, error) {
	timeout, err := getenvDuration("KUBECOST_TIMEOUT", defaultTimeoutDuration)
	if err != nil {
		return Config{}, err
	}
	skipVerify, err := getenvBool("KUBECOST_TLS_SKIP_VERIFY")
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		BaseURL:          os.Getenv("KUBECOST_BASE_URL"),
		APIToken:         os.Getenv("KUBECOST_API_TOKEN"),
		Profile:          os.Getenv("OPENCOST_PROFILE"),
		DefaultWindow:    getenvDefault("KUBECOST_DEFAULT_WINDOW", "30d"),
		Timeout:          timeout,
		TLSSkipVerify:    skipVerify,
		ClusterID:        os.Getenv("KUBECOST_CLUSTER_ID"),
		DefaultNamespace: getenvDefault("KUBECOST_DEFAULT_NAMESPACE", "default"),
		PredictionWindow: getenvDefault("KUBECOST_PREDICTION_WINDOW", "2d"),
		Currency:         os.Getenv("OPENCOST_CURRENCY"),
	}
	if path != "" {
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			// If file doesn't exist, just use environment/default values
			if !errors.Is(readErr, os.ErrNotExist) {
				return cfg, readErr
			}
		} else if yamlErr := yaml.Unmarshal(b, &cfg); yamlErr != nil {
			return Config{}, fmt.Errorf("config file %s: %w", path, yamlErr)
		}
	}
	if profile := os.Getenv("OPENCOST_PROFILE"); profile != "" {
		cfg.Profile = profile
	}
	if currency := os.Getenv("OPENCOST_CURRENCY"); currency != "" {
		cfg.Currency = currency
	}
	cfg.APIToken = os.Getenv("KUBECOST_API_TOKEN")
	return cfg, nil
}

// Validate checks the loaded config before the plugin serves.
// Error messages name the key, never a value: values can carry secrets.
func (c Config) Validate() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("config: baseUrl must be an http(s) URL with a host")
	}
	switch c.Profile {
	case "", ProfileOpenCost, ProfileKubecost:
	default:
		return fmt.Errorf("config: profile must be %q or %q, got %q", ProfileOpenCost, ProfileKubecost, c.Profile)
	}
	if c.Currency != "" && !isCurrencyCode(c.Currency) {
		return errors.New("config: currency must be three upper-case letters")
	}
	if c.Timeout < 0 {
		return errors.New("config: timeout must not be negative")
	}
	if c.RequestsPerSecond < 0 {
		return errors.New("config: requestsPerSecond must not be negative")
	}
	if c.RateBurst < 0 {
		return errors.New("config: rateBurst must not be negative")
	}
	return nil
}

const currencyCodeLength = 3

func isCurrencyCode(s string) bool {
	if len(s) != currencyCodeLength {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// ProfileName returns opencost or kubecost. An empty profile is opencost.
func (c Config) ProfileName() (string, error) {
	return c.resolvedProfile()
}

func (c Config) resolvedProfile() (string, error) {
	switch c.Profile {
	case "", ProfileOpenCost:
		return ProfileOpenCost, nil
	case ProfileKubecost:
		return ProfileKubecost, nil
	default:
		return "", fmt.Errorf("unknown allocation profile %q", c.Profile)
	}
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

//nolint:unparam // Function kept generic for potential future use
func getenvDuration(k string, def time.Duration) (time.Duration, error) {
	if v := os.Getenv(k); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("environment variable %s: %w", k, err)
		}
		return d, nil
	}
	return def, nil
}

// getenvBool accepts only the exact texts "true" and "false". Anything
// else is an error: a mistyped value must fail loud, not silently verify.
func getenvBool(k string) (bool, error) {
	switch v := os.Getenv(k); v {
	case "":
		return false, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("environment variable %s must be true or false", k)
	}
}

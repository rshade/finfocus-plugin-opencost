package allocation

import (
	"errors"
	"fmt"
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
	cfg := Config{
		BaseURL:          os.Getenv("KUBECOST_BASE_URL"),
		APIToken:         os.Getenv("KUBECOST_API_TOKEN"),
		Profile:          os.Getenv("OPENCOST_PROFILE"),
		DefaultWindow:    getenvDefault("KUBECOST_DEFAULT_WINDOW", "30d"),
		Timeout:          getenvDuration("KUBECOST_TIMEOUT", defaultTimeoutDuration),
		TLSSkipVerify:    os.Getenv("KUBECOST_TLS_SKIP_VERIFY") == "true",
		ClusterID:        os.Getenv("KUBECOST_CLUSTER_ID"),
		DefaultNamespace: getenvDefault("KUBECOST_DEFAULT_NAMESPACE", "default"),
		PredictionWindow: getenvDefault("KUBECOST_PREDICTION_WINDOW", "2d"),
		Currency:         os.Getenv("OPENCOST_CURRENCY"),
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			// If file doesn't exist, just use environment/default values
			if !errors.Is(err, os.ErrNotExist) {
				return cfg, err
			}
		} else {
			_ = yaml.Unmarshal(b, &cfg)
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
func getenvDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

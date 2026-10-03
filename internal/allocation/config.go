package allocation

import (
	"errors"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultTimeoutDuration = 15 * time.Second

type Config struct {
	BaseURL       string        `yaml:"baseUrl"`
	APIToken      string        `yaml:"apiToken"`
	DefaultWindow string        `yaml:"defaultWindow"` // e.g. "30d"
	Timeout       time.Duration `yaml:"timeout"`
	TLSSkipVerify bool          `yaml:"tlsSkipVerify"`
	// Prediction API specific configuration
	ClusterID        string `yaml:"clusterId"`
	DefaultNamespace string `yaml:"defaultNamespace"`
	PredictionWindow string `yaml:"predictionWindow"` // e.g. "2d" (default for prediction API)
}

func LoadConfigFromEnvOrFile(path string) (Config, error) {
	cfg := Config{
		BaseURL:          os.Getenv("KUBECOST_BASE_URL"),
		APIToken:         os.Getenv("KUBECOST_API_TOKEN"),
		DefaultWindow:    getenvDefault("KUBECOST_DEFAULT_WINDOW", "30d"),
		Timeout:          getenvDuration("KUBECOST_TIMEOUT", defaultTimeoutDuration),
		TLSSkipVerify:    os.Getenv("KUBECOST_TLS_SKIP_VERIFY") == "true",
		ClusterID:        os.Getenv("KUBECOST_CLUSTER_ID"),
		DefaultNamespace: getenvDefault("KUBECOST_DEFAULT_NAMESPACE", "default"),
		PredictionWindow: getenvDefault("KUBECOST_PREDICTION_WINDOW", "2d"),
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
	return cfg, nil
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

package allocation

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	// AllowInsecureHTTP permits a non-loopback http base URL.
	AllowInsecureHTTP bool `yaml:"allowInsecureHttp"`
	// CACertFile is a PEM bundle added to the system trust pool.
	CACertFile string `yaml:"caCertFile"`
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
	// IncludeUsage adds efficiency and request/usage extended columns to actual cost results.
	IncludeUsage bool `yaml:"includeUsage"`
	// Recommendations turns on right-sizing GetRecommendations. It needs RecommendationHeadroom.
	Recommendations bool `yaml:"recommendations"`
	// RecommendationHeadroom is the fraction above average usage a recommended request keeps.
	// It has no default: an operator sets it.
	RecommendationHeadroom float64 `yaml:"recommendationHeadroom"`
	// Kube reaches OpenCost through the Kubernetes API server service proxy
	// instead of BaseURL. A set Kube.Service selects it.
	Kube Kube `yaml:"kube"`
}

// Kube names the in-cluster OpenCost service and the kubeconfig that reaches it.
type Kube struct {
	// Context is the kubeconfig context. Empty uses the current context.
	Context string `yaml:"context"`
	// Namespace defaults to opencost.
	Namespace string `yaml:"namespace"`
	// Service turns kube mode on.
	Service string `yaml:"service"`
	// Port defaults to 9003.
	Port int `yaml:"port"`
	// Kubeconfig overrides KUBECONFIG and the default path.
	Kubeconfig string `yaml:"kubeconfig"`
}

const (
	// DefaultKubeNamespace is the OpenCost Helm chart namespace.
	DefaultKubeNamespace = "opencost"
	// DefaultKubePort is the OpenCost allocation API port.
	DefaultKubePort = 9003
	maxPort         = 65535
)

func LoadConfigFromEnvOrFile(path string) (Config, error) {
	timeout, err := getenvDuration("KUBECOST_TIMEOUT", defaultTimeoutDuration)
	if err != nil {
		return Config{}, err
	}
	skipVerify, err := getenvBool("KUBECOST_TLS_SKIP_VERIFY")
	if err != nil {
		return Config{}, err
	}
	allowInsecure, err := getenvBool("KUBECOST_ALLOW_INSECURE_HTTP")
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		BaseURL:           os.Getenv("KUBECOST_BASE_URL"),
		APIToken:          os.Getenv("KUBECOST_API_TOKEN"),
		Profile:           os.Getenv("OPENCOST_PROFILE"),
		DefaultWindow:     getenvDefault("KUBECOST_DEFAULT_WINDOW", "30d"),
		Timeout:           timeout,
		TLSSkipVerify:     skipVerify,
		AllowInsecureHTTP: allowInsecure,
		CACertFile:        os.Getenv("KUBECOST_CA_CERT_FILE"),
		ClusterID:         os.Getenv("KUBECOST_CLUSTER_ID"),
		DefaultNamespace:  getenvDefault("KUBECOST_DEFAULT_NAMESPACE", "default"),
		PredictionWindow:  getenvDefault("KUBECOST_PREDICTION_WINDOW", "2d"),
		Currency:          os.Getenv("OPENCOST_CURRENCY"),
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
	if cfg, err = cfg.withFeatureEnv(); err != nil {
		return Config{}, err
	}
	cfg.APIToken = os.Getenv("KUBECOST_API_TOKEN")
	kube, kubeErr := cfg.Kube.withEnv()
	if kubeErr != nil {
		return Config{}, kubeErr
	}
	cfg.Kube = kube
	return cfg, nil
}

// withFeatureEnv returns c with the feature switches laid over the file values.
// Each variable wins only when it is set; an unparsable value names the variable.
func (c Config) withFeatureEnv() (Config, error) {
	for env, dst := range map[string]*bool{
		"OPENCOST_INCLUDE_USAGE":   &c.IncludeUsage,
		"OPENCOST_RECOMMENDATIONS": &c.Recommendations,
	} {
		if os.Getenv(env) == "" {
			continue
		}
		value, err := getenvBool(env)
		if err != nil {
			return Config{}, err
		}
		*dst = value
	}
	if raw := os.Getenv("OPENCOST_RECOMMENDATION_HEADROOM"); raw != "" {
		headroom, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Config{}, fmt.Errorf(
				"environment variable OPENCOST_RECOMMENDATION_HEADROOM must be a number: %w",
				err,
			)
		}
		c.RecommendationHeadroom = headroom
	}
	return c, nil
}

// withEnv returns k with the OPENCOST_KUBE_* variables laid over it.
func (k Kube) withEnv() (Kube, error) {
	for env, dst := range map[string]*string{
		"OPENCOST_KUBE_CONTEXT":   &k.Context,
		"OPENCOST_KUBE_NAMESPACE": &k.Namespace,
		"OPENCOST_KUBE_SERVICE":   &k.Service,
	} {
		if v := os.Getenv(env); v != "" {
			*dst = v
		}
	}
	if v := os.Getenv("OPENCOST_KUBE_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Kube{}, errors.New("environment variable OPENCOST_KUBE_PORT must be an integer")
		}
		k.Port = port
	}
	return k, nil
}

// Enabled reports whether requests go through the API server service proxy.
func (k Kube) Enabled() bool { return k.Service != "" }

// Target returns the namespace and port with defaults applied.
func (k Kube) Target() (string, int) {
	namespace, port := k.Namespace, k.Port
	if namespace == "" {
		namespace = DefaultKubeNamespace
	}
	if port == 0 {
		port = DefaultKubePort
	}
	return namespace, port
}

func (c Config) validateKube() error {
	for _, conflict := range []struct {
		key string
		set bool
	}{
		{"baseUrl", c.BaseURL != ""},
		{"tlsSkipVerify", c.TLSSkipVerify},
		{"caCertFile", c.CACertFile != ""},
		{"allowInsecureHttp", c.AllowInsecureHTTP},
	} {
		if conflict.set {
			return fmt.Errorf(
				"config: %s cannot be set with kube.service; the kubeconfig owns that connection",
				conflict.key,
			)
		}
	}
	if c.Profile == ProfileKubecost {
		return errors.New("config: profile must be opencost with kube.service")
	}
	if c.Kube.Port < 0 || c.Kube.Port > maxPort {
		return errors.New("config: kube.port must be 1 to 65535")
	}
	return nil
}

// Validate checks the loaded config before the plugin serves.
// Error messages name the key, never a value: values can carry secrets.
func (c Config) Validate() error {
	if c.Kube.Enabled() {
		if err := c.validateKube(); err != nil {
			return err
		}
	} else if err := c.validateBaseURL(); err != nil {
		return err
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
	return c.validateRecommendations()
}

func (c Config) validateRecommendations() error {
	if !c.Recommendations {
		return nil
	}
	if c.RecommendationHeadroom <= 0 {
		return errors.New("config: recommendations needs recommendationHeadroom above zero")
	}
	if c.Profile == ProfileKubecost {
		return errors.New("config: recommendations needs profile opencost")
	}
	return nil
}

func (c Config) validateBaseURL() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("config: baseUrl must be an http(s) URL with a host")
	}
	if u.Scheme == "http" && !c.AllowInsecureHTTP && !loopbackHost(u.Hostname()) {
		return errors.New("config: base URL uses http; use https or set allowInsecureHttp")
	}
	if c.CACertFile != "" {
		if _, caErr := rootPool(c.CACertFile); caErr != nil {
			return caErr
		}
	}
	return nil
}

const currencyCodeLength = 3

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

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

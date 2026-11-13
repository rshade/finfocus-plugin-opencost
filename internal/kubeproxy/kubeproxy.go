// Package kubeproxy reaches an in-cluster service through the Kubernetes API
// server service proxy, using the caller's kubeconfig or pod credentials.
package kubeproxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Target names the service to reach and where the credentials come from.
type Target struct {
	// Context is the kubeconfig context. Empty uses the current context.
	Context string
	// Namespace and Port must be set by the caller.
	Namespace string
	Service   string
	Port      int
	// Kubeconfig is a file path. Empty uses KUBECONFIG, then the default path,
	// then the pod's service account.
	Kubeconfig string
}

// Connection is a resolved kube target: the API server origin, the transport
// that proxies requests to the service, and the kubeconfig context in use
// ("in-cluster" when the pod's service account was used).
type Connection struct {
	BaseURL   string
	Transport http.RoundTripper
	Context   string
}

// Connect resolves a target into a Connection. The origin has no path: the
// transport prepends the proxy path itself.
func Connect(t Target) (Connection, error) {
	return resolve(t, sources{home: clientcmd.RecommendedHomeFile, inCluster: rest.InClusterConfig})
}

// Resolve returns the API server origin and a transport that sends every
// request through the service proxy with the target's credentials.
func Resolve(t Target) (string, http.RoundTripper, error) {
	conn, err := Connect(t)
	return conn.BaseURL, conn.Transport, err
}

// sources are the ambient inputs Resolve reads besides the Target, so a test
// can replace them without touching client-go's package variables.
type sources struct {
	// home is the kubeconfig used when KUBECONFIG is unset.
	home string
	// inCluster loads a pod's service account.
	inCluster func() (*rest.Config, error)
}

func resolve(t Target, src sources) (Connection, error) {
	cfg, contextName, err := restConfig(t, src)
	if err != nil {
		return Connection{}, err
	}
	origin, hostPath, err := splitHost(cfg.Host)
	if err != nil {
		return Connection{}, err
	}
	base, err := rest.TransportFor(cfg)
	if err != nil {
		return Connection{}, fmt.Errorf("kube: build transport for context %q: %w", contextName, err)
	}
	return Connection{BaseURL: origin, Context: contextName, Transport: &proxyTransport{
		next:   base,
		origin: origin,
		prefix: fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s:%d/proxy",
			hostPath, url.PathEscape(t.Namespace), url.PathEscape(t.Service), t.Port),
		desc: describe{
			context:   contextName,
			namespace: t.Namespace,
			service:   t.Service,
			execCmd:   execCommand(cfg),
		},
	}}, nil
}

func execCommand(cfg *rest.Config) string {
	if cfg.ExecProvider == nil {
		return ""
	}
	return cfg.ExecProvider.Command
}

func loadingRules(t Target, src sources) *clientcmd.ClientConfigLoadingRules {
	if t.Kubeconfig != "" {
		return &clientcmd.ClientConfigLoadingRules{ExplicitPath: t.Kubeconfig}
	}
	if env := os.Getenv(clientcmd.RecommendedConfigPathEnvVar); env != "" {
		return &clientcmd.ClientConfigLoadingRules{Precedence: filepath.SplitList(env)}
	}
	return &clientcmd.ClientConfigLoadingRules{Precedence: []string{src.home}}
}

func restConfig(t Target, src sources) (*rest.Config, string, error) {
	rules := loadingRules(t, src)
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{CurrentContext: t.Context},
	)
	raw, err := loader.RawConfig()
	if err != nil {
		return nil, "", fmt.Errorf("kube: read kubeconfig: %w", err)
	}
	if len(raw.Contexts) == 0 && t.Kubeconfig == "" {
		return inCluster(t, src)
	}
	contextName, err := selectContext(t, raw)
	if err != nil {
		return nil, "", err
	}
	cfg, err := loader.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("kube: context %q: %w", contextName, err)
	}
	return cfg, contextName, nil
}

func selectContext(t Target, raw clientcmdapi.Config) (string, error) {
	if t.Context != "" {
		if _, ok := raw.Contexts[t.Context]; !ok {
			return "", fmt.Errorf("kube: context %q is not in the kubeconfig", t.Context)
		}
		return t.Context, nil
	}
	if raw.CurrentContext == "" {
		return "", errors.New("kube: kubeconfig has no current context; set kube.context")
	}
	return raw.CurrentContext, nil
}

func inCluster(t Target, src sources) (*rest.Config, string, error) {
	if t.Context != "" {
		return nil, "", fmt.Errorf("kube: context %q requested but no kubeconfig was found", t.Context)
	}
	cfg, err := src.inCluster()
	if err != nil {
		return nil, "", errors.New("kube: no kubeconfig found and no pod service account is mounted")
	}
	return cfg, "in-cluster", nil
}

func splitHost(host string) (string, string, error) {
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return "", "", errors.New("kube: kubeconfig server address is not a URL with a host")
	}
	return u.Scheme + "://" + u.Host, strings.TrimRight(u.Path, "/"), nil
}

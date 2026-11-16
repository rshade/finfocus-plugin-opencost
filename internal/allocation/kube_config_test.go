package allocation_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestKubeConfigValidation(t *testing.T) {
	kube := allocation.Kube{Service: "opencost"}

	invalid := []struct {
		name string
		cfg  allocation.Config
		want []string
	}{
		{
			"address and kube target",
			allocation.Config{BaseURL: "https://h:1", Kube: kube},
			[]string{"baseUrl", "kube.service"},
		},
		{"ignored trust file", allocation.Config{CACertFile: "ca.pem", Kube: kube}, []string{"caCertFile"}},
		{"ignored skip verify", allocation.Config{TLSSkipVerify: true, Kube: kube}, []string{"tlsSkipVerify"}},
		{
			"ignored insecure http",
			allocation.Config{AllowInsecureHTTP: true, Kube: kube},
			[]string{"allowInsecureHttp"},
		},
		{"port too large", allocation.Config{Kube: allocation.Kube{Service: "oc", Port: 70000}}, []string{"kube.port"}},
		{"port negative", allocation.Config{Kube: allocation.Kube{Service: "oc", Port: -1}}, []string{"kube.port"}},
		{"kubecost profile", allocation.Config{Profile: allocation.ProfileKubecost, Kube: kube}, []string{"profile"}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			require.Error(t, err)
			for _, want := range tc.want {
				require.Contains(t, err.Error(), want)
			}
		})
	}

	valid := []struct {
		name string
		cfg  allocation.Config
	}{
		{"service only", allocation.Config{Kube: kube}},
		{"empty profile", allocation.Config{Kube: kube, Profile: ""}},
		{"opencost profile", allocation.Config{Kube: kube, Profile: allocation.ProfileOpenCost}},
		{"explicit port", allocation.Config{Kube: allocation.Kube{Service: "oc", Port: 8080}}},
	}
	for _, tc := range valid {
		t.Run("valid "+tc.name, func(t *testing.T) {
			require.NoError(t, tc.cfg.Validate())
		})
	}

	t.Run("kube block without a service needs baseUrl", func(t *testing.T) {
		err := allocation.Config{Kube: allocation.Kube{Context: "prod"}}.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "baseUrl")
	})
}

func TestKubeEnvOverrides(t *testing.T) {
	path := writeConfigFile(
		t,
		"kube:\n  context: file-ctx\n  namespace: file-ns\n  service: a\n  port: 8080\n  kubeconfig: /k\n",
	)

	t.Run("file loads", func(t *testing.T) {
		cfg, err := allocation.LoadConfigFromEnvOrFile(path)
		require.NoError(t, err)
		require.Equal(
			t,
			allocation.Kube{Context: "file-ctx", Namespace: "file-ns", Service: "a", Port: 8080, Kubeconfig: "/k"},
			cfg.Kube,
		)
	})

	t.Run("environment wins", func(t *testing.T) {
		t.Setenv("OPENCOST_KUBE_SERVICE", "b")
		t.Setenv("OPENCOST_KUBE_CONTEXT", "env-ctx")
		t.Setenv("OPENCOST_KUBE_NAMESPACE", "env-ns")
		t.Setenv("OPENCOST_KUBE_PORT", "9100")
		cfg, err := allocation.LoadConfigFromEnvOrFile(path)
		require.NoError(t, err)
		require.Equal(t, "b", cfg.Kube.Service)
		require.Equal(t, "env-ctx", cfg.Kube.Context)
		require.Equal(t, "env-ns", cfg.Kube.Namespace)
		require.Equal(t, 9100, cfg.Kube.Port)
		require.Equal(t, "/k", cfg.Kube.Kubeconfig)
	})

	t.Run("empty variable keeps the file value", func(t *testing.T) {
		t.Setenv("OPENCOST_KUBE_SERVICE", "")
		cfg, err := allocation.LoadConfigFromEnvOrFile(path)
		require.NoError(t, err)
		require.Equal(t, "a", cfg.Kube.Service)
	})

	t.Run("port that is not a number", func(t *testing.T) {
		t.Setenv("OPENCOST_KUBE_PORT", "nine")
		_, err := allocation.LoadConfigFromEnvOrFile(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "OPENCOST_KUBE_PORT")
	})
}

func TestKubeDefaultsNamespaceAndPort(t *testing.T) {
	ns, port := allocation.Kube{Service: "opencost"}.Target()
	require.Equal(t, "opencost", ns)
	require.Equal(t, 9003, port)

	ns, port = allocation.Kube{Service: "oc", Namespace: "cost", Port: 8080}.Target()
	require.Equal(t, "cost", ns)
	require.Equal(t, 8080, port)
}

package allocation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestRecommendationsConfig(t *testing.T) {
	load := func(t *testing.T, file string) (allocation.Config, error) {
		t.Helper()
		path := ""
		if file != "" {
			path = filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(file), 0o600))
		}
		return allocation.LoadConfigFromEnvOrFile(path)
	}

	t.Run("defaults", func(t *testing.T) {
		t.Setenv("OPENCOST_RECOMMENDATIONS", "")
		t.Setenv("OPENCOST_RECOMMENDATION_HEADROOM", "")
		cfg, err := load(t, "")
		require.NoError(t, err)
		require.False(t, cfg.Recommendations)
		require.Zero(t, cfg.RecommendationHeadroom)
	})

	t.Run("file", func(t *testing.T) {
		t.Setenv("OPENCOST_RECOMMENDATIONS", "")
		t.Setenv("OPENCOST_RECOMMENDATION_HEADROOM", "")
		cfg, err := load(t, "recommendations: true\nrecommendationHeadroom: 0.2\n")
		require.NoError(t, err)
		require.True(t, cfg.Recommendations)
		require.InDelta(t, 0.2, cfg.RecommendationHeadroom, 0)
	})

	t.Run("env overrides file", func(t *testing.T) {
		t.Setenv("OPENCOST_RECOMMENDATIONS", "false")
		t.Setenv("OPENCOST_RECOMMENDATION_HEADROOM", "0.5")
		cfg, err := load(t, "recommendations: true\nrecommendationHeadroom: 0.2\n")
		require.NoError(t, err)
		require.False(t, cfg.Recommendations)
		require.InDelta(t, 0.5, cfg.RecommendationHeadroom, 0)
	})

	for _, key := range []string{"OPENCOST_RECOMMENDATIONS", "OPENCOST_RECOMMENDATION_HEADROOM"} {
		t.Run("invalid "+key, func(t *testing.T) {
			t.Setenv("OPENCOST_RECOMMENDATIONS", "")
			t.Setenv("OPENCOST_RECOMMENDATION_HEADROOM", "")
			t.Setenv(key, "lots")
			_, err := load(t, "")
			require.ErrorContains(t, err, key)
		})
	}

	base := allocation.Config{BaseURL: "https://opencost.example.com"}
	cases := []struct {
		name    string
		mutate  func(*allocation.Config)
		wantErr string
	}{
		{name: "off needs nothing", mutate: func(*allocation.Config) {}},
		{name: "on with headroom", mutate: func(c *allocation.Config) {
			c.Recommendations, c.RecommendationHeadroom = true, 0.2
		}},
		{name: "on without headroom", mutate: func(c *allocation.Config) {
			c.Recommendations = true
		}, wantErr: "recommendationHeadroom"},
		{name: "negative headroom", mutate: func(c *allocation.Config) {
			c.Recommendations, c.RecommendationHeadroom = true, -0.1
		}, wantErr: "recommendationHeadroom"},
		{name: "profile kubecost", mutate: func(c *allocation.Config) {
			c.Recommendations, c.RecommendationHeadroom, c.Profile = true, 0.2, allocation.ProfileKubecost
		}, wantErr: "profile"},
	}
	for _, tc := range cases {
		t.Run("validate "+tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			err := cfg.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

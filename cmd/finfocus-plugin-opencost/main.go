package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func main() {
	os.Exit(run())
}

func run() int {
	showVersion := flag.Bool("version", false, "Show version information")
	showVersionFull := flag.Bool("version-full", false, "Show detailed version information")
	flag.Parse()

	if *showVersion {
		_, _ = os.Stdout.WriteString(version.String() + "\n")
		return 0
	}
	if *showVersionFull {
		_, _ = os.Stdout.WriteString(version.FullString() + "\n")
		return 0
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Str("plugin", "opencost").Logger()

	configPath := os.Getenv("OPENCOST_CONFIG")
	if configPath == "" {
		configPath = os.Getenv("KUBECOST_CONFIG")
	}
	cfg, err := allocation.LoadConfigFromEnvOrFile(configPath)
	if err != nil {
		logger.Error().Err(err).Msg("config")
		return 1
	}
	if validateErr := cfg.Validate(); validateErr != nil {
		logger.Error().Err(validateErr).Msg("config")
		return 1
	}
	cli, err := allocation.NewClient(context.Background(), cfg)
	if err != nil {
		logger.Error().Err(err).Msg("client")
		return 1
	}
	cli.SetLogger(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	port := pluginsdk.ParsePortFlag()
	if port == 0 {
		port = pluginsdk.GetPort()
	}

	plugin := server.New(cli)
	plugin.SetLogger(logger)
	serveErr := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
		Plugin:     plugin,
		PluginInfo: server.Info(),
		Port:       port,
		Logger:     &logger,
	})
	if serveErr != nil {
		logger.Error().Err(serveErr).Msg("serve")
		return 1
	}
	return 0
}

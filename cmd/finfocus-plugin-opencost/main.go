package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
	// TODO: Add when finfocus-spec is available
	// pbc "github.com/rshade/finfocus-spec/sdk/go/proto"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showVersionFull := flag.Bool("version-full", false, "Show detailed version information")
	flag.Parse()

	// Handle version flags
	if *showVersion {
		_, _ = os.Stdout.WriteString(version.String() + "\n")
		os.Exit(0)
	}
	if *showVersionFull {
		_, _ = os.Stdout.WriteString(version.FullString() + "\n")
		os.Exit(0)
	}

	cfg, err := allocation.LoadConfigFromEnvOrFile(os.Getenv("KUBECOST_CONFIG"))
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cli, err := allocation.NewClient(cubectx(context.Background()), cfg)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	log.Printf("finfocus-plugin-opencost starting, %s", version.String())

	// Pulumi-style plugins often use stdin/stdout. For simplicity here, use a TCP loopback.
	// Your plugin host can launch and connect to this ephemeral port; or adapt to stdio transport.
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	_ = server.NewKubecostServer(cli)
	// TODO: Uncomment when finfocus-spec protobuf definitions are available
	// kubecostServer.RegisterService(grpcServer)

	log.Printf("listening on %s", lis.Addr().String())
	if serveErr := grpcServer.Serve(lis); serveErr != nil {
		log.Fatalf("serve: %v", serveErr)
	}
}

const defaultTimeoutSeconds = 30

func cubectx(ctx context.Context) context.Context {
	t := defaultTimeoutSeconds * time.Second
	if d := os.Getenv("KUBECOST_TIMEOUT"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			t = parsed
		}
	}
	c, cancel := context.WithTimeout(ctx, t)
	// Note: cancel is not called because the timeout context is returned for immediate use.
	// The caller is responsible for cleanup via the context's done channel or timeout expiration.
	_ = cancel
	return c
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pauvalls/arx/internal/infrastructure/server"
	"github.com/spf13/cobra"
)

var (
	serverPort int
	serverBind string
	serverPath string
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start architecture web server with interactive dashboard",
	Long: `Start an HTTP server with an interactive dashboard and REST API
for real-time architecture monitoring.

The server runs architecture checks on your project and serves the results
via both a web dashboard and JSON API endpoints.

API endpoints:
  GET /api/health      - Health check (always returns 200)
  GET /api/status      - Server status, version, violation count
  GET /api/violations  - List of current violations
  GET /api/coupling    - Coupling matrix between layers
  GET /api/debt        - Technical debt score

Example:
  arx server                    # Start on localhost:8080
  arx server --port 3000        # Start on port 3000
  arx server --bind 0.0.0.0     # Bind to all interfaces
  arx server -d ./my-project    # Check a specific project directory`,
	RunE: runServer,
}

func init() {
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Server port")
	serverCmd.Flags().StringVar(&serverBind, "bind", "127.0.0.1", "Bind address")
	serverCmd.Flags().StringVarP(&serverPath, "path", "d", ".", "Project root path")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	// Resolve project root to absolute path
	projectRoot := serverPath
	if len(args) > 0 {
		projectRoot = args[0]
	}
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Verify config exists
	configPath := filepath.Join(projectRoot, "arx.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nRun 'arx init' to generate a configuration file", configPath)
	}

	// Create server state with version info
	versionInfo := GetVersionInfo()
	state := server.NewServerState(server.VersionInfo{
		Version:   versionInfo.Version,
		Commit:    versionInfo.Commit,
		BuildDate: versionInfo.BuildDate,
		GoVersion: versionInfo.GoVersion,
	})

	// Create CheckService
	service := server.NewDefaultCheckService()

	// Run initial check
	ctx := context.Background()
	server.RunCheck(ctx, service, projectRoot, state)

	// Create and start server
	cachePath := filepath.Join(projectRoot, ".arx-cache", "server-state.json")
	srv := server.New(serverPort, serverBind, projectRoot, cachePath, service, state)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait for signal or server error
	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived %v, shutting down...\n", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		return srv.Stop(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

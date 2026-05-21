package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/bootstrap"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/lsp"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// lspCmd represents the LSP server command
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start an LSP server for real-time architecture diagnostics",
	Long: `Start a Language Server Protocol (LSP) server that provides
real-time architecture diagnostics, code actions, and hover information.

The server communicates over stdin/stdout using JSON-RPC 2.0 with
Content-Length headers. It is compatible with any editor that supports
the LSP protocol (VS Code, Neovim, Helix, Zed, etc.).

The server reads arx.yaml from the current working directory for
architecture rules and layer configuration.

Example:
  arx lsp                      # Start LSP server on stdio`,
	RunE: runLSP,
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

func runLSP(cmd *cobra.Command, args []string) error {
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Load config
	configPath := filepath.Join(projectRoot, "arx.yaml")
	reader := config.NewYAMLReader()

	var cfg *domain.Config
	if _, statErr := os.Stat(configPath); statErr == nil {
		loadedCfg, loadErr := reader.Read(configPath)
		if loadErr != nil {
			return fmt.Errorf("loading config: %w", loadErr)
		}
		cfg = loadedCfg
	} else {
		fmt.Fprintf(os.Stderr, "Warning: no arx.yaml found in %s\n", projectRoot)
	}

	// Create CheckService (for diagnostics)
	var checkService *application.CheckService
	if cfg != nil {
		var detectors []ports.Detector
		detectors = bootstrap.BuildDetectorsWithPlugins(cfg)
		checkService = application.NewCheckService(reader, detectors, nil)
	}

	// Create FixEngine (for code actions)
	fixEngine := application.NewFixEngine()

	// Create LSP server
	server := lsp.NewServer(checkService, fixEngine, cfg)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- lsp.Run(ctx, server, os.Stdin, os.Stdout)
	}()

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived %v, shutting down...\n", sig)
		server.Shutdown()
		// Close stdin to unblock the ReadMessage call in Run
		os.Stdin.Close()
		cancel()
		<-errCh // wait for Run to exit
		return nil
	case err := <-errCh:
		if err != nil {
			server.Shutdown()
			return err
		}
		return nil
	}
}

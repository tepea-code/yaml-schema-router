// Package main is the entry point for yaml-schema-router.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/detector/kubernetes"
	"go.trai.ch/yaml-schema-router/internal/lspproxy"
	"go.trai.ch/yaml-schema-router/internal/schemaregistry"
)

const componentName = "Main"

func main() {
	if err := run(); err != nil {
		log.Fatalf("[%s] Fatal error: %v", componentName, err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	defaultLogPath := ""
	homeDir, homeErr := os.UserHomeDir()
	if homeErr == nil {
		defaultLogPath = filepath.Join(homeDir, ".config", config.DefaultConfigDirName, "router.log")
	} else {
		defaultLogPath = filepath.Join(os.TempDir(), "yaml-schema-router.log")
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("could not determine user cache dir: %w", err)
	}
	defaultCacheDir := filepath.Join(userCacheDir, config.DefaultConfigDirName)

	logFile := flag.String(
		"log-file",
		defaultLogPath,
		"Path to write logs (don't log to stdout!)",
	)
	lspPath := flag.String(
		"lsp-path",
		"yaml-language-server",
		"Path to the yaml-language-server executable. Defaults to checking the system PATH.",
	)
	kubernetesVersion := flag.String(
		"kubernetes-version",
		config.DefaultK8sSchemaVersion,
		"Kubernetes schema version to use (for example, v1.36.0).",
	)
	schemaDir := flag.String(
		"schema-dir",
		filepath.Join(defaultCacheDir, "schemas"),
		"Directory containing downloaded and user-provided schemas.",
	)
	_ = flag.Bool(
		"stdio",
		true,
		"Ignored. Kept for compatibility with LSP clients that automatically append it.",
	)
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if *logFile != "" {
		logDir := filepath.Dir(*logFile)
		if err := os.MkdirAll(logDir, config.DefaultDirPerm); err != nil {
			return err
		}

		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, config.DefaultFilePerm)
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("[%s] error closing file: %v", componentName, err)
			}
		}()
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stderr)
	}

	log.Printf("[%s] Starting yaml-schema-router. Using LSP executable: %s", componentName, *lspPath)

	registry, err := schemaregistry.NewRegistry(
		*schemaDir,
		filepath.Join(defaultCacheDir, "generated"),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize schema registry: %v", err)
	}

	k8sDetector := &kubernetes.K8sDetector{
		Registry: registry,
		Version:  *kubernetesVersion,
	}
	crdDetector := &kubernetes.CRDDetector{
		Registry: registry,
		Version:  *kubernetesVersion,
	}
	chain := detector.NewChain(k8sDetector, crdDetector)

	proxy := lspproxy.NewProxy(*lspPath, chain, registry)

	if err := proxy.Start(ctx); err != nil {
		return err
	}

	log.Printf("[%s] Proxy shut down cleanly.", componentName)

	return nil
}

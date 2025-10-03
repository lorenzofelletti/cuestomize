package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/Workday/cuestomize/api"
	"github.com/Workday/cuestomize/cmd"
	krm "github.com/Workday/cuestomize/internal/pkg/cuestomize"
	"github.com/Workday/cuestomize/internal/pkg/processor"
	"github.com/rs/zerolog"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

const (
	// LogLevelEnvVar is the name of the environment variable that can be used to set the log level.
	LogLevelEnvVar = "LOG_LEVEL"
)

// Version is the version of Cuestomize.
//
//go:embed semver
var Version string

func main() {
	if err := setupLogging(); err != nil {
		os.Exit(1)
	}

	tempDir, err := os.MkdirTemp("", "cuestomize-cli-")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// This is the KRM function processor for the root command.
	config := new(api.KRMInput)
	fn, err := krm.NewBuilder().SetConfig(config).SetResourcesPath(tempDir).Build()
	if err != nil {
		log.Fatalf("failed to build KRM function: %v", err)
	}
	p := processor.NewSimpleProcessor(config, kio.FilterFunc(fn), true)

	// rootCmd is the command that runs as a KRM function.
	rootCmd := command.Build(p, command.StandaloneEnabled, false)
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("v{{.Version}}")

	// cliCmd is the command that runs as a standalone CLI.
	cliCmd := cmd.NewCLICommand()
	rootCmd.AddCommand(cliCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// setupLogging configures the global logging level based on the log level environment variable.
func setupLogging() error {
	logLevel := os.Getenv(LogLevelEnvVar)
	if logLevel == "" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("failed to parse log level from environment variable %s: %w", LogLevelEnvVar, err)
	}
	zerolog.SetGlobalLevel(level)
	return nil
}

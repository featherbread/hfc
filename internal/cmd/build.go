package cmd

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var buildCmd = &cobra.Command{
	Use:    "build",
	Short:  "Build the Go binary for Lambda",
	PreRun: initializePreRun,
	Run:    runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) {
	outputPath, err := rootState.BinaryPath(rootConfig.Project.Name)
	if err != nil {
		log.Fatal(err)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.RemoveAll(outputDir); err != nil {
		log.Fatal("cleaning output directory: ", err)
	}
	if err := os.MkdirAll(outputDir, fs.ModeDir|0755); err != nil {
		log.Fatal("creating output directory: ", err)
	}

	shelley.ExitIfError(shelley.
		Command(
			"go", "build", "-v",
			"-ldflags", "-s -w",
			"-tags", "lambda.norpc",
			"-o", outputPath,
			rootConfig.Build.Path,
		).
		Env("CGO_ENABLED", "0").Env("GOOS", "linux").Env("GOARCH", "arm64").
		Run())
}

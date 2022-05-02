package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var currentImageCmd = &cobra.Command{
	Use:   "current-image [flags] stack",
	Short: "Display the currently deployed image for a named stack",
	Args:  cobra.ExactArgs(1),
	Run:   runCurrentImage,
}

func init() {
	rootCmd.AddCommand(currentImageCmd)
}

func runCurrentImage(cmd *cobra.Command, args []string) {
	stackName := args[0]
	if _, ok := rootConfig.FindStack(stackName); !ok {
		log.Fatalf("stack %s is not configured", stackName)
	}

	shelley.ExitIfError(shelley.
		Command(
			"aws", "cloudformation", "describe-stacks",
			"--stack-name", stackName,
			"--query", "Stacks[0].Parameters[?ParameterKey=='ImageUri'].ParameterValue",
			"--output", "text",
		).
		Run())
}

package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hfc",
	Short: "Build and deploy serverless Go apps with AWS Lambda and CloudFormation",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[hfc] ")
		log.SetFlags(0)
	},
}

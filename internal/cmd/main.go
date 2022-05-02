package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/config"
	"go.alexhamlin.co/hfc/internal/shelley"
	"go.alexhamlin.co/hfc/internal/state"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	rootConfig config.Config
	rootState  state.State
)

var rootCmd = &cobra.Command{
	Use:   "hfc",
	Short: "Build and deploy serverless Go apps with AWS Lambda and CloudFormation",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[hfc] ")
		log.SetFlags(0)

		shelley.DefaultContext.Aliases["zeroimage"] = []string{
			"go", "run", "go.alexhamlin.co/zeroimage@main"}

		configPath, err := config.FindPath()
		if err != nil {
			log.Fatal(err)
		}
		rootConfig, err = config.Load()
		if err != nil {
			log.Fatal(err)
		}
		rootState, err = state.Get(configPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

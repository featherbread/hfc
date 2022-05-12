package cmd

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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
	awsConfig  aws.Config
)

var rootCmd = &cobra.Command{
	Use:   "hfc",
	Short: "Build and deploy serverless Go apps with AWS Lambda and CloudFormation",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[hfc] ")
		log.SetFlags(0)

		shelley.DefaultContext.DebugLogger = log.New(log.Writer(), "[hfc] $ ", 0)
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

		awsConfig, err = awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	},
}

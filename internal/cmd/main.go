package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/config"
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
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.Load()
		if err != nil {
			log.Fatal("unable to load config: ", err)
		}

		configJSON, _ := json.MarshalIndent(config, "", "  ")
		fmt.Println(string(configJSON))
	},
}

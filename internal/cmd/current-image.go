package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
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

	cfnClient := cloudformation.NewFromConfig(awsConfig)
	output, err := cfnClient.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range output.Stacks[0].Parameters {
		if *p.ParameterKey == "ImageUri" {
			fmt.Println(*p.ParameterValue)
			return
		}
	}

	log.Fatalf("stack %s deployed without an ImageUri parameter", stackName)
}

package cmd

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

var outputsCmd = &cobra.Command{
	Use:               "outputs stack",
	Short:             "Display the outputs for a CloudFormation stack",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeStackNames,
	PreRun:            initializePreRun,
	Run:               runOutputs,
}

func init() {
	rootCmd.AddCommand(outputsCmd)
}

func runOutputs(cmd *cobra.Command, args []string) {
	stackName := args[0]
	_, ok := rootConfig.FindStack(stackName)
	if !ok {
		log.Fatalf("stack %s is not configured", stackName)
	}

	cfnClient := cloudformation.NewFromConfig(awsConfig)
	description, err := cfnClient.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		log.Print("unable to read stack info, will skip printing output")
		return
	}

	for _, output := range description.Stacks[0].Outputs {
		log.Printf("%s (%s):\n\t%s", *output.Description, *output.OutputKey, *output.OutputValue)
	}
}

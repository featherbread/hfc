package cmd

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [flags] stack [parameters]",
	Short: "Deploy a CloudFormation stack using the latest uploaded binary",
	Args:  cobra.MinimumNArgs(1),
	Run:   runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	latestImageRaw, err := os.ReadFile(rootState.LatestImagePath())
	latestImage := string(latestImageRaw)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Fatal("must upload a binary before deploying")
	case err != nil:
		log.Fatal(err)
	}

	stackName := args[0]
	stack, ok := rootConfig.FindStack(stackName)
	if !ok {
		log.Fatalf("stack %s is not configured", stackName)
	}

	var capabilityArgs []string
	if len(rootConfig.Template.Capabilities) > 0 {
		capabilityArgs = append([]string{"--capabilities"}, rootConfig.Template.Capabilities...)
	}

	parameterOverrideArgs := slices.Clone(args[1:])
	for key, value := range stack.Parameters {
		parameterOverrideArgs = append(parameterOverrideArgs, key+"="+value)
	}
	sort.Strings(parameterOverrideArgs)

	deployArgs := concat(
		[]string{
			"aws", "cloudformation", "deploy",
			"--template-file", rootConfig.Template.Path,
			"--stack-name", stackName,
			"--no-fail-on-empty-changeset",
		},
		capabilityArgs,
		[]string{"--parameter-overrides", "ImageUri=" + latestImage},
		parameterOverrideArgs,
	)
	shelley.ExitIfError(shelley.Command(deployArgs...).Run())

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

func concat(slices ...[]string) []string {
	var total int
	for _, slice := range slices {
		total += len(slice)
	}

	result := make([]string, 0, total)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

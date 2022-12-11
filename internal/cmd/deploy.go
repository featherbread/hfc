package cmd

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var deployCmd = &cobra.Command{
	Use:               "deploy [flags] stack [parameters]",
	Short:             "Deploy a CloudFormation stack using the latest uploaded binary",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeStackNames,
	PreRun:            initializePreRun,
	Run:               runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	stackName := args[0]
	stack, ok := rootConfig.FindStack(stackName)
	if !ok {
		log.Fatalf("stack %s is not configured", stackName)
	}

	var regionArgs []string
	if rootConfig.AWS.Region != "" {
		regionArgs = []string{"--region", rootConfig.AWS.Region}
	}

	var capabilityArgs []string
	if len(rootConfig.Template.Capabilities) > 0 {
		capabilityArgs = append([]string{"--capabilities"}, rootConfig.Template.Capabilities...)
	}

	deploymentParameters, err := getDeploymentParameters()
	if err != nil {
		log.Fatal(err)
	}

	overrideParameters := slices.Clone(args[1:])
	for key, value := range stack.Parameters {
		overrideParameters = append(overrideParameters, key+"="+value)
	}
	sort.Strings(overrideParameters)

	deployArgs := concat(
		[]string{"aws", "cloudformation", "deploy"},
		regionArgs,
		[]string{
			"--template-file", rootConfig.Template.Path,
			"--stack-name", stackName,
			"--no-fail-on-empty-changeset",
		},
		capabilityArgs,
		[]string{"--parameter-overrides"},
		deploymentParameters,
		overrideParameters,
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

func getDeploymentParameters() ([]string, error) {
	latestPackageRaw, err := os.ReadFile(rootState.LatestLambdaPackagePath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, errors.New("must upload a deployment package before deploying")
	case err != nil:
		return nil, err
	}

	latestPackage := strings.TrimSpace(string(latestPackageRaw))
	return []string{
		"CodeS3Bucket=" + rootConfig.Upload.Bucket,
		"CodeS3Key=" + latestPackage,
	}, nil
}

func concat(slices ...[]string) []string {
	var totalLen int
	for _, slice := range slices {
		totalLen += len(slice)
	}

	result := make([]string, 0, totalLen)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

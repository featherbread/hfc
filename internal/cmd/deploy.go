package cmd

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/featherbread/hfc/internal/shelley"
)

var deployCmd = &cobra.Command{
	Use:               "deploy [flags] stack [parameters]",
	Short:             "Deploy the CloudFormation stack with the latest upload",
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

	lambdaParameters, err := getLambdaPackageParameters()
	if err != nil {
		log.Fatal(err)
	}

	cliParameters := slices.Clone(args[1:])
	allParameters := lo.Flatten([][]string{
		lambdaParameters,
		cliParameters,
		lo.MapToSlice(stack.Parameters, func(k, v string) string { return k + "=" + v }),
	})
	slices.Sort(allParameters)

	deployArgs := lo.Flatten([][]string{
		{"aws", "cloudformation", "deploy"},
		lo.Ternary(
			rootConfig.AWS.Region == "", nil,
			[]string{"--region", rootConfig.AWS.Region},
		),
		{
			"--template-file", rootConfig.Template.Path,
			"--stack-name", stackName,
			"--no-fail-on-empty-changeset",
		},
		lo.Ternary(
			len(rootConfig.Template.Capabilities) == 0, nil,
			lo.Flatten([][]string{{"--capabilities"}, rootConfig.Template.Capabilities}),
		),
		{"--parameter-overrides"},
		allParameters,
	})

	awsCmd := shelley.Command(deployArgs...)
	if helperCreds, ok := getHelperAWSCredentials(); ok {
		awsCmd.SecretEnv("AWS_ACCESS_KEY_ID", helperCreds.AccessKeyID)
		awsCmd.SecretEnv("AWS_SECRET_ACCESS_KEY", helperCreds.SecretAccessKey)
		if helperCreds.SessionToken != "" {
			awsCmd.SecretEnv("AWS_SESSION_TOKEN", helperCreds.SessionToken)
		}
	}

	shelley.ExitIfError(awsCmd.Run())
	runOutputs(cmd, args)
}

func getLambdaPackageParameters() ([]string, error) {
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

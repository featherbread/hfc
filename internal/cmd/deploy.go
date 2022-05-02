package cmd

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"go.alexhamlin.co/hfc/internal/config"
	"go.alexhamlin.co/hfc/internal/shelley"
	"golang.org/x/exp/slices"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [flags] stack [parameters]",
	Short: "Deploy a CloudFormation stack using the latest uploaded binary",
	Run:   runDeploy,
	Args:  cobra.MinimumNArgs(1),
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
	stackIdx := slices.IndexFunc(rootConfig.Stacks,
		func(s config.StackConfig) bool { return s.Name == stackName })
	if stackIdx < 0 {
		log.Fatalf("stack %s is not configured", stackName)
	}

	var capabilityArgs []string
	if len(rootConfig.Template.Capabilities) > 0 {
		capabilityArgs = append([]string{"--capabilities"}, rootConfig.Template.Capabilities...)
	}

	var parameterOverrideArgs []string
	stack := rootConfig.Stacks[stackIdx]
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

	description, err := shelley.
		Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName).
		NoStderr().
		Text()
	if err != nil {
		log.Print("unable to read stack info, will skip printing output")
		return
	}

	var stackInfo struct {
		Stacks []struct {
			Outputs []struct {
				OutputKey   string
				OutputValue string
				Description string
			}
		}
	}
	if err := json.Unmarshal([]byte(description), &stackInfo); err != nil || len(stackInfo.Stacks) < 1 {
		log.Print("unable to read stack info, will skip printing output")
		return
	}

	for _, output := range stackInfo.Stacks[0].Outputs {
		log.Printf("%s (%s):\n\t%s", output.Description, output.OutputKey, output.OutputValue)
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

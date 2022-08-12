package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Summarize the current deployment status of all stacks",
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	latestPackageRaw, err := os.ReadFile(rootState.LatestLambdaPackagePath())
	latestPackage := strings.TrimSpace(string(latestPackageRaw))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fmt.Printf("LATEST BUILD: (no current build)\n\n")
	case err != nil:
		log.Fatal(err)
	default:
		fmt.Printf("LATEST BUILD: %s\n\n", latestPackage)
	}

	fmt.Printf("DEPLOYED VERSIONS:\n")

	if len(rootConfig.Stacks) == 0 {
		fmt.Println("(no stacks configured)")
		return
	}

	const (
		minwidth = 1
		tabwidth = 8
		padding  = 2
		padchar  = ' '
		flags    = 0
	)
	tw := tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, padchar, flags)
	lines := make(chan string, 1)

	group, ctx := errgroup.WithContext(context.Background())

	group.Go(func() error {
		for line := range lines {
			if _, err := io.WriteString(tw, line); err != nil {
				return err
			}
		}
		return tw.Flush()
	})

	group.Go(func() error {
		defer close(lines)

		cfnClient := cloudformation.NewFromConfig(awsConfig)
		group, ctx := errgroup.WithContext(ctx)
		group.SetLimit(5) // TODO: This is arbitrary, is there a specific limit that makes sense?

		for _, stack := range rootConfig.Stacks {
			name := stack.Name
			group.Go(func() error {
				key, err := getStackS3Key(ctx, cfnClient, name)

				line := name
				switch {
				case err == nil:
					line += "\t" + key
					if key == latestPackage {
						line += "\t(latest)"
					} else {
						line += "\t(out of date)"
					}

				case err != nil:
					line += "\t(no data)"
				}

				lines <- line + "\n"
				return nil
			})
		}

		return group.Wait()
	})

	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
}

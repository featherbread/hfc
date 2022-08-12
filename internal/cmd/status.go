package cmd

import (
	"context"
	"errors"
	"fmt"
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

	cfnClient := cloudformation.NewFromConfig(awsConfig)
	var group errgroup.Group
	group.SetLimit(5) // TODO: This is arbitrary, is there a specific limit that makes sense?
	stackS3Keys := make([]string, len(rootConfig.Stacks))
	for i, stack := range rootConfig.Stacks {
		i, name := i, stack.Name
		group.Go(func() error {
			// Errors here are intentionally not hard failures. One misconfigured or
			// not-yet-deployed stack should not prevent reporting for other stacks.
			if key, err := getStackS3Key(context.Background(), cfnClient, name); err == nil {
				stackS3Keys[i] = key
			}
			return nil
		})
	}
	group.Wait()

	const (
		minwidth = 1
		tabwidth = 8
		padding  = 2
		padchar  = ' '
		flags    = 0
	)
	tw := tabWriter{
		Writer: tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, padchar, flags),
	}

	for i, stack := range rootConfig.Stacks {
		tw.WriteString(stack.Name)
		key := stackS3Keys[i]
		if key != "" {
			tw.WriteByte('\t')
			tw.WriteString(key)
			if key == latestPackage {
				tw.WriteString("\t(latest)")
			} else {
				tw.WriteString("\t(out of date)")
			}
		} else {
			tw.WriteString("\t(no data)")
		}
		tw.WriteByte('\n')
	}

	if err := tw.Flush(); err != nil {
		log.Fatal(err)
	}
}

type tabWriter struct {
	*tabwriter.Writer
	err error
}

func (b *tabWriter) Write(buf []byte) (n int, err error) {
	if b.err != nil {
		return 0, b.err
	}
	n, b.err = b.Writer.Write(buf)
	return n, b.err
}

func (b *tabWriter) WriteString(s string) error {
	_, err := b.Write([]byte(s))
	return err
}

func (b *tabWriter) WriteByte(x byte) error {
	_, err := b.Write([]byte{x})
	return err
}

func (b *tabWriter) Flush() error {
	if b.err != nil {
		return b.err
	}
	return b.Writer.Flush()
}

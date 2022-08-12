package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var cleanUploadsCmd = &cobra.Command{
	Use:   "clean-uploads",
	Short: "Remove uploaded Lambda packages not used by any configured stack",
	Long: `Remove uploaded Lambda packages not used by any configured stack

The clean-uploads command deletes S3 objects that start with the prefix in the
hfc upload configuration but are not in use by any configured stack.

If the S3 bucket for hfc uploads is shared with other projects, and no prefix is
defined in the hfc upload configuration, clean-uploads may delete unrelated
objects from the bucket.

The command prints the keys of objects to be deleted and requests confirmation
before proceeding.
`,
	Run: runCleanUploads,
}

func init() {
	rootCmd.AddCommand(cleanUploadsCmd)
}

func runCleanUploads(cmd *cobra.Command, args []string) {
	cfnClient := cloudformation.NewFromConfig(awsConfig)
	s3Client := s3.NewFromConfig(awsConfig)

	var candidateS3Keys, stackS3Keys []string
	group, ctx := errgroup.WithContext(context.Background())
	group.Go(func() (err error) {
		candidateS3Keys, err = getUploadedS3Keys(ctx, s3Client)
		return
	})
	group.Go(func() (err error) {
		stackS3Keys, err = getAllStackS3Keys(ctx, cfnClient)
		return
	})
	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}

	var (
		candidateKeys = mapset.NewThreadUnsafeSet(candidateS3Keys...)
		stackKeys     = mapset.NewThreadUnsafeSet(stackS3Keys...)

		keepKeys   = candidateKeys.Intersect(stackKeys).ToSlice()
		deleteKeys = candidateKeys.Difference(stackKeys).ToSlice()
	)
	sort.Strings(keepKeys)
	sort.Strings(deleteKeys)

	if len(deleteKeys) == 0 {
		log.Print("Bucket is clean enough, no objects to delete.")
		return
	}

	if len(keepKeys) > 0 {
		log.Print("Will keep the following in-use objects:\n\n")
		for _, key := range keepKeys {
			fmt.Fprintf(log.Writer(), "\t%s\n", key)
		}
		fmt.Fprint(log.Writer(), "\n")
	}

	log.Print("Will delete the following unused objects:\n\n")
	for _, key := range deleteKeys {
		fmt.Fprintf(log.Writer(), "\t%s\n", key)
	}
	fmt.Fprint(log.Writer(), "\n"+log.Prefix()+"Press Enter to continue...")
	fmt.Scanln()

	deleteIdentifiers := make([]types.ObjectIdentifier, len(deleteKeys))
	for i, key := range deleteKeys {
		deleteIdentifiers[i] = types.ObjectIdentifier{
			// Reminder: &key will create pain and sadness here.
			// https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
			Key: aws.String(key),
		}
	}
	output, err := s3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: aws.String(rootConfig.Upload.Bucket),
		Delete: &types.Delete{
			Objects: deleteIdentifiers,
			Quiet:   true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(output.Errors) > 0 {
		for _, e := range output.Errors {
			log.Printf("failed to delete %s: %s", *e.Key, *e.Message)
		}
		os.Exit(1)
	}

	log.Print("Deleted all unused objects.")
}

// getUploadedS3Keys returns the S3 keys of all Lambda packages currently in the
// deployment bucket, in the standard order returned by S3.
//
// The current implementation is limited to returning 1,000 keys.
func getUploadedS3Keys(ctx context.Context, s3Client *s3.Client) ([]string, error) {
	output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(rootConfig.Upload.Bucket),
		Prefix: aws.String(rootConfig.Upload.Prefix),
	})
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(output.Contents))
	for i, object := range output.Contents {
		keys[i] = *object.Key
	}
	return keys, nil
}

// getAllStackS3Keys returns the S3 key for the Lambda package currently in use
// by each known stack, as a list in the same order in which stacks are defined
// in the hfc configuration.
func getAllStackS3Keys(ctx context.Context, cfnClient *cloudformation.Client) ([]string, error) {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(5) // TODO: This is arbitrary, is there a specific limit that makes sense?

	keys := make([]string, len(rootConfig.Stacks))
	for i, stack := range rootConfig.Stacks {
		i, stack := i, stack
		group.Go(func() (err error) {
			keys[i], err = getStackS3Key(ctx, cfnClient, stack.Name)
			return
		})
	}

	err := group.Wait()
	return keys, err
}

func getStackS3Key(ctx context.Context, cfnClient *cloudformation.Client, stackName string) (string, error) {
	stack, err := cfnClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return "", err
	}

	for _, p := range stack.Stacks[0].Parameters {
		if *p.ParameterKey == "CodeS3Key" {
			return *p.ParameterValue, nil
		}
	}
	return "", fmt.Errorf("stack %s deployed without CodeS3Key parameter", stackName)
}

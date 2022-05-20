package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var cleanRepositoryCmd = &cobra.Command{
	Use:   "clean-repository",
	Short: "Remove uploaded binaries not used by any configured stacks",
	Run:   runCleanRepository,
}

func init() {
	rootCmd.AddCommand(cleanRepositoryCmd)
}

func runCleanRepository(cmd *cobra.Command, args []string) {
	cfnClient := cloudformation.NewFromConfig(awsConfig)
	ecrClient := ecr.NewFromConfig(awsConfig)

	group, ctx := errgroup.WithContext(context.Background())

	var repoTagList []string
	group.Go(goGetRepoTags(ctx, ecrClient, &repoTagList))

	stackTagList := make([]string, len(rootConfig.Stacks))
	for i, stack := range rootConfig.Stacks {
		// TODO: Limit concurrent stack reads?
		group.Go(goGetStackTag(ctx, cfnClient, stack.Name, &stackTagList[i]))
	}

	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}

	repoTags := mapset.NewThreadUnsafeSet(repoTagList...)
	stackTags := mapset.NewThreadUnsafeSet(stackTagList...)

	if repoTags.Cardinality() != len(repoTagList) {
		log.Fatal("repository tag list contained duplicate tags")
	}

	keepTags := repoTags.Intersect(stackTags).ToSlice()
	deleteTags := repoTags.Difference(stackTags).ToSlice()
	sort.Strings(keepTags)
	sort.Strings(deleteTags)

	if len(deleteTags) == 0 {
		log.Printf("Repository is clean enough, no tags to delete.")
		return
	}

	// BatchDeleteImage supports up to 100 IDs at a time.
	// https://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_BatchDeleteImage.html#ECR-BatchDeleteImage-request-imageIds
	if len(deleteTags) > 100 {
		log.Print("Repository has more than 100 unused tags. Will only delete the first 100.")
		deleteTags = deleteTags[:100]
	}

	log.Printf("Tags to keep:   %v", keepTags)
	log.Printf("Tags to delete: %v", deleteTags)
	fmt.Fprint(log.Writer(), log.Prefix()+"Press Enter to continue...")
	fmt.Scanln()

	ids := make([]types.ImageIdentifier, len(deleteTags))
	for i, tag := range deleteTags {
		ids[i] = types.ImageIdentifier{ImageTag: aws.String(tag)}
	}
	output, err := ecrClient.BatchDeleteImage(context.Background(), &ecr.BatchDeleteImageInput{
		RepositoryName: aws.String(rootConfig.Repository.Name),
		ImageIds:       ids,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, id := range output.ImageIds {
		log.Printf("Deleted tag %s.", *id.ImageTag)
	}
	if len(output.Failures) == 0 {
		return
	}

	for _, failure := range output.Failures {
		msg := "failed to delete tag"
		if failure.ImageId != nil && failure.ImageId.ImageTag != nil {
			msg = msg + " " + *failure.ImageId.ImageTag
		}
		if failure.FailureReason != nil {
			msg = msg + ": " + *failure.FailureReason
		}
		log.Print(msg)
	}
	os.Exit(1)
}

func goGetRepoTags(ctx context.Context, client *ecr.Client, repoTags *[]string) func() error {
	return func() error {
		images, err := client.DescribeImages(ctx, &ecr.DescribeImagesInput{
			RepositoryName: aws.String(rootConfig.Repository.Name),
		})
		if err != nil {
			return err
		}

		for _, image := range images.ImageDetails {
			*repoTags = append(*repoTags, image.ImageTags...)
		}
		return nil
	}
}

func goGetStackTag(ctx context.Context, client *cloudformation.Client, stackName string, tag *string) func() error {
	return func() error {
		stack, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return err
		}

		for _, p := range stack.Stacks[0].Parameters {
			if *p.ParameterKey == "ImageUri" {
				parts := strings.Split(*p.ParameterValue, ":")
				*tag = parts[len(parts)-1]
				return nil
			}
		}

		return fmt.Errorf("stack %s deployed without ImageUri parameter", stackName)
	}
}

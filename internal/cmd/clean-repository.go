package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.alexhamlin.co/hfc/internal/shelley"
	"golang.org/x/exp/slices"
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
	repoTagsJSON := shelley.GetOrExit(shelley.
		Command(
			"aws", "ecr", "describe-images",
			"--repository-name", rootConfig.Repository.Name,
			"--query", "imageDetails[].imageTags[]",
		).
		Text())
	var repoTags []string
	if err := json.Unmarshal([]byte(repoTagsJSON), &repoTags); err != nil {
		log.Fatalf("tags list is not valid JSON:\n%s", repoTagsJSON)
	}

	stackTags := make([]string, len(rootConfig.Stacks))
	for i, stack := range rootConfig.Stacks {
		image := shelley.GetOrExit(shelley.
			Command(
				"aws", "cloudformation", "describe-stacks",
				"--stack-name", stack.Name,
				"--query", "Stacks[0].Parameters[?ParameterKey=='ImageUri'].ParameterValue",
				"--output", "text",
			).
			Text())
		parts := strings.Split(image, ":")
		stackTags[i] = parts[len(parts)-1]
	}

	sort.Strings(repoTags)
	if len(repoTags) != len(slices.Compact(repoTags)) {
		log.Fatal("repository tag list contained duplicate tags")
	}

	sort.Strings(stackTags)
	stackTags = slices.Compact(stackTags)

	var keepTags, deleteTags []string
	for len(repoTags) > 0 && len(stackTags) > 0 {
		switch {
		case repoTags[0] == stackTags[0]:
			keepTags = append(keepTags, repoTags[0])
			repoTags, stackTags = repoTags[1:], stackTags[1:]
		case repoTags[0] < stackTags[0]:
			deleteTags = append(deleteTags, repoTags[0])
			repoTags = repoTags[1:]
		case repoTags[0] > stackTags[0]:
			log.Fatalf("stack deployed with image tag not in repository: %s", stackTags[0])
		}
	}
	deleteTags = append(deleteTags, repoTags...)

	if len(deleteTags) == 0 {
		log.Printf("Repository is clean enough, no tags to delete.")
		return
	}

	log.Printf("Tags to keep:   %v", keepTags)
	log.Printf("Tags to delete: %v", deleteTags)
	fmt.Fprint(log.Writer(), log.Prefix()+"Press Enter to continue...")
	fmt.Scanln()

	deleteArgs := []string{
		"aws", "ecr", "batch-delete-image",
		"--repository-name", rootConfig.Repository.Name,
		"--image-ids",
	}
	for _, tag := range deleteTags {
		deleteArgs = append(deleteArgs, "imageTag="+tag)
	}
	shelley.ExitIfError(shelley.Command(deleteArgs...).Run())
}

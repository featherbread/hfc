package cmd

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload the latest binary to the container registry",
	Run:   runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) {
	outputPath, err := rootState.BinaryPath(rootConfig.Project.Name)
	if err != nil {
		log.Fatal(err)
	}

	stat, err := os.Stat(outputPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Fatal("must build a binary before uploading")
	case err != nil:
		log.Fatal(err)
	case !stat.Mode().IsRegular():
		log.Fatalf("%s is not a regular file", outputPath)
	}

	repository := shelley.GetOrExit(shelley.
		Command(
			"aws", "ecr", "describe-repositories",
			"--repository-names", rootConfig.Repository.Name,
			"--query", "repositories[0].repositoryUri", "--output", "text",
		).
		Text())

	registry := strings.SplitN(repository, "/", 2)[0]
	tag := strconv.FormatInt(time.Now().Unix(), 10)
	image := repository + ":" + tag

	authenticated := shelley.GetOrExit(shelley.
		Command("zeroimage", "check-auth", "--push", image).
		Successful())

	if !authenticated {
		shelley.ExitIfError(shelley.
			Command("aws", "ecr", "get-login-password").
			Pipe("zeroimage", "login", "--username", "AWS", "--password-stdin", registry).
			Run())
	}

	shelley.ExitIfError(shelley.
		Command("zeroimage", "build", "--platform", "linux/arm64", "--push", image, outputPath).
		Run())

	if err := os.WriteFile(rootState.LatestImagePath(), []byte(image), 0644); err != nil {
		log.Fatal(err)
	}
}

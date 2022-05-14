package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
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

	ecrClient := ecr.NewFromConfig(awsConfig)
	repository, err := getRepositoryURI(ecrClient, rootConfig.Repository.Name)
	if err != nil {
		log.Fatal(err)
	}

	registry := strings.SplitN(repository, "/", 2)[0]
	tag := strconv.FormatInt(time.Now().Unix(), 10)
	image := repository + ":" + tag

	authenticated, err := shelley.Command("zeroimage", "check-auth", "--push", image).Test()
	if err != nil {
		log.Fatal(err)
	}

	if !authenticated {
		password, err := getLoginPassword(ecrClient)
		if err != nil {
			log.Fatal(err)
		}

		shelley.ExitIfError(shelley.
			Command("zeroimage", "login", "--username", "AWS", "--password-stdin", registry).
			Stdin(strings.NewReader(password)).
			Run())
	}

	shelley.ExitIfError(shelley.
		Command("zeroimage", "build", "--platform", "linux/arm64", "--push", image, outputPath).
		Run())

	if err := os.WriteFile(rootState.LatestImagePath(), []byte(image), 0644); err != nil {
		log.Fatal(err)
	}
}

func getRepositoryURI(client *ecr.Client, name string) (string, error) {
	output, err := client.DescribeRepositories(context.Background(), &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{name},
	})
	if err != nil {
		return "", err
	}
	return *output.Repositories[0].RepositoryUri, nil
}

func getLoginPassword(client *ecr.Client) (string, error) {
	output, err := client.GetAuthorizationToken(context.Background(), nil)
	if err != nil {
		return "", err
	}

	rawToken := *output.AuthorizationData[0].AuthorizationToken
	token, err := base64.StdEncoding.DecodeString(rawToken)
	if err != nil {
		return "", fmt.Errorf("invalid base64 data from ECR: %w", err)
	}

	password := strings.SplitN(string(token), ":", 2)[1]
	return password, nil
}

package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	switch {
	case rootConfig.Bucket.Name != "":
		uploadAsLambdaPackage(outputPath)
	case rootConfig.Repository.Name != "":
		uploadAsContainerImage(outputPath)
	default:
		log.Fatal("no valid upload configuration available")
	}
}

func uploadAsLambdaPackage(outputPath string) {
	log.Print("Building deployment package")
	lambdaPackage, err := createLambdaPackage(outputPath)
	if err != nil {
		log.Fatalf("failed to create deployment package: %v", err)
	}

	var (
		s3Client   = s3.NewFromConfig(awsConfig)
		bucket     = rootConfig.Bucket.Name
		key        = strconv.FormatInt(time.Now().Unix(), 10) + ".zip"
		hashBytes  = sha256.Sum256(lambdaPackage)
		hashString = base64.StdEncoding.EncodeToString(hashBytes[:])
	)

	log.Printf("Uploading deployment package to s3://%s/%s", bucket, key)
	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:         aws.String(bucket),
		Key:            aws.String(key),
		Body:           bytes.NewReader(lambdaPackage),
		ContentLength:  int64(len(lambdaPackage)),
		ChecksumSHA256: aws.String(hashString),
	})
	if err != nil {
		log.Fatalf("failed to upload deployment package: %v", err)
	}

	if err := os.WriteFile(rootState.LatestLambdaPackagePath(), append([]byte(key), '\n'), 0644); err != nil {
		log.Fatal(err)
	}
}

func createLambdaPackage(handlerPath string) ([]byte, error) {
	handlerBinary, err := os.Open(handlerPath)
	if err != nil {
		return nil, err
	}
	defer handlerBinary.Close()

	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	hw, err := zw.Create("bootstrap")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(hw, handlerBinary); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func uploadAsContainerImage(outputPath string) {
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

	if err := os.WriteFile(rootState.LatestImagePath(), append([]byte(image), '\n'), 0644); err != nil {
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

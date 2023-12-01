package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:    "upload",
	Short:  "Upload the latest binary to the container registry",
	PreRun: initializePreRun,
	Run:    runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) {
	outputPath, err := rootState.BinaryPath(rootConfig.Project.Name)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Building deployment package")
	lambdaPackage, err := createLambdaPackage(outputPath)
	if err != nil {
		log.Fatalf("failed to create deployment package: %v", err)
	}

	var (
		s3Client   = s3.NewFromConfig(awsConfig)
		bucket     = rootConfig.Upload.Bucket
		key        = rootConfig.Upload.Prefix + strconv.FormatInt(time.Now().Unix(), 10) + ".zip"
		hashBytes  = sha256.Sum256(lambdaPackage)
		hashString = base64.StdEncoding.EncodeToString(hashBytes[:])
	)

	log.Printf("Uploading deployment package to s3://%s/%s", bucket, key)
	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:         aws.String(bucket),
		Key:            aws.String(key),
		Body:           bytes.NewReader(lambdaPackage),
		ContentLength:  aws.Int64(int64(len(lambdaPackage))),
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
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, errors.New("must build a binary before uploading")
	case err != nil:
		return nil, err
	}
	defer handlerBinary.Close()

	var output bytes.Buffer
	zipWriter := zip.NewWriter(&output)
	handlerWriter, err := zipWriter.Create("bootstrap")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(handlerWriter, handlerBinary); err != nil {
		return nil, err
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

package cmd

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log"
	"os/exec"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

var getAWSConfig = sync.OnceValue(func() aws.Config {
	options := []func(*config.LoadOptions) error{
		config.WithRegion(rootConfig.AWS.Region),
	}

	if helperCreds, ok := getHelperAWSCredentials(); ok {
		options = append(options, config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{Value: helperCreds}))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), options...)
	if err != nil {
		log.Fatal(err)
	}
	return cfg
})

var getHelperAWSCredentials = sync.OnceValues(func() (aws.Credentials, bool) {
	const helperName = "hfc-aws-credentials"

	helperPath, err := exec.LookPath(helperName)
	if err != nil {
		return aws.Credentials{}, false
	}

	var helperStdout bytes.Buffer
	helper := exec.Command(helperPath)
	helper.Stdout = &helperStdout
	err = helper.Run()
	if err != nil {
		log.Fatalf("%s helper failed: %v", helperName, err)
	}

	var credentials aws.Credentials
	err = json.Unmarshal(helperStdout.Bytes(), &credentials)
	if err != nil {
		log.Fatalf("%s produced invalid JSON: %v", helperName, err)
	}

	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" {
		log.Fatalf(
			"%s produced incomplete AWS credentials: need at least AccessKeyID and SecretAccessKey",
			helperName)
	}

	return credentials, true
})

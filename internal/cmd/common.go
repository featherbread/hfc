package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// getStackS3Key returns the full S3 key (including prefix) for the Lambda
// package currently in use by the named stack.
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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoad(t *testing.T) {
	want := Config{
		Project: ProjectConfig{
			Name: "hfc",
		},
		AWS: AWSConfig{
			Region: "us-west-2",
		},
		Build: BuildConfig{
			Path: "./cmd/hfc",
			Tags: []string{"grpcnotrace"},
		},
		Upload: UploadConfig{
			Bucket: "hfc",
		},
		Template: TemplateConfig{
			Path:         "CloudFormation.yaml",
			Capabilities: []string{"CAPABILITY_IAM"},
		},
		Stacks: []StackConfig{{
			Name:       "HFCStaging",
			Parameters: map[string]string{"Environment": "staging"},
		}, {
			Name:       "HFCProduction",
			Parameters: map[string]string{"Environment": "production"},
		}},
	}

	t.Chdir("testdata")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}
}

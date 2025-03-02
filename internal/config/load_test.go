package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
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

	switchBack := switchDir("testdata")
	defer switchBack()

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result (-want +got):\n%s", diff)
	}
}

func switchDir(dir string) (switchBack func()) {
	original := lo.Must(os.Getwd())
	lo.Must0(os.Chdir(filepath.Join(original, dir)))
	return func() { lo.Must0(os.Chdir(original)) }
}

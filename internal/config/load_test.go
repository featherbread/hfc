package config

import (
	"os"
	"path/filepath"
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
		},
		Repository: RepositoryConfig{
			Name: "hfc",
		},
		Template: TemplateConfig{
			Path:         "CloudFormation.yaml",
			Capabilities: []string{"CAPABILITY_IAM"},
			Outputs: []TemplateOutputConfig{{
				Help: "Access the API at the following URL:",
				Key:  "ApiUrl",
			}},
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
	original, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if err := os.Chdir(filepath.Join(original, dir)); err != nil {
		panic(err)
	}

	return func() {
		if err := os.Chdir(original); err != nil {
			panic(err)
		}
	}
}

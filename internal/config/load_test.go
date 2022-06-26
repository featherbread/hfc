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
		Build: BuildConfig{
			Path: "./cmd/hfc",
		},
		Repository: RepositoryConfig{
			Name: "hfc",
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

func TestCheck(t *testing.T) {
	testCases := []struct {
		Description string
		Config      Config
		Valid       bool
	}{{
		Description: "only repository",
		Config:      Config{Repository: RepositoryConfig{Name: "test"}},
		Valid:       true,
	}, {
		Description: "only bucket",
		Config:      Config{Bucket: BucketConfig{Name: "test"}},
		Valid:       true,
	}, {
		Description: "both",
		Config: Config{
			Repository: RepositoryConfig{Name: "test"},
			Bucket:     BucketConfig{Name: "test"},
		},
		Valid: true,
	}, {
		Description: "neither",
		Config:      Config{},
		Valid:       false,
	}}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			ok, err := Check(tc.Config)
			if ok != (err == nil) {
				t.Errorf("ok == %v inconsistent with err == %v", ok, err)
			}
			if ok != tc.Valid {
				t.Fatalf("Check() == %v, want %v", ok, tc.Valid)
			}
		})
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

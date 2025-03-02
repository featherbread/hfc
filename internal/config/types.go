package config

import "github.com/samber/lo"

// Config represents a full configuration.
type Config struct {
	Project  ProjectConfig  `toml:"project"`
	AWS      AWSConfig      `toml:"aws"`
	Build    BuildConfig    `toml:"build"`
	Upload   UploadConfig   `toml:"upload"`
	Template TemplateConfig `toml:"template"`
	Stacks   []StackConfig  `toml:"stacks"`
}

// FindStack searches for the stack with the given name. If no stack is defined
// with the provided name, FindStack returns ok == false.
func (c *Config) FindStack(name string) (stack StackConfig, ok bool) {
	return lo.Find(c.Stacks, func(s StackConfig) bool { return s.Name == name })
}

// ProjectConfig represents the configuration for this project, which is
// expected to be common across all possible deployments.
type ProjectConfig struct {
	Name string `toml:"name"`
}

// AWSConfig represents the configuration for all AWS operations in this
// project.
type AWSConfig struct {
	Region string `toml:"region"`
}

// BuildConfig represents the configuration for building a deployable Go binary.
type BuildConfig struct {
	Path string   `toml:"path"`
	Tags []string `toml:"tags"`
}

// UploadConfig represents the configuration for uploading a Go binary in a
// Lambda .zip archive to an Amazon S3 bucket.
type UploadConfig struct {
	Bucket string `toml:"bucket"`
	Prefix string `toml:"prefix"`
}

// TemplateConfig represents the configuration of the AWS CloudFormation
// template associated with the deployment.
type TemplateConfig struct {
	Path         string   `toml:"path"`
	Capabilities []string `toml:"capabilities"`
}

// StackConfig represents the configuration of an AWS CloudFormation stack, a
// specific deployment of the CloudFormation template with a unique set of
// parameters.
type StackConfig struct {
	Name       string            `toml:"name"`
	Parameters map[string]string `toml:"parameters"`
}

package config

// Config represents a full configuration.
type Config struct {
	Project    ProjectConfig    `toml:"project"`
	AWS        AWSConfig        `toml:"aws"`
	Build      BuildConfig      `toml:"build"`
	Repository RepositoryConfig `toml:"repository"`
	Template   TemplateConfig   `toml:"template"`
	Stacks     []StackConfig    `toml:"stacks"`
}

// ProjectConfig represents the configuration for this project, which is
// expected to be common across all possible deployments.
type ProjectConfig struct {
	Name string `toml:"name"`
}

// AWSConfig represents the common configuration for usage of AWS APIs.
type AWSConfig struct {
	Region string `toml:"region"`
}

// BuildConfig represents the configuration for building a deployable Go binary.
type BuildConfig struct {
	Path string `toml:"path"`
}

// RepositoryConfig represents the configuration for uploading a containerized
// Go binary to the AWS Elastic Container Registry.
type RepositoryConfig struct {
	Name string `toml:"name"`
}

// TemplateConfig represents the configuration of the AWS CloudFormation
// template associated with the deployment.
type TemplateConfig struct {
	Path         string                 `toml:"path"`
	Capabilities []string               `toml:"capabilities"`
	Outputs      []TemplateOutputConfig `toml:"outputs"`
}

// TemplateOutputConfig represents an output of the AWS CloudFormation template,
// which may be printed after a successful deployment to provide usage
// instructions.
type TemplateOutputConfig struct {
	Key  string `toml:"key"`
	Help string `toml:"help"`
}

// StackConfig represents the configuration of an AWS CloudFormation stack, a
// specific deployment of the CloudFormation template with a unique set of
// parameters.
type StackConfig struct {
	Name       string            `toml:"name"`
	Parameters map[string]string `toml:"parameters"`
}

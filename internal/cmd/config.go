package cmd

type ProjectConfig struct {
	Project  ProjectMetaConfig `toml:"project"`
	AWS      AWSConfig         `toml:"aws"`
	Build    BuildConfig       `toml:"build"`
	Template TemplateConfig    `toml:"template"`
}

type LocalConfig struct {
	AWS        AWSConfig        `toml:"aws"`
	Repository RepositoryConfig `toml:"repository"`
	Stacks     []StackConfig    `toml:"stacks"`
}

type AWSConfig struct {
	Region string `toml:"region"`
}

func (a AWSConfig) Merge(b AWSConfig) AWSConfig {
	if b.Region != "" {
		a.Region = b.Region
	}
	return a
}

type ProjectMetaConfig struct {
	Name string `toml:"name"`
}

type BuildConfig struct {
	Path string `toml:"path"`
}

type RepositoryConfig struct {
	Name string `toml:"name"`
}

type TemplateConfig struct {
	Path         string                 `toml:"path"`
	Capabilities []string               `toml:"capabilities"`
	Outputs      []TemplateOutputConfig `toml:"outputs"`
}

type TemplateOutputConfig struct {
	Key  string `toml:"key"`
	Help string `toml:"help"`
}

type StackConfig struct {
	Name       string            `toml:"name"`
	Parameters map[string]string `toml:"parameters"`
}

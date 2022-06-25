package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"
)

const (
	// Filename is the base name of the configuration file.
	Filename = "hfc.toml"
	// LocalFilename is the base name of the local configuration file, whose
	// values are deeply merged with the base configuration.
	LocalFilename = "hfc.local.toml"
)

// Load automatically loads the full configuration by finding, loading, and
// merging the base and local configurations.
func Load() (Config, error) {
	baseConfigPath, err := FindPath()
	if err != nil {
		return Config{}, err
	}

	baseConfig, err := LoadFile(baseConfigPath)
	if err != nil {
		return Config{}, err
	}

	var localConfig Config
	localConfigPath := filepath.Join(filepath.Dir(baseConfigPath), LocalFilename)
	if _, err := os.Stat(localConfigPath); err == nil {
		localConfig, err = LoadFile(localConfigPath)
		if err != nil {
			return Config{}, err
		}
	}

	config := Merge(baseConfig, localConfig)
	if ok, err := Check(config); !ok {
		return Config{}, err
	}
	return config, nil
}

// FindPath returns the rooted path to the configuration file in the current
// directory or its parents, or an error if it cannot find the file.
func FindPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		fullPath := filepath.Join(dir, Filename)
		stat, err := os.Stat(fullPath)
		if err == nil && !stat.IsDir() {
			return fullPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find %s", Filename)
		}

		dir = parent
	}
}

// LoadFile loads configuration from a TOML file.
func LoadFile(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var config Config
	_, err = toml.NewDecoder(file).Decode(&config)
	return config, err
}

// Merge deeply merges the provided configs, overriding the values in earlier
// configs with those in later configs.
func Merge(configs ...Config) Config {
	var result Config
	for _, config := range configs {
		err := mergo.Merge(&result, config, mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			panic(err)
		}
	}
	return result
}

// Validate checks the provided config for errors that may prevent hfc from
// operating as expected.
func Check(config Config) (ok bool, err error) {
	var (
		hasRepository = (config.Repository != RepositoryConfig{})
		hasBucket     = (config.Bucket != BucketConfig{})
	)
	if hasRepository == hasBucket {
		return false, errors.New("config needs exactly one of [repository] or [bucket]")
	}

	return true, nil
}

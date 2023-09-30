package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Dirname is the name of the state directory next to the configuration file.
const Dirname = ".hfc"

// State represents the state directory for a project.
type State struct {
	path string
}

// Get returns the state associated with the configuration at the provided path,
// creating the state directory if necessary.
func Get(configPath string) (State, error) {
	statePath := filepath.Join(filepath.Dir(configPath), Dirname)

	stat, err := os.Stat(statePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.Mkdir(statePath, fs.ModeDir|0755); err != nil {
			return State{}, err
		}
	case err != nil:
		return State{}, err
	case !stat.IsDir():
		return State{}, fmt.Errorf("cannot create state directory %s, file already exists", statePath)
	}

	return State{path: statePath}, nil
}

// BinaryPath returns the relative file path to the named Go binary in the
// state directory.
func (s State) BinaryPath(name string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fullPath := s.Path("output", name)
	return filepath.Rel(cwd, fullPath)
}

// LatestLambdaPackagePath returns the absolute path to the file containing the
// S3 key of the latest Lambda deployment package.
func (s State) LatestLambdaPackagePath() string {
	return s.Path("latest-lambda-package")
}

// Path returns the absolute file path formed by joining the provided path
// elements to the state directory path.
func (s State) Path(parts ...string) string {
	allParts := append([]string{s.path}, parts...)
	return filepath.Join(allParts...)
}

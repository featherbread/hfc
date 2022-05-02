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

// Path returns the absolute file path formed by joining the provided path
// elements to the state directory path.
func (s State) Path(elem ...string) string {
	return filepath.Join(append([]string{s.path}, elem...)...)
}

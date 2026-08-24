package runtime

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultRootName = ".unishell"
	runtimeDirName  = "runtime"
)

// Paths describes the filesystem locations used by a uniShell runtime
// version.
type Paths struct {
	Root    string
	Runtime string
	Bin     string
	Config  string
}

// NewPaths creates the filesystem layout for the supplied version.
//
// Runtime points to the version directory. Individual Session instances
// derive their own isolated runtime directory beneath it.
func NewPaths(root, version string) (Paths, error) {
	if version == "" {
		return Paths{}, fmt.Errorf("runtime version cannot be empty")
	}

	if root == "" {
		root = filepath.Join(os.TempDir(), defaultRootName)
	}

	root = filepath.Clean(root)
	runtime := filepath.Join(root, runtimeDirName, version)

	return Paths{
		Root:    root,
		Runtime: runtime,
		Bin:     filepath.Join(runtime, "bin"),
		Config:  filepath.Join(runtime, "config"),
	}, nil
}

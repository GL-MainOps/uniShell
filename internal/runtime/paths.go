package runtime

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	runtimeRootName = ".unishell"
	runtimeDirName  = "runtime"
)

// Paths describes the filesystem locations used by a uniShell runtime.
type Paths struct {
	Root    string
	Runtime string
	Bin     string
	Config  string
}

// NewPaths creates the filesystem layout for the supplied version.
//
// No directories are created by this function. It only calculates paths.
func NewPaths(version string) (Paths, error) {
	if version == "" {
		return Paths{}, fmt.Errorf("runtime version cannot be empty")
	}

	root := filepath.Join(os.TempDir(), runtimeRootName)
	runtime := filepath.Join(root, runtimeDirName, version)

	return Paths{
		Root:    root,
		Runtime: runtime,
		Bin:     filepath.Join(runtime, "bin"),
		Config:  filepath.Join(runtime, "config"),
	}, nil
}

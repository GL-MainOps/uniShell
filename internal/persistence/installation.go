package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const runtimePointerFilename = ".unishell-runtime-dir"

func RuntimePointer(home string) (string, error) {
	path := filepath.Join(home, ".local", "bin", runtimePointerFilename)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect runtime path setting: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("runtime path setting %q is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read runtime path setting: %w", err)
	}
	root := strings.TrimSpace(string(data))
	if root == "" {
		return "", fmt.Errorf("runtime path setting %q is empty", path)
	}
	return filepath.Clean(root), nil
}

func SetRuntimePointer(home, root string) error {
	path := filepath.Join(home, ".local", "bin", runtimePointerFilename)
	defaultRoot := filepath.Join(home, ".local", "unishell")
	if filepath.Clean(root) == filepath.Clean(defaultRoot) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove custom runtime path setting: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create runtime path setting directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".unishell-runtime-*.tmp")
	if err != nil {
		return fmt.Errorf("create runtime path setting: %w", err)
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("protect runtime path setting: %w", err)
	}
	if _, err := fmt.Fprintln(file, filepath.Clean(root)); err != nil {
		_ = file.Close()
		return fmt.Errorf("write runtime path setting: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync runtime path setting: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime path setting: %w", err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("save runtime path setting: %w", err)
	}
	return nil
}

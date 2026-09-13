package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/acquisition"
)

func TestInstallBinary(t *testing.T) {
	dir := t.TempDir()

	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "nested", "example")

	if err := os.WriteFile(
		source,
		[]byte("binary"),
		0644,
	); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := installBinary(source, destination); err != nil {
		t.Fatalf("installBinary() error = %v", err)
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}

	if string(data) != "binary" {
		t.Fatalf(
			"destination contents = %q, want %q",
			string(data),
			"binary",
		)
	}

	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Fatalf(
			"destination permissions = %o, want %o",
			info.Mode().Perm(),
			0755,
		)
	}
}

func TestRunRejectsMissingToolDirectory(t *testing.T) {
	root := t.TempDir()

	err := run(
		context.Background(),
		filepath.Join(root, "missing-tools"),
		filepath.Join(root, "bin"),
		filepath.Join(root, "cache"),
		"",
		acquisition.Platform("linux"),
		acquisition.Architecture("amd64"),
		"",
		false,
		nil,
	)

	if err == nil {
		t.Fatal("run() error = nil, want missing-tool-directory error")
	}
}

func TestRunAcceptsEmptyProfile(t *testing.T) {
	root := t.TempDir()

	err := run(
		context.Background(),
		filepath.Join(root, "missing-tools"),
		filepath.Join(root, "bin"),
		filepath.Join(root, "cache"),
		"",
		acquisition.Platform("linux"),
		acquisition.Architecture("amd64"),
		"",
		false,
		nil,
	)

	if err == nil {
		t.Fatal("run() error = nil, want missing-tool-directory error")
	}
}

func TestRunRejectsReservedCommonProfile(t *testing.T) {
	root := t.TempDir()

	toolsDir := filepath.Join(root, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatalf("create tools directory: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(toolsDir, "example.toml"),
		[]byte(`
[[tools]]
name = "example"
profiles = ["common"]

[[tools.artifacts]]
name = "example"
`),
		0644,
	); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	err := run(
		context.Background(),
		toolsDir,
		filepath.Join(root, "bin"),
		filepath.Join(root, "cache"),
		"",
		acquisition.Platform("linux"),
		acquisition.Architecture("amd64"),
		acquisition.CommonProfile,
		false,
		nil,
	)

	if err == nil {
		t.Fatal("run() error = nil, want reserved-profile error")
	}
}

func TestMaterializeCachedToolsSelectsProfileTools(t *testing.T) {
	root := t.TempDir()

	sourceDir := filepath.Join(root, "source")
	outputDir := filepath.Join(root, "output")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}

	files := map[string]string{
		"common":   "common-binary",
		"k8s-tool": "k8s-binary",
	}

	for name, data := range files {
		if err := os.WriteFile(
			filepath.Join(sourceDir, name),
			[]byte(data),
			0755,
		); err != nil {
			t.Fatalf("write cached binary %q: %v", name, err)
		}
	}

	tools := []acquisition.Tool{
		{
			Name:     "common",
			Profiles: []string{acquisition.CommonProfile},
			Artifacts: []acquisition.Artifact{
				{BinaryName: "common"},
			},
		},
		{
			Name:     "k8s-tool",
			Profiles: []string{"k8s"},
			Artifacts: []acquisition.Artifact{
				{BinaryName: "k8s-tool"},
			},
		},
	}

	if err := materializeCachedTools(
		tools,
		sourceDir,
		outputDir,
	); err != nil {
		t.Fatalf("materializeCachedTools() error = %v", err)
	}

	for name, want := range files {
		if name == "security" {
			continue
		}

		data, err := os.ReadFile(
			filepath.Join(outputDir, name),
		)
		if err != nil {
			t.Fatalf("read materialized %q: %v", name, err)
		}

		if string(data) != want {
			t.Fatalf(
				"materialized %q = %q, want %q",
				name,
				data,
				want,
			)
		}
	}

	if _, err := os.Stat(
		filepath.Join(outputDir, "security"),
	); !os.IsNotExist(err) {
		t.Fatal("security tool was materialized unexpectedly")
	}
}

func TestMaterializeCachedToolsRejectsMissingBinary(t *testing.T) {
	root := t.TempDir()

	tools := []acquisition.Tool{
		{
			Name:     "missing",
			Profiles: []string{"k8s"},
			Artifacts: []acquisition.Artifact{
				{BinaryName: "missing"},
			},
		},
	}

	err := materializeCachedTools(
		tools,
		filepath.Join(root, "source"),
		filepath.Join(root, "output"),
	)

	if err == nil {
		t.Fatal(
			"materializeCachedTools() error = nil, want missing-binary error",
		)
	}
}

func TestResolveProfile(t *testing.T) {
	profile, err := resolveProfile("  k8s  ")
	if err != nil {
		t.Fatalf("resolveProfile() error = %v", err)
	}

	if profile != "k8s" {
		t.Fatalf(
			"resolveProfile() = %q, want %q",
			profile,
			"k8s",
		)
	}
}

func TestResolveProfileAllowsEmpty(t *testing.T) {
	profile, err := resolveProfile("")
	if err != nil {
		t.Fatalf("resolveProfile() error = %v", err)
	}

	if profile != "" {
		t.Fatalf(
			"resolveProfile() = %q, want empty profile",
			profile,
		)
	}
}

func TestResolveProfileRejectsMultipleProfiles(t *testing.T) {
	_, err := resolveProfile("k8s,security")
	if err == nil {
		t.Fatal(
			"resolveProfile() error = nil, want multiple-profile error",
		)
	}
}

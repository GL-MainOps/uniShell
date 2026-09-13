package acquisition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeToolMetadata(
	t *testing.T,
	dir string,
	name string,
	contents string,
) string {
	t.Helper()

	path := filepath.Join(dir, name)

	if err := os.WriteFile(
		path,
		[]byte(contents),
		0600,
	); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	return path
}

const validToolMetadata = `
[[tools]]
name = "example"
profiles = ["common", "k8s"]

[[tools.artifacts]]
version = "1.0.0"
platform = "linux"
architecture = "amd64"
archive_type = ""
binary_path = ""
binary_name = "example"
checksum = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

[tools.artifacts.source]
kind = "direct-url"

[tools.artifacts.source.direct_url]
url = "https://example.com/example"
`

func TestLoadToolsLoadsMultipleDefinitionsDeterministically(t *testing.T) {
	dir := t.TempDir()

	writeToolMetadata(
		t,
		dir,
		"z-tool.toml",
		strings.Replace(
			validToolMetadata,
			`name = "example"`,
			`name = "z-tool"`,
			1,
		),
	)

	writeToolMetadata(
		t,
		dir,
		"a-tool.toml",
		strings.Replace(
			validToolMetadata,
			`name = "example"`,
			`name = "a-tool"`,
			1,
		),
	)

	writeToolMetadata(
		t,
		dir,
		"README.txt",
		"not TOML metadata",
	)

	tools, err := LoadTools(dir)
	if err != nil {
		t.Fatalf("LoadTools() error = %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("LoadTools() returned %d tools, want 2", len(tools))
	}

	if tools[0].Name != "a-tool" {
		t.Fatalf(
			"tools[0].Name = %q, want %q",
			tools[0].Name,
			"a-tool",
		)
	}

	if tools[1].Name != "z-tool" {
		t.Fatalf(
			"tools[1].Name = %q, want %q",
			tools[1].Name,
			"z-tool",
		)
	}
}

func TestLoadToolsRejectsDuplicateToolNames(t *testing.T) {
	dir := t.TempDir()

	writeToolMetadata(
		t,
		dir,
		"first.toml",
		validToolMetadata,
	)

	writeToolMetadata(
		t,
		dir,
		"second.toml",
		validToolMetadata,
	)

	_, err := LoadTools(dir)
	if err == nil {
		t.Fatal("LoadTools() error = nil, want duplicate-tool error")
	}

	if !strings.Contains(err.Error(), `duplicate tool "example"`) {
		t.Fatalf(
			"LoadTools() error = %v, want duplicate-tool error",
			err,
		)
	}
}

func TestLoadToolsRejectsInvalidMetadata(t *testing.T) {
	dir := t.TempDir()

	writeToolMetadata(
		t,
		dir,
		"broken.toml",
		`
[[tools]]
name = "broken"

[[tools.artifacts]]
platform = "linux"
architecture = "amd64"

[tools.artifacts.source]
kind = "unsupported"
`,
	)

	_, err := LoadTools(dir)
	if err == nil {
		t.Fatal("LoadTools() error = nil, want metadata error")
	}

	if !strings.Contains(err.Error(), "broken.toml") {
		t.Fatalf(
			"LoadTools() error = %v, want filename context",
			err,
		)
	}
}

func TestLoadToolsRejectsMissingDirectory(t *testing.T) {
	dir := filepath.Join(
		t.TempDir(),
		"does-not-exist",
	)

	_, err := LoadTools(dir)
	if err == nil {
		t.Fatal("LoadTools() error = nil, want missing-directory error")
	}
}

func TestLoadToolsRejectsDirectoryWithoutTOMLDefinitions(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(dir, "README.md"),
		[]byte("documentation"),
		0600,
	); err != nil {
		t.Fatalf("write README: %v", err)
	}

	_, err := LoadTools(dir)
	if err == nil {
		t.Fatal(
			"LoadTools() error = nil, want no-definitions error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"contains no TOML definitions",
	) {
		t.Fatalf(
			"LoadTools() error = %v, want no-definitions error",
			err,
		)
	}
}

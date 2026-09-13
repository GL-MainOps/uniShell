package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/mainops/uniShell/internal/acquisition"
)

func main() {
	var (
		toolsDir = flag.String(
			"tools-dir",
			"assets/tools",
			"directory containing tool acquisition TOML definitions",
		)
		outputDir = flag.String(
			"output-dir",
			"assets/bin",
			"directory where acquired executables are installed",
		)
		sourceDir = flag.String(
			"source-dir",
			"",
			"directory containing already acquired executables",
		)
		cacheDir = flag.String(
			"cache-dir",
			"tmp/acquisition-cache",
			"directory used for acquired artifact cache",
		)
		platform = flag.String(
			"platform",
			"linux",
			"target platform",
		)
		architecture = flag.String(
			"architecture",
			"amd64",
			"target architecture",
		)
		profile = flag.String(
			"profile",
			"",
			"profile name to acquire",
		)
		cached = flag.Bool(
			"cached",
			false,
			"use already acquired executables without downloading",
		)
	)

	flag.Parse()

	profileName, err := resolveProfile(*profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := run(
		context.Background(),
		*toolsDir,
		*outputDir,
		*sourceDir,
		*cacheDir,
		acquisition.Platform(*platform),
		acquisition.Architecture(*architecture),
		profileName,
		*cached,
		http.DefaultClient,
	); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func resolveProfile(value string) (string, error) {
	profile := strings.TrimSpace(value)

	if strings.Contains(profile, ",") {
		return "", fmt.Errorf(
			"--profile accepts exactly one profile name",
		)
	}

	return profile, nil
}

func run(
	ctx context.Context,
	toolsDir string,
	outputDir string,
	sourceDir string,
	cacheDir string,
	platform acquisition.Platform,
	architecture acquisition.Architecture,
	profile string,
	cached bool,
	client *http.Client,
) error {
	tools, err := acquisition.LoadTools(toolsDir)
	if err != nil {
		return err
	}

	if profile != "" {
		tools, err = acquisition.SelectToolsForProfile(
			tools,
			profile,
		)
		if err != nil {
			return err
		}
	}

	if cached {
		if sourceDir == "" {
			return fmt.Errorf(
				"source directory is required in cached mode",
			)
		}

		return materializeCachedTools(
			tools,
			sourceDir,
			outputDir,
		)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf(
			"create output directory %q: %w",
			outputDir,
			err,
		)
	}

	stagingDir := filepath.Join(
		filepath.Dir(cacheDir),
		"acquisition-stage",
	)

	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf(
			"create staging directory %q: %w",
			stagingDir,
			err,
		)
	}

	provider := acquisition.NewHTTPProvider(client)
	cache := acquisition.NewFilesystemCache(cacheDir)
	downloader := acquisition.HTTPDownloader{
		Client: client,
	}
	acquirer := acquisition.NewAcquirer(
		downloader,
		cache,
	)
	stager := acquisition.NewFilesystemStager(stagingDir)
	validator := acquisition.NewCompositeValidator(
		acquisition.StaticELFValidator{},
		acquisition.MuslValidator{},
	)

	pipeline := acquisition.NewPipeline(
		acquirer,
		stager,
		validator,
	)

	providers := map[acquisition.SourceKind]acquisition.Provider{
		acquisition.SourceKindGitHubRelease: provider,
		acquisition.SourceKindGitHubFile:    provider,
		acquisition.SourceKindDirectURL:     provider,
	}

	toolAcquirer := acquisition.NewToolAcquirer(
		pipeline,
		providers,
	)

	for _, tool := range tools {
		fmt.Printf(
			"==> Acquiring %s for %s/%s\n",
			tool.Name,
			platform,
			architecture,
		)

		staged, err := toolAcquirer.AcquireTool(
			ctx,
			tool,
			platform,
			architecture,
		)
		if err != nil {
			return fmt.Errorf(
				"acquire tool %q: %w",
				tool.Name,
				err,
			)
		}

		if err := installBinary(
			staged.BinaryPath,
			filepath.Join(outputDir, staged.BinaryName),
		); err != nil {
			os.RemoveAll(staged.RootPath)

			return fmt.Errorf(
				"install tool %q: %w",
				tool.Name,
				err,
			)
		}

		if err := os.RemoveAll(staged.RootPath); err != nil {
			return fmt.Errorf(
				"remove staging directory for tool %q: %w",
				tool.Name,
				err,
			)
		}
	}

	return nil
}

func materializeCachedTools(
	tools []acquisition.Tool,
	sourceDir string,
	outputDir string,
) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf(
			"create output directory %q: %w",
			outputDir,
			err,
		)
	}

	for _, tool := range tools {
		for _, artifact := range tool.Artifacts {
			sourcePath := filepath.Join(
				sourceDir,
				artifact.BinaryName,
			)
			destinationPath := filepath.Join(
				outputDir,
				artifact.BinaryName,
			)

			if err := copyCachedBinary(
				sourcePath,
				destinationPath,
			); err != nil {
				return fmt.Errorf(
					"materialize cached tool %q: %w",
					tool.Name,
					err,
				)
			}

			break
		}
	}

	return nil
}

func copyCachedBinary(
	sourcePath string,
	destinationPath string,
) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf(
			"stat cached binary %q: %w",
			sourcePath,
			err,
		)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf(
			"cached binary %q is not a regular file",
			sourcePath,
		)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf(
			"open cached binary %q: %w",
			sourcePath,
			err,
		)
	}
	defer source.Close()

	if err := os.MkdirAll(
		filepath.Dir(destinationPath),
		0755,
	); err != nil {
		return fmt.Errorf(
			"create destination directory: %w",
			err,
		)
	}

	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf(
			"create destination binary %q: %w",
			destinationPath,
			err,
		)
	}

	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		_ = os.Remove(destinationPath)

		return fmt.Errorf(
			"copy cached binary: %w",
			err,
		)
	}

	if err := destination.Chmod(0755); err != nil {
		_ = destination.Close()
		_ = os.Remove(destinationPath)

		return fmt.Errorf(
			"set executable permissions: %w",
			err,
		)
	}

	if err := destination.Close(); err != nil {
		_ = os.Remove(destinationPath)

		return fmt.Errorf(
			"close destination binary: %w",
			err,
		)
	}

	return nil
}

func installBinary(sourcePath, destinationPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf(
			"stat staged binary %q: %w",
			sourcePath,
			err,
		)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf(
			"staged binary %q is not a regular file",
			sourcePath,
		)
	}

	destinationDir := filepath.Dir(destinationPath)

	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return fmt.Errorf(
			"create destination directory %q: %w",
			destinationDir,
			err,
		)
	}

	tempFile, err := os.CreateTemp(
		destinationDir,
		".unishell-tool-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary destination for %q: %w",
			destinationPath,
			err,
		)
	}

	tempPath := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if err := tempFile.Chmod(0755); err != nil {
		cleanup()

		return fmt.Errorf(
			"set executable permissions for temporary binary: %w",
			err,
		)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		cleanup()

		return fmt.Errorf(
			"open staged binary %q: %w",
			sourcePath,
			err,
		)
	}

	_, copyErr := io.Copy(tempFile, source)
	sourceCloseErr := source.Close()
	fileCloseErr := tempFile.Close()

	if copyErr != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"copy staged binary %q: %w",
			sourcePath,
			copyErr,
		)
	}

	if sourceCloseErr != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"close staged binary %q: %w",
			sourcePath,
			sourceCloseErr,
		)
	}

	if fileCloseErr != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"close temporary binary: %w",
			fileCloseErr,
		)
	}

	if err := os.Rename(tempPath, destinationPath); err != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"install binary %q: %w",
			destinationPath,
			err,
		)
	}

	return nil
}

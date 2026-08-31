package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
)

var pumlCmd = &cobra.Command{
	Use:           "puml",
	Short:         "Render PlantUML (.puml) files in docs/ to images",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		rp, err := repoPath()
		if err != nil {
			return err
		}

		inputPath := filepath.Join(rp, "docs")
		if !isDir(inputPath) {
			return fmt.Errorf("no docs directory at %s", inputPath)
		}

		pumlFiles, err := findPumlFiles(inputPath)
		if err != nil {
			return err
		}
		if len(pumlFiles) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No .puml files under %s\n", inputPath)
			return nil
		}

		outputPath := filepath.Join(rp, "target", "bartleby", "puml_images")
		if err := os.MkdirAll(outputPath, 0o755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", outputPath, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Rendering %d PlantUML file(s) to %s...\n", len(pumlFiles), outputPath)

		return runner.RunPuml(cmd.Context(), runner.PumlConfig{
			ImageName:  imageName(os.Getenv),
			InputPath:  inputPath,
			OutputPath: outputPath,
			Files:      pumlFiles,
		})
	},
}

// findPumlFiles returns every .puml file under root, as forward-slash paths
// relative to root, because the container resolves them inside a Linux mount.
// Results are sorted so runs are reproducible.
func findPumlFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".puml") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s for .puml files: %w", root, err)
	}

	sort.Strings(files)
	return files, nil
}

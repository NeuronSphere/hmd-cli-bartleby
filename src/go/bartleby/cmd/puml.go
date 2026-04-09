package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
)

var pumlCmd = &cobra.Command{
	Use:   "puml",
	Short: "Generate images from PlantUML (.puml) files",
	RunE: func(cmd *cobra.Command, args []string) error {
		rp := repoPath()
		inputPath := filepath.Join(rp, "docs")
		outputPath := filepath.Join(rp, "target", "bartleby", "puml_images")

		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("docs/ directory not found in %s", rp)
		}

		pumlFiles, err := findPumlFiles(inputPath)
		if err != nil {
			return fmt.Errorf("failed to scan for puml files: %w", err)
		}

		if len(pumlFiles) == 0 {
			fmt.Println("No .puml files found in docs/")
			return nil
		}

		if err := os.MkdirAll(outputPath, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		fmt.Printf("Found %d .puml file(s)\n", len(pumlFiles))

		return runner.RunPuml(runner.PumlConfig{
			ImageName:  imageName(),
			InputPath:  inputPath,
			OutputPath: outputPath,
			Files:      pumlFiles,
		})
	},
}

// findPumlFiles returns all .puml files under root as paths relative to root,
// using forward slashes (as the container expects a Linux path).
func findPumlFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".puml") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files, err
}

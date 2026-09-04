package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/mount"
)

// PumlConfig describes a PlantUML rendering run.
type PumlConfig struct {
	ImageName  string
	InputPath  string
	OutputPath string
	// Files are paths relative to InputPath, using forward slashes.
	Files []string
}

// Env builds the environment for the PlantUML entrypoint.
func (c PumlConfig) Env() []string {
	return []string{"PUML_FILES=" + strings.Join(c.Files, ",")}
}

// Mounts builds the bind mounts for a PlantUML run.
func (c PumlConfig) Mounts() []mount.Mount {
	return []mount.Mount{
		{Type: mount.TypeBind, Source: c.InputPath, Target: inputMount},
		{Type: mount.TypeBind, Source: c.OutputPath, Target: outputMount},
	}
}

// RunPuml renders every .puml file in the config to an image.
//
// The container is left unnamed: PlantUML runs are short and independent, so
// there is nothing to be gained from a stable name and nothing to collide with.
func RunPuml(ctx context.Context, cfg PumlConfig) error {
	if len(cfg.Files) == 0 {
		return nil
	}

	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer cli.Close()

	return run(ctx, cli, spec{
		Image:  cfg.ImageName,
		Env:    cfg.Env(),
		Cmd:    []string{"python", "entry_puml.py"},
		Mounts: cfg.Mounts(),
	})
}

package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ErrImageNotFound reports that the image was not present locally, which is not
// a failure when the goal is to replace it.
var ErrImageNotFound = errors.New("image not present locally")

// PullImage pulls an image, reporting progress as readable lines rather than the
// raw JSON stream the API returns.
func PullImage(ctx context.Context, imageName string) error {
	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer cli.Close()

	fmt.Printf("Pulling %s...\n", imageName)

	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling %s: %w", imageName, err)
	}
	defer reader.Close()

	return reportProgress(reader, os.Stdout)
}

// RemoveImage deletes a local image. A missing image returns ErrImageNotFound so
// callers can decide whether that matters.
func RemoveImage(ctx context.Context, imageName string) error {
	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer cli.Close()

	if _, err := cli.ImageRemove(ctx, imageName, image.RemoveOptions{}); err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("%s: %w", imageName, ErrImageNotFound)
		}
		return fmt.Errorf("removing %s: %w", imageName, err)
	}
	return nil
}

// progressMessage is the subset of the Docker JSON progress stream worth showing.
type progressMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"error"`
}

// reportProgress decodes a Docker JSON progress stream and prints one line per
// distinct status, skipping the per-layer byte counts that make the raw stream
// unreadable when it is not attached to a terminal.
func reportProgress(r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)
	lastStatus := map[string]string{}

	for {
		var msg progressMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading progress stream: %w", err)
		}

		if msg.Error != "" {
			return errors.New(msg.Error)
		}
		if msg.Status == "" {
			continue
		}
		if lastStatus[msg.ID] == msg.Status {
			continue
		}
		lastStatus[msg.ID] = msg.Status

		if msg.ID != "" {
			fmt.Fprintf(w, "  %s: %s\n", msg.ID, msg.Status)
		} else {
			fmt.Fprintf(w, "  %s\n", msg.Status)
		}
	}
}

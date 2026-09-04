package explain

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RunOptions drives one explanation attempt.
type RunOptions struct {
	Collect Options
	Prompt  PromptOptions

	// Requester performs the request. Required unless DryRun is set.
	Requester Requester
	// Model is only used in the status line; the requester holds the real value.
	Model string

	// Out receives the explanation, Status the one-line summary of what is being
	// sent. Keeping them apart lets the answer be piped somewhere.
	Out    io.Writer
	Status io.Writer

	// DryRun assembles and prints the payload without sending it, which is how
	// you check what would leave the machine.
	DryRun bool

	// SaveDir is where the answer is written as <builder>-explain.md. Empty means
	// the directory the logs came from.
	SaveDir string
}

// Run collects the evidence for a failed build and asks for an explanation.
//
// The answer is returned as well as written, and the saved copy sits beside the
// logs it explains.
func Run(ctx context.Context, o RunOptions) (string, error) {
	payload, err := Collect(o.Collect)
	if err != nil {
		return "", err
	}

	prompt, err := ResolvePrompt(o.Prompt)
	if err != nil {
		return "", err
	}

	rendered := payload.Render()

	if o.Status != nil {
		fmt.Fprintf(o.Status, "Reading %s", payload.LogFile)
		if payload.WarningsFile != "" {
			fmt.Fprint(o.Status, " + warnings")
		}
		if payload.LatexFile != "" {
			fmt.Fprint(o.Status, " + the LaTeX log")
		}
		if n := len(payload.Excerpts); n > 0 {
			fmt.Fprintf(o.Status, " + %d source excerpt(s)", n)
		}
		fmt.Fprintln(o.Status)

		if o.DryRun {
			fmt.Fprintf(o.Status, "Would send %.1f KiB to %s using the %s prompt.\n",
				float64(len(rendered))/1024, displayModel(o.Model), prompt.Source)
		} else {
			fmt.Fprintf(o.Status, "Sending %.1f KiB to %s using the %s prompt...\n\n",
				float64(len(rendered))/1024, displayModel(o.Model), prompt.Source)
		}
	}

	if o.DryRun {
		if o.Out != nil {
			fmt.Fprintln(o.Out, rendered)
		}
		return rendered, nil
	}

	if o.Requester == nil {
		return "", fmt.Errorf("no requester configured")
	}

	answer, err := o.Requester.Explain(ctx, prompt.System, rendered)
	if err != nil {
		return "", err
	}

	// The streaming writer may already have printed the answer; only write it
	// again when it did not.
	if o.Out != nil && !streamed(o.Requester) {
		fmt.Fprintln(o.Out, answer)
	}

	if path, err := save(o, payload, answer); err != nil {
		if o.Status != nil {
			fmt.Fprintf(o.Status, "\nwarning: could not save the explanation: %v\n", err)
		}
	} else if o.Status != nil {
		fmt.Fprintf(o.Status, "\n\nSaved to %s\n", path)
	}

	return answer, nil
}

// save writes the explanation next to the logs it explains.
func save(o RunOptions, payload *Payload, answer string) (string, error) {
	dir := o.SaveDir
	if dir == "" {
		dir = filepath.Dir(payload.LogFile)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, payload.Builder+"-explain.md")
	content := fmt.Sprintf("# What went wrong: %s (%s builder)\n\n%s\n", payload.Repo, payload.Builder, answer)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// streamed reports whether the requester already printed the answer itself.
func streamed(r Requester) bool {
	claude, ok := r.(Claude)
	return ok && claude.Stream != nil
}

func displayModel(model string) string {
	if model == "" {
		return DefaultModel
	}
	return model
}

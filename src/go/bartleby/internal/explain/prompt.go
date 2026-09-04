package explain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepoPromptPath is a per-repository prompt override, checked into the repo whose
// documentation is being built.
var RepoPromptPath = filepath.Join(".bartleby", "explain-prompt.md")

// Environment variables that override the prompt.
const (
	EnvPrompt     = "BARTLEBY_EXPLAIN_PROMPT"
	EnvPromptFile = "BARTLEBY_EXPLAIN_PROMPT_FILE"
	EnvModel      = "BARTLEBY_EXPLAIN_MODEL"
	EnvEnabled    = "BARTLEBY_EXPLAIN"
)

// DefaultSystemPrompt frames the task. It asks for a single best explanation
// rather than a list of possibilities, because this is one pass: the point is to
// turn a wall of log into the next thing to do.
const DefaultSystemPrompt = `You are helping a developer understand why a Sphinx documentation build failed.

You are given the warnings Sphinx emitted, the end of the build log, the LaTeX log
if the PDF builder ran, and the source lines the warnings refer to. You get one
pass — there is no opportunity to ask questions or read more files.

Answer in this shape, in plain prose with no preamble:

1. **What went wrong** — one or two sentences naming the actual cause, in the
   user's terms rather than Sphinx's. If several things are wrong, lead with the
   one that stopped the build.
2. **Where** — the file and line to change, when the evidence identifies one.
3. **The fix** — concretely. Show the corrected reStructuredText, configuration,
   or manifest entry where a snippet is clearer than a description.
4. **Anything else worth fixing** — remaining warnings, briefly, only if they are
   real problems rather than noise.

Rules:

- Distinguish what the log proves from what you are inferring, and say which.
- If the evidence does not identify the cause, say so plainly and name the single
  most useful thing to look at next. Do not invent a file, a line, or a
  configuration key.
- A LaTeX failure is usually a character or construct that is legal in HTML and
  not in LaTeX; check the cited source for that before blaming configuration.
- Be brief. The user is mid-task.`

// Prompt is the resolved instruction plus where it came from, so the CLI can say
// which prompt it used.
type Prompt struct {
	System string
	Source string
}

// PromptOptions controls prompt resolution.
type PromptOptions struct {
	// File is the --prompt-file flag.
	File string
	// RepoPath is where the per-repository override is looked for.
	RepoPath string
	// Env reads environment variables; nil means os.Getenv.
	Env func(string) string
}

// ResolvePrompt returns the system prompt to use, in precedence order: the
// --prompt-file flag, BARTLEBY_EXPLAIN_PROMPT_FILE, BARTLEBY_EXPLAIN_PROMPT, a
// per-repository .bartleby/explain-prompt.md, then the built-in.
func ResolvePrompt(o PromptOptions) (Prompt, error) {
	env := o.Env
	if env == nil {
		env = os.Getenv
	}

	if o.File != "" {
		text, err := readPromptFile(o.File)
		if err != nil {
			return Prompt{}, err
		}
		return Prompt{System: text, Source: o.File}, nil
	}

	if path := env(EnvPromptFile); path != "" {
		text, err := readPromptFile(path)
		if err != nil {
			return Prompt{}, fmt.Errorf("%s: %w", EnvPromptFile, err)
		}
		return Prompt{System: text, Source: fmt.Sprintf("%s (%s)", path, EnvPromptFile)}, nil
	}

	if text := strings.TrimSpace(env(EnvPrompt)); text != "" {
		return Prompt{System: text, Source: EnvPrompt}, nil
	}

	if o.RepoPath != "" {
		path := filepath.Join(o.RepoPath, RepoPromptPath)
		if text, err := readPromptFile(path); err == nil {
			return Prompt{System: text, Source: RepoPromptPath}, nil
		} else if !os.IsNotExist(err) {
			return Prompt{}, err
		}
	}

	return Prompt{System: DefaultSystemPrompt, Source: "built-in"}, nil
}

func readPromptFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return text, nil
}

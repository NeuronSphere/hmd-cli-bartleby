// Package agents carries the agent definitions bartleby ships with.
//
// Like the skills, they are embedded in the binary so that installing them
// needs no network and no checkout, and they live inside the module because
// go:embed cannot reach outside it.
package agents

import (
	"embed"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/bundle"
)

//go:embed all:*/AGENT.md
var files embed.FS

// File is the name every agent definition is bundled under.
const File = "AGENT.md"

// Set is the bundled agents.
//
// Note the asymmetry with skills: an agent is bundled as a directory containing
// AGENT.md, but installs as a single <name>.md file, because that is what an
// agent runtime reads from its agents directory.
var Set = bundle.Set{
	FS:         files,
	SourceFile: File,
	Kind:       "agent",
	Home:       "agents",
	DestName:   func(name string) string { return name + ".md" },
}

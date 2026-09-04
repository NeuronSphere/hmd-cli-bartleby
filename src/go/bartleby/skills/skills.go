// Package skills carries the agent skills bartleby ships with.
//
// They are embedded in the binary rather than fetched, so a brew-installed
// bartleby can seed a machine with no network and no Python runtime. They live
// in this directory because go:embed cannot reach outside the module; the
// legacy Python package copies them from here too.
package skills

import (
	"embed"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/bundle"
)

//go:embed all:*/SKILL.md
var files embed.FS

// File is the name every skill's instructions are stored under, which is what
// an agent looks for.
const File = "SKILL.md"

// Set is the bundled skills. A skill is a directory containing SKILL.md, both
// where it is bundled and where it installs.
var Set = bundle.Set{
	FS:         files,
	SourceFile: File,
	Kind:       "skill",
	Home:       "skills",
	DestName:   func(name string) string { return name + "/" + File },
}

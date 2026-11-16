package reqtrace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/needs"
)

// Stages are the small vocabulary every raw :status: is normalised to, so a
// consumer can ask "what is open?" across repositories that each spell their
// statuses their own way. The raw status is always kept beside the stage.
const (
	StageDraft       = "draft"
	StageProposed    = "proposed"
	StageInProgress  = "in_progress"
	StageImplemented = "implemented"
	StageDeferred    = "deferred"
	StageClosed      = "closed"
	StageUnknown     = "unknown"
)

// defaultStages maps the statuses found across NeuronSphere repositories to
// stages. A repository extends it with needs_status_stages in its manifest.
var defaultStages = map[string]string{
	"draft":       StageDraft,
	"proposed":    StageProposed,
	"open":        StageProposed,
	"defined":     StageProposed,
	"future":      StageProposed,
	"in_progress": StageInProgress,
	"partial":     StageInProgress,
	"partly":      StageInProgress,
	"amended":     StageInProgress,
	"implemented": StageImplemented,
	"completed":   StageImplemented,
	"done":        StageImplemented,
	"fixed":       StageImplemented,
	"deferred":    StageDeferred,
	"withdrawn":   StageClosed,
	"superseded":  StageClosed,
	"rejected":    StageClosed,
}

var knownStages = map[string]bool{
	StageDraft: true, StageProposed: true, StageInProgress: true, StageImplemented: true,
	StageDeferred: true, StageClosed: true,
}

// repoConfig is what reqtrace reads from a repository's manifest beyond its
// name: the needs vocabulary and the status table, from any bartleby root's
// config.
type repoConfig struct {
	Name string `json:"name,omitempty"`
	// Types replaces the default directive vocabulary when non-empty, as
	// needs_types does in Sphinx.
	Types []string `json:"types,omitempty"`
	// Stages extends the default status table.
	Stages map[string]string `json:"stages,omitempty"`
	// Strict makes an unknown status a traceability failure.
	Strict bool `json:"strict,omitempty"`
}

// readRepoConfig reads meta-data/manifest.json. A repository without one gets
// the zero config and ok=false; a manifest that is present but unreadable is
// an error.
func readRepoConfig(repoRoot string) (cfg repoConfig, ok bool, err error) {
	path := filepath.Join(repoRoot, "meta-data", "manifest.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return repoConfig{}, false, nil
	}
	if err != nil {
		return repoConfig{}, false, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest struct {
		Name     string `json:"name"`
		Bartleby struct {
			Roots map[string]struct {
				Config struct {
					NeedsTypes        []json.RawMessage `json:"needs_types"`
					NeedsStatusStages map[string]string `json:"needs_status_stages"`
					NeedsStatusStrict bool              `json:"needs_status_strict"`
				} `json:"config"`
			} `json:"roots"`
		} `json:"bartleby"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return repoConfig{}, true, fmt.Errorf("parsing %s: %w", path, err)
	}

	cfg.Name = strings.TrimSpace(manifest.Name)
	seenType := map[string]bool{}
	for _, root := range manifest.Bartleby.Roots {
		c := root.Config
		// needs_types entries are objects with a "directive" in Sphinx;
		// accept a bare string too.
		for _, raw := range c.NeedsTypes {
			var entry struct {
				Directive string `json:"directive"`
			}
			var name string
			if json.Unmarshal(raw, &entry) == nil && entry.Directive != "" {
				name = entry.Directive
			} else {
				_ = json.Unmarshal(raw, &name)
			}
			if name = strings.ToLower(strings.TrimSpace(name)); name != "" && !seenType[name] {
				seenType[name] = true
				cfg.Types = append(cfg.Types, name)
			}
		}
		for status, stage := range c.NeedsStatusStages {
			if cfg.Stages == nil {
				cfg.Stages = map[string]string{}
			}
			cfg.Stages[normaliseStatus(status)] = stage
		}
		cfg.Strict = cfg.Strict || c.NeedsStatusStrict
	}
	return cfg, true, nil
}

// types returns the directive vocabulary for this repository.
func (c repoConfig) types() map[string]bool {
	if len(c.Types) == 0 {
		return needs.DefaultTypes()
	}
	out := make(map[string]bool, len(c.Types))
	for _, t := range c.Types {
		out[t] = true
	}
	return out
}

// stageOf returns the stage for a raw status, and whether the status was
// recognised. An empty status is unknown but not unrecognised: nothing was
// claimed, so there is nothing to correct.
func (c repoConfig) stageOf(status string) (stage string, recognised bool) {
	key := normaliseStatus(status)
	if key == "" {
		return StageUnknown, true
	}
	if s, ok := c.Stages[key]; ok && knownStages[s] {
		return s, true
	}
	if s, ok := defaultStages[key]; ok {
		return s, true
	}
	return StageUnknown, false
}

// normaliseStatus lower-cases a status and treats runs of spaces and hyphens
// as one underscore, so "In Progress" and "in-progress" match in_progress.
func normaliseStatus(s string) string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '-' || r == '_'
	})
	return strings.Join(fields, "_")
}

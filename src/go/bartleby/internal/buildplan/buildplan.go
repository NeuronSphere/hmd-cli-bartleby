// Package buildplan turns a manifest plus flags and environment into the exact
// list of (document, builder) pairs to run, with each pair's resolved config.
//
// It is deliberately free of Docker and filesystem work so the resolution rules —
// which are the fiddliest part of the CLI — can be tested directly.
package buildplan

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

// AllShells is the sentinel meaning "every builder each root declares".
const AllShells = "all"

// Build is one container run: one root document rendered by one builder.
type Build struct {
	// Name is the manifest key for the root (the "name" in the transform context).
	Name string
	// Shell is the Sphinx builder / make target.
	Shell string
	// RootDoc is the document to build, defaulted to "index" when unset.
	RootDoc string
	// Config is the merged builder config handed to the container.
	Config map[string]any
}

// Documents returns the roots to build, filtered by rootDoc ("all" or a
// comma-separated list of manifest root names). Unknown names produce a warning
// on warnTo; a filter that matches nothing is an error listing what is available.
func Documents(m *manifest.Manifest, rootDoc string, warnTo *os.File) (map[string]manifest.Root, error) {
	roots := m.Bartleby.Roots
	if len(roots) == 0 {
		return manifest.DefaultRoots(), nil
	}

	if rootDoc == "" || rootDoc == AllShells {
		return roots, nil
	}

	docs := make(map[string]manifest.Root)
	for _, name := range splitList(rootDoc) {
		if root, ok := roots[name]; ok {
			docs[name] = root
			continue
		}
		if warnTo != nil {
			fmt.Fprintf(warnTo, "warning: root document %q not found in manifest (available: %s)\n",
				name, strings.Join(rootNames(roots), ", "))
		}
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("none of the requested root documents (%s) are in %s (available: %s)",
			rootDoc, "meta-data/manifest.json", strings.Join(rootNames(roots), ", "))
	}
	return docs, nil
}

// Builds expands docs into one Build per (root, builder), keeping only builders
// that match shellFilter.
//
// shellFilter is "all" or a comma-separated list of builder names — the form the
// documentation has always described but neither implementation supported. A
// filter that matches no declared builder is an error rather than a silent
// no-op; the old behaviour printed "No builds to run." and exited 0, which reads
// as success.
//
// Builds are returned in a deterministic order (root name, then builder name) so
// output and tests do not depend on Go's map iteration order.
func Builds(docs map[string]manifest.Root, shellFilter string, cfg manifest.Config) ([]Build, error) {
	wanted, all := parseShellFilter(shellFilter)

	var builds []Build
	declared := make(map[string]bool)

	for _, rootName := range rootNames(docs) {
		root := docs[rootName]
		for _, builder := range root.Builders {
			if builder.Shell == "" {
				continue
			}
			declared[builder.Shell] = true
			if !all && !wanted[builder.Shell] {
				continue
			}
			builds = append(builds, Build{
				Name:    rootName,
				Shell:   builder.Shell,
				RootDoc: root.Doc(),
				Config:  BuilderConfig(root, builder, cfg),
			})
		}
	}

	if len(builds) == 0 {
		available := make([]string, 0, len(declared))
		for shell := range declared {
			available = append(available, shell)
		}
		sort.Strings(available)
		if len(available) == 0 {
			return nil, fmt.Errorf("no builders are declared for the selected root document(s)")
		}
		return nil, fmt.Errorf("no builder named %q is declared for the selected root document(s) (available: %s)",
			shellFilter, strings.Join(available, ", "))
	}

	// Warn about a requested builder that no root declares, so a typo in a list
	// of several does not pass unnoticed just because the others matched.
	if !all {
		for shell := range wanted {
			if !declared[shell] {
				fmt.Fprintf(os.Stderr, "warning: no root declares the %q builder — skipping it\n", shell)
			}
		}
	}

	sort.Slice(builds, func(i, j int) bool {
		if builds[i].Name != builds[j].Name {
			return builds[i].Name < builds[j].Name
		}
		return builds[i].Shell < builds[j].Shell
	})

	return builds, nil
}

// BuilderConfig merges the four config layers for one builder, lowest priority
// first:
//
//  1. the root's own "config" object
//  2. bartleby.config.builders.<shell> from the manifest, or, if the manifest
//     declares none, the HMD_BARTLEBY_<SHELL>_CONFIG environment variable
//     parsed as JSON
//  3. the builder's inline config, when the manifest used the
//     {"shell": ..., "config": ...} form
//  4. individual HMD_BARTLEBY__<SHELL>__<KEY> environment variables
//
// This matches the Python CLI's precedence, with the difference that the Python
// implementation could not actually reach layer 3 without raising.
func BuilderConfig(root manifest.Root, builder manifest.Builder, cfg manifest.Config) map[string]any {
	merged := make(map[string]any)

	for k, v := range root.Config {
		merged[k] = v
	}

	manifestBuilder := cfg.BuilderConfig(builder.Shell)
	if len(manifestBuilder) == 0 {
		manifestBuilder = envJSONConfig(builder.Shell)
	}
	for k, v := range manifestBuilder {
		merged[k] = v
	}

	for k, v := range builder.Config {
		merged[k] = v
	}

	for k, v := range EnvScalarConfig(builder.Shell, os.Environ()) {
		merged[k] = v
	}

	return merged
}

// envJSONConfig reads HMD_BARTLEBY_<SHELL>_CONFIG as a JSON object. Malformed
// JSON is ignored with a warning rather than failing the build, matching the
// permissiveness of the rest of the config path.
func envJSONConfig(shell string) map[string]any {
	key := fmt.Sprintf("HMD_BARTLEBY_%s_CONFIG", strings.ToUpper(shell))
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring %s: not a JSON object (%v)\n", key, err)
		return nil
	}
	return parsed
}

// EnvScalarConfig collects HMD_BARTLEBY__<SHELL>__<KEY> variables from environ
// (in os.Environ() form) into lower-cased config keys.
func EnvScalarConfig(shell string, environ []string) map[string]any {
	prefix := fmt.Sprintf("HMD_BARTLEBY__%s__", strings.ToUpper(shell))
	config := make(map[string]any)

	for _, kv := range environ {
		key, value, found := strings.Cut(kv, "=")
		if !found || !strings.HasPrefix(key, prefix) {
			continue
		}
		configKey := strings.ToLower(strings.TrimPrefix(key, prefix))
		if configKey == "" {
			continue
		}
		config[configKey] = value
	}

	if len(config) == 0 {
		return nil
	}
	return config
}

// parseShellFilter turns a --shell value into a set of builder names. It reports
// all=true for "all", an empty value, or a list containing "all".
func parseShellFilter(filter string) (wanted map[string]bool, all bool) {
	names := splitList(filter)
	if len(names) == 0 {
		return nil, true
	}

	wanted = make(map[string]bool, len(names))
	for _, name := range names {
		if name == AllShells {
			return nil, true
		}
		wanted[name] = true
	}
	return wanted, false
}

// splitList splits a comma-separated flag value, dropping blanks and whitespace.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// rootNames returns the sorted keys of a roots map.
func rootNames(roots map[string]manifest.Root) []string {
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Root struct {
	RootDoc  string                 `json:"root_doc"`
	Builders []string               `json:"builders"`
	Config   map[string]interface{} `json:"config"`
}

type Source struct {
	ArtifactPath string `json:"artifact_path"`
	DocsRoot     string `json:"docs_root"`
	Title        string `json:"title"`
}

type BartlebySection struct {
	Roots   map[string]Root   `json:"roots"`
	Sources map[string]Source `json:"sources"`
	Config  map[string]interface{} `json:"config"`
}

type Manifest struct {
	Name     string          `json:"name"`
	Bartleby BartlebySection `json:"bartleby"`
}

// Read loads meta-data/manifest.json from repoPath. Returns an empty manifest if not found.
func Read(repoPath string) (*Manifest, error) {
	path := filepath.Join(repoPath, "meta-data", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &Manifest{}, nil
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ReadVersion reads meta-data/VERSION from repoPath. Returns "stable" if not found.
func ReadVersion(repoPath string) string {
	path := filepath.Join(repoPath, "meta-data", "VERSION")
	data, err := os.ReadFile(path)
	if err != nil {
		return "stable"
	}
	return strings.TrimSpace(string(data))
}

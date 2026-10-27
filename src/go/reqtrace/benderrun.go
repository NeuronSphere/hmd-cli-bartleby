package reqtrace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Bender archives every run (hmd-tf-bender run_identity.py, NERD003 SPEC004)
// as a folder bender/runs/<stamp>_<repo>_<run_id>/ holding
// <stamp>_<repo>[_<label>]_<kind> files: output.xml, log.html, report.html and
// run.json. <stamp> is the run's UTC start time with no colons, so folder and
// file names sort in run order.
const (
	runOutputSuffix = "_output.xml"
	runRecordSuffix = "_run.json"
	runLabelTag     = "hmd_run_label:"
	runIDTag        = "hmd_run_id:"
	defaultLabel    = "default"
)

// benderRunRecord is the part of a Bender run.json that verification uses.
type benderRunRecord struct {
	RunID       string `json:"hmd_run_id"`
	Started     string `json:"hmd_run_started"`
	RepoName    string `json:"hmd_repo_name"`
	RepoVersion string `json:"hmd_repo_version"`
	Label       string `json:"hmd_run_label"`
}

// isRunFolder reports whether path is a directory, which -results treats as a
// Bender run folder.
func isRunFolder(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// singleFile returns the one file in dir whose name ends with suffix: none is
// "" with no error, more than one is an error.
func singleFile(dir, suffix string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*"+suffix))
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	}
	return "", fmt.Errorf("%s: %d files end in %s, want one", dir, len(matches), suffix)
}

// fileLabel is the label segment of <stamp>_<repo>_<label>_output.xml, or ""
// when the name has none. Repository names use hyphens, never underscores.
func fileLabel(outputPath string) string {
	stem := strings.TrimSuffix(filepath.Base(outputPath), runOutputSuffix)
	parts := strings.SplitN(stem, "_", 3)
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}

// folderStamp is the <stamp> that starts a run folder's name, or "".
func folderStamp(dir string) string {
	stamp, _, found := strings.Cut(filepath.Base(dir), "_")
	if !found {
		return ""
	}
	return stamp
}

// tagValue is the value of the first "prefix<value>" tag on any result.
func tagValue(results []TestResult, prefix string) string {
	for _, r := range results {
		for _, tag := range r.Tags {
			if strings.HasPrefix(tag, prefix) {
				return strings.TrimPrefix(tag, prefix)
			}
		}
	}
	return ""
}

// loadRunFolder reads one Bender run folder. A label or version given in the
// spec wins; otherwise the label comes from run.json, then the hmd_run_label:
// tag, then the file name, and the version from run.json's repo version. The
// run's time is its start time from run.json when there is one, rather than
// the time Robot wrote output.xml.
func loadRunFolder(spec ResultSpec) (RunResults, error) {
	output, err := singleFile(spec.Path, runOutputSuffix)
	if err != nil {
		return RunResults{}, err
	}
	if output == "" {
		return RunResults{}, fmt.Errorf("%s: no *%s; not a Bender run folder", spec.Path, runOutputSuffix)
	}
	run, err := loadResultFile(ResultSpec{Implementation: spec.Implementation, Version: spec.Version, Path: output})
	if err != nil {
		return RunResults{}, err
	}

	var record benderRunRecord
	recordPath, err := singleFile(spec.Path, runRecordSuffix)
	if err != nil {
		return RunResults{}, err
	}
	if recordPath != "" {
		data, err := os.ReadFile(recordPath)
		if err != nil {
			return RunResults{}, err
		}
		if err := json.Unmarshal(data, &record); err != nil {
			return RunResults{}, fmt.Errorf("%s: %w", recordPath, err)
		}
	}

	if spec.Implementation == "" || spec.Implementation == defaultLabel {
		run.Implementation = firstNonEmpty(record.Label, tagValue(run.Results, runLabelTag), fileLabel(output), defaultLabel)
	}
	if spec.Version == "" {
		run.Version = record.RepoVersion
	}
	run.Generated = firstNonEmpty(record.Started, run.Generated)
	run.RunID = firstNonEmpty(record.RunID, tagValue(run.Results, runIDTag))
	run.Stamp = folderStamp(spec.Path)
	return run, nil
}

// LoadRunsDir reads every Bender run folder directly under dir (such as
// bender/runs). Unless all is set it keeps, per label, only the latest run by
// its folder's start stamp. With all, each run stays its own implementation
// column, named "<label> <stamp>".
func LoadRunsDir(dir string, all bool) ([]RunResults, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var runs []RunResults
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		output, err := singleFile(path, runOutputSuffix)
		if err != nil {
			return nil, err
		}
		if output == "" {
			continue
		}
		run, err := loadRunFolder(ResultSpec{Implementation: defaultLabel, Path: path})
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("%s: no Bender run folders (<stamp>_<repo>_<run_id>/ with a *%s)", dir, runOutputSuffix)
	}
	// Folder names start with the stamp, so name order is run order.
	sort.SliceStable(runs, func(i, j int) bool { return runOrderKey(runs[i]) < runOrderKey(runs[j]) })

	if all {
		for i := range runs {
			runs[i].Implementation += " " + firstNonEmpty(runs[i].Stamp, runs[i].Generated)
		}
		return runs, nil
	}
	latest := map[string]int{}
	var order []string
	for i, run := range runs {
		if _, seen := latest[run.Implementation]; !seen {
			order = append(order, run.Implementation)
		}
		latest[run.Implementation] = i
	}
	kept := make([]RunResults, 0, len(order))
	for _, label := range order {
		kept = append(kept, runs[latest[label]])
	}
	return kept, nil
}

func runOrderKey(r RunResults) string {
	return firstNonEmpty(r.Stamp, r.Generated) + "\x00" + r.Source
}

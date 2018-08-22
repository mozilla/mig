package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

const defaultReleasesDir string = "releases"
const releaseJSONFileName string = "releases.json"

// ReleaseJSON contains information about releases of various MIG components,
// such as the agent, the agent configuration generator tool, and so on.
type ReleaseJSON struct {
	Agent           Component `json:"agent"`
	AgentConfigTool Component `json:"agentConfig"`
}

// Component describes the latest and all past releases of MIG components,
// such as the agent and its configuration generator tool, and past releases.
type Component struct {
	LatestReleaseTag string    `json:"latest"`
	ReleaseHistory   []Release `json:"releases"`
}

// Release contains information about a release of any component.
type Release struct {
	Tag    string `json:"tag"`
	Binary string `json:"binary"`
	Date   string `json:"date"`
	Notes  string `json:"notes"`
	Sha256 string `json:"sha256"`
}

// formatDate translates a date into a "year/month/day" format, using
// 1-based indexes for days and months to be human readable.
// For example, August 16, 2018 is formatted "2018/8/16".
func formatDate(date time.Time) string {
	year, month, day := date.Date()
	return fmt.Sprintf("%d/%d/%d", year, int(month), day)
}

// setLatestRelease updates a component with a new release, setting the
// latest release tag and adding to its release history.
func setLatestRelease(comp *Component, binaryName, tag, notes, sha256 string) {
	comp.LatestReleaseTag = tag

	comp.ReleaseHistory = append(comp.ReleaseHistory, Release{
		Tag:    tag,
		Binary: binaryName,
		Date:   formatDate(time.Now()),
		Notes:  notes,
		Sha256: sha256,
	})
}

// knownComponents returns a list of names of components understood by this
// tool, for which new releases can be documented.
func knownComponents() []string {
	return []string{
		"agent",
		"agentConfig",
	}
}

// component retrieves a pointer to a known component identified by its name,
// which must match one of the "knownComponents", so that a new release can be
// appended to it.
func component(releases *ReleaseJSON, compName string) (*Component, error) {
	switch compName {
	case "agent":
		return &releases.Agent, nil
	case "agentConfig":
		return &releases.AgentConfigTool, nil
	default:
		return nil, fmt.Errorf("unknown component \"%s\"", compName)
	}
}

// isUniqueTag determines whether a particular tag already appears in a
// component's release history or not.
func isUniqueTag(comp Component, tag string) bool {
	for _, release := range comp.ReleaseHistory {
		if release.Tag == tag {
			return false
		}
	}
	return true
}

// usage prints a usage string giving a more detailed explanation about
// how to use this tool.
func usage() {
	fmt.Fprintf(os.Stderr, "This program is used to update a JSON file describing releases of MIG components.\n\n")
	fmt.Fprintf(os.Stderr, "Note that the -component, -binary and -tag arguments are required.\n\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage

	releaseJSONPath := flag.String(
		"releases",
		path.Join(defaultReleasesDir, releaseJSONFileName),
		fmt.Sprintf("Path to the %s file to update", releaseJSONFileName))
	componentName := flag.String(
		"component",
		"",
		fmt.Sprintf(
			"The name of the component to update. Must be one of: %s",
			strings.Join(knownComponents(), ", ")))
	binary := flag.String(
		"binary",
		"",
		"The name of the binary file")
	tag := flag.String(
		"tag",
		"",
		"A unique tag string identifying the release")
	notes := flag.String(
		"notes",
		"",
		"Notes describing the release")
	sha256 := flag.String(
		"sha256",
		"",
		"Hex encoding of the SHA256 hash of the released binary")

	flag.Parse()

	releaseFile, err := os.Open(*releaseJSONPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open releases JSON file \"%s\"\n", *releaseJSONPath)
		os.Exit(1)
	}
	defer releaseFile.Close()

	var releaseJSON ReleaseJSON
	decoder := json.NewDecoder(releaseFile)
	decodeErr := decoder.Decode(&releaseJSON)
	if decodeErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode release JSON. Error: %s\n", decodeErr.Error())
		os.Exit(1)
	}

	component, err := component(&releaseJSON, *componentName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid component specified. Error: %s\n", err.Error())
		os.Exit(1)
	}

	if *binary == "" {
		fmt.Fprintf(os.Stderr, "No binary name supplied.\n")
	}

	if *tag == "" {
		fmt.Fprintf(os.Stderr, "No tag identifier supplied.\n")
		os.Exit(1)
	}

	setLatestRelease(component, *binary, *tag, *notes, *sha256)

	prefix := ""
	indent := "    "
	encoded, _ := json.MarshalIndent(releaseJSON, prefix, indent)
	fmt.Printf("%s\n", string(encoded))
}

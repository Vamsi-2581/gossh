package versioncheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	releaseAPI  = "https://api.github.com/repos/ankushT369/gossh/releases/latest"
	releasesURL = "https://github.com/ankushT369/gossh/releases/latest"
)

type release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// Check returns the latest version and release URL if an update is available.
func Check(currentVersion string) (string, string, error) {
	client := &http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest(http.MethodGet, releaseAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var latest release
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return "", "", err
	}

	// Prefer the tag, but support the existing "tunnel" tag
	// by falling back to the release name, currently "v1.6.0".
	latestVersion := normalize(latest.TagName)
	if latestVersion == "" {
		latestVersion = normalize(latest.Name)
	}
	if latestVersion == "" {
		return "", "", fmt.Errorf("latest release has no valid semantic version")
	}

	current := normalize(currentVersion)
	if current == "" {
		return "", "", fmt.Errorf("invalid current version %q", currentVersion)
	}

	if latest.HTMLURL == "" {
		latest.HTMLURL = releasesURL
	}

	if semver.Compare(latestVersion, current) > 0 {
		return latestVersion, latest.HTMLURL, nil
	}

	return "", "", nil
}

func normalize(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")

	version = "v" + version
	if !semver.IsValid(version) {
		return ""
	}

	return version
}

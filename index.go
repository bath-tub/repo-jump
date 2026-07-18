package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dataDir returns the per-user data directory for repo-jump, honoring
// XDG_DATA_HOME and falling back to ~/.local/share.
func dataDir() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "repo-jump")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".repo-jump" // last-resort relative path
	}
	return filepath.Join(home, ".local", "share", "repo-jump")
}

func indexPath() string { return filepath.Join(dataDir(), "repos.txt") }

// LoadIndex reads the cached repo-name list. A missing cache is reported with a
// clear, actionable error pointing at --refresh.
func LoadIndex() ([]string, error) {
	b, err := os.ReadFile(indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no repo index yet — run `repo-jump --refresh` to build it")
		}
		return nil, err
	}
	var names []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			names = append(names, line)
		}
	}
	return names, sc.Err()
}

// RefreshIndex pulls the repo list for org via the gh CLI and writes the cache.
// It returns the number of repos written.
func RefreshIndex(org string) (int, error) {
	cmd := exec.Command("gh", "repo", "list", org,
		"--limit", "4000", "--json", "name", "--jq", ".[].name")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("gh repo list %s failed: %v: %s", org, err, strings.TrimSpace(errb.String()))
	}

	var names []string
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			names = append(names, line)
		}
	}
	if len(names) == 0 {
		return 0, fmt.Errorf("gh returned no repos for org %q — is it correct and are you authed?", org)
	}

	if err := os.MkdirAll(dataDir(), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(indexPath(), []byte(strings.Join(names, "\n")+"\n"), 0o644); err != nil {
		return 0, err
	}
	return len(names), nil
}

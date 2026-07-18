package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// configDir returns the per-user config directory, honoring XDG_CONFIG_HOME and
// falling back to ~/.config.
func configDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "repo-jump")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".repo-jump"
	}
	return filepath.Join(home, ".config", "repo-jump")
}

func orgConfigPath() string { return filepath.Join(configDir(), "org") }

// readSavedOrg returns the org persisted by a previous --refresh, or "".
func readSavedOrg() string {
	b, err := os.ReadFile(orgConfigPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// saveOrg persists org so later plain `repo-jump` runs target the same owner
// the index was built from (keeping the opened URLs consistent).
func saveOrg(org string) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(orgConfigPath(), []byte(org+"\n"), 0o644)
}

// ghCurrentUser returns the login of the gh-authenticated account, or "".
func ghCurrentUser() string {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

// resolveOrg determines which GitHub org/owner to target, in priority order:
// explicit --org flag > REPO_JUMP_ORG env > saved config > gh-authenticated
// account. There is no hardcoded default, so the tool carries no org identity
// of its own.
func resolveOrg(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if env := os.Getenv("REPO_JUMP_ORG"); env != "" {
		return env, nil
	}
	if saved := readSavedOrg(); saved != "" {
		return saved, nil
	}
	if u := ghCurrentUser(); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("no org set — pass --org <name>, set REPO_JUMP_ORG, or run `gh auth login`")
}

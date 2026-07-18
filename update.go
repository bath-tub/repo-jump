package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func srcConfigPath() string { return filepath.Join(configDir(), "src") }

// readSrcDir returns where rj's source repo lives, recorded by install.sh (or
// REPO_JUMP_SRC), so `rj update` can rebuild from it.
func readSrcDir() string {
	if v := os.Getenv("REPO_JUMP_SRC"); v != "" {
		return v
	}
	b, err := os.ReadFile(srcConfigPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// runUpdate pulls the latest source, rebuilds, and replaces the running binary.
func runUpdate() {
	src := readSrcDir()
	if src == "" {
		fatal(fmt.Errorf("don't know where rj's source is — re-run install.sh, or set REPO_JUMP_SRC=/path/to/repo-jump"))
	}
	if _, err := os.Stat(filepath.Join(src, "go.mod")); err != nil {
		fatal(fmt.Errorf("no source repo at %s — re-run install.sh, or set REPO_JUMP_SRC", src))
	}

	dest, err := os.Executable()
	if err != nil {
		fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(dest); err == nil {
		dest = resolved
	}

	fmt.Println("pulling latest…")
	if err := streamCmd(src, "git", "pull", "--ff-only"); err != nil {
		fatal(fmt.Errorf("git pull failed: %w", err))
	}
	fmt.Println("building…")
	if err := streamCmd(src, "go", "build", "-o", "rj", "."); err != nil {
		fatal(fmt.Errorf("build failed: %w", err))
	}
	if err := replaceBinary(filepath.Join(src, "rj"), dest); err != nil {
		fatal(fmt.Errorf("install failed: %w", err))
	}

	sha := gitSHA(src)
	if sha != "" {
		fmt.Printf("updated %s to %s\n", dest, sha)
	} else {
		fmt.Printf("updated %s\n", dest)
	}
}

func streamCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitSHA(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// replaceBinary copies src over dst via a temp file + rename, which works even
// when dst is the currently-running executable (the OS keeps the open inode;
// rename just swaps the directory entry).
func replaceBinary(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

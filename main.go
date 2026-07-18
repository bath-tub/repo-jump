// Command repo-jump is a fast terminal fuzzy-finder that opens a GitHub repo in
// your browser. Type part of a repo name, press enter, and it opens
// https://github.com/<org>/<name>. Ranking blends fuzzy match quality with a
// self-learning frecency signal, so the repos you actually use float to the top.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func itoa(n int) string { return strconv.Itoa(n) }

func main() {
	org := flag.String("org", "", "GitHub org/owner to jump within (default: saved org, else your gh account)")
	refresh := flag.Bool("refresh", false, "rebuild the repo index via `gh repo list` and exit")
	alpha := flag.Float64("alpha", envFloat("REPO_JUMP_ALPHA", 2.0), "weight applied to the frecency signal")
	flag.Parse()

	targetOrg, err := resolveOrg(*org)
	if err != nil {
		fatal(err)
	}

	if *refresh {
		n, err := RefreshIndex(targetOrg)
		if err != nil {
			fatal(err)
		}
		if err := saveOrg(targetOrg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save org: %v\n", err)
		}
		fmt.Printf("indexed %d repos for %s\n", n, targetOrg)
		return
	}

	repos, err := LoadIndex()
	if err != nil {
		fatal(err)
	}
	fre, err := LoadFrecency()
	if err != nil {
		fatal(err)
	}

	m := newModel(repos, fre, time.Now().Unix(), *alpha)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fatal(err)
	}

	sel := final.(model).Selected
	if sel == "" {
		return // user quit without choosing
	}
	if err := fre.Bump(sel, time.Now().Unix()); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record usage: %v\n", err)
	}
	url := fmt.Sprintf("https://github.com/%s/%s", targetOrg, sel)
	if err := openBrowser(url); err != nil {
		fatal(fmt.Errorf("could not open %s: %w", url, err))
	}
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, append(args, url)...).Start()
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "repo-jump:", err)
	os.Exit(1)
}

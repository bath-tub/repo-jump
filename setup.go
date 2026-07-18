package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type setupStep int

const (
	stepChecking setupStep = iota
	stepAuthPrompt
	stepOrg
	stepRefreshing
	stepKeybind
	stepDone
	stepError
)

type ghCheckedMsg struct {
	haveGH bool
	authed bool
	user   string
}
type authDoneMsg struct{}
type refreshDoneMsg struct {
	n   int
	err error
}
type keybindMsg struct{}

type setupModel struct {
	step   setupStep
	spin   spinner.Model
	org    textinput.Model
	user   string
	count  int
	errMsg string
	zsh    bool
}

func newSetupModel() setupModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Prompt = "org> "
	return setupModel{
		step: stepChecking,
		spin: sp,
		org:  ti,
		zsh:  strings.Contains(os.Getenv("SHELL"), "zsh"),
	}
}

// checkGHCmd verifies the gh CLI is installed and authenticated.
func checkGHCmd() tea.Msg {
	if _, err := exec.LookPath("gh"); err != nil {
		return ghCheckedMsg{haveGH: false}
	}
	authed := exec.Command("gh", "auth", "status").Run() == nil
	user := ""
	if authed {
		user = ghCurrentUser()
	}
	return ghCheckedMsg{haveGH: true, authed: authed, user: user}
}

func refreshCmd(org string) tea.Cmd {
	return func() tea.Msg {
		n, err := RefreshIndex(org)
		return refreshDoneMsg{n: n, err: err}
	}
}

// addKeybindCmd appends the Ctrl-G launcher binding to ~/.zshrc if not present.
func addKeybindCmd() tea.Msg {
	home, err := os.UserHomeDir()
	if err != nil {
		return keybindMsg{}
	}
	rc := filepath.Join(home, ".zshrc")
	line := `bindkey -s '^g' 'rj\n'`
	if b, err := os.ReadFile(rc); err == nil && strings.Contains(string(b), line) {
		return keybindMsg{}
	}
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return keybindMsg{}
	}
	defer f.Close()
	_, _ = f.WriteString("\n# repo-jump: Ctrl-G to jump to a repo\n" + line + "\n")
	return keybindMsg{}
}

func (m setupModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, checkGHCmd)
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ghCheckedMsg:
		switch {
		case !msg.haveGH:
			m.step = stepError
			m.errMsg = "GitHub CLI (gh) not found — install it: https://cli.github.com"
		case !msg.authed:
			m.step = stepAuthPrompt
		default:
			m.user = msg.user
			m.org.SetValue(msg.user)
			m.org.Focus()
			m.step = stepOrg
			return m, textinput.Blink
		}
		return m, nil

	case authDoneMsg:
		m.step = stepChecking // re-check after interactive login
		return m, checkGHCmd

	case refreshDoneMsg:
		if msg.err != nil {
			m.step = stepError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.count = msg.n
		_ = saveOrg(m.org.Value())
		if m.zsh {
			m.step = stepKeybind
		} else {
			m.step = stepDone
		}
		return m, nil

	case keybindMsg:
		m.step = stepDone
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		switch m.step {
		case stepAuthPrompt:
			switch msg.String() {
			case "enter":
				return m, tea.ExecProcess(exec.Command("gh", "auth", "login"),
					func(error) tea.Msg { return authDoneMsg{} })
			case "esc", "q":
				return m, tea.Quit
			}
		case stepOrg:
			switch msg.Type {
			case tea.KeyEnter:
				org := strings.TrimSpace(m.org.Value())
				if org == "" {
					org = m.user
					m.org.SetValue(org)
				}
				if org == "" {
					return m, nil
				}
				m.step = stepRefreshing
				return m, tea.Batch(m.spin.Tick, refreshCmd(org))
			case tea.KeyEsc:
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.org, cmd = m.org.Update(msg)
			return m, cmd
		case stepKeybind:
			switch msg.String() {
			case "y", "Y", "enter":
				return m, addKeybindCmd
			case "n", "N":
				m.step = stepDone
			}
			return m, nil
		case stepDone, stepError:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m setupModel) View() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, titleStyle.Render("repo-jump setup")+"\n")

	switch m.step {
	case stepChecking:
		fmt.Fprintf(b, "%s checking GitHub CLI…\n", m.spin.View())
	case stepAuthPrompt:
		fmt.Fprintln(b, "gh is not authenticated.")
		fmt.Fprintln(b, hintStyle.Render("press enter to run `gh auth login` · q to quit"))
	case stepOrg:
		fmt.Fprintf(b, "%s authenticated as %s\n\n", okStyle.Render("✓"), m.user)
		fmt.Fprintln(b, "Which GitHub org/owner do you want to jump within?")
		fmt.Fprintln(b, m.org.View())
		fmt.Fprintln(b, hintStyle.Render("enter to accept · blank = your own account"))
	case stepRefreshing:
		fmt.Fprintf(b, "%s indexing repos for %s…\n", m.spin.View(), m.org.Value())
	case stepKeybind:
		fmt.Fprintf(b, "%s indexed %d repos for %s\n\n", okStyle.Render("✓"), m.count, m.org.Value())
		fmt.Fprintln(b, "Add a Ctrl-G zsh keybinding to launch repo-jump? [Y/n]")
	case stepDone:
		fmt.Fprintf(b, "%s indexed %d repos for %s\n", okStyle.Render("✓"), m.count, m.org.Value())
		fmt.Fprintln(b, okStyle.Render("✓")+" setup complete\n")
		fmt.Fprintln(b, "Run "+titleStyle.Render("rj")+" to jump · refresh with "+titleStyle.Render("rj --refresh")+".")
		if m.zsh {
			fmt.Fprintln(b, hintStyle.Render("reload your shell for Ctrl-G: source ~/.zshrc"))
		}
		fmt.Fprintln(b, hintStyle.Render("\npress any key to exit"))
	case stepError:
		fmt.Fprintln(b, errStyle.Render("✗ "+m.errMsg))
		fmt.Fprintln(b, hintStyle.Render("\npress any key to exit"))
	}
	return b.String()
}

func runSetup() {
	if _, err := tea.NewProgram(newSetupModel()).Run(); err != nil {
		fatal(err)
	}
}

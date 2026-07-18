package main

import (
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxRows = 15

var (
	matchStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))  // green
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))             // cyan pointer
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // help text
	freStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Faint(true) // ×N visit count
)

type scored struct {
	name      string
	positions map[int]bool
	final     float64
	fre       float64
	count     int // times opened, shown as ×N
}

type model struct {
	ti      textinput.Model
	repos   []string
	fre     *Frecency
	now     int64
	alpha   float64
	results []scored
	cursor  int

	// Selected is set to the chosen repo name when the user presses Enter.
	Selected string
}

func newModel(repos []string, fre *Frecency, now int64, alpha float64) model {
	ti := textinput.New()
	ti.Placeholder = "type a repo…"
	ti.Prompt = "repo> "
	ti.Focus()

	m := model{ti: ti, repos: repos, fre: fre, now: now, alpha: alpha}
	m.recompute()
	return m
}

// recompute rebuilds the ranked result set for the current query. With an empty
// query every repo "matches" and ordering is pure frecency, so the list opens
// on your most-used repos.
func (m *model) recompute() {
	q := m.ti.Value()
	res := make([]scored, 0, len(m.repos))
	for _, name := range m.repos {
		mt, ok := FuzzyMatch(q, name)
		if !ok {
			continue
		}
		raw := m.fre.Score(name, m.now)
		freWeight := math.Log2(1 + raw) // dampen so a hot repo boosts but doesn't bulldoze
		pos := make(map[int]bool, len(mt.Positions))
		for _, p := range mt.Positions {
			pos[p] = true
		}
		res = append(res, scored{
			name:      name,
			positions: pos,
			final:     mt.Score + m.alpha*freWeight,
			fre:       raw,
			count:     m.fre.Count(name),
		})
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].final != res[j].final {
			return res[i].final > res[j].final
		}
		return res[i].name < res[j].name
	})
	m.results = res
	if m.cursor >= len(res) {
		m.cursor = 0
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			if len(m.results) > 0 {
				m.Selected = m.results[m.cursor].name
			}
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.results)-1 && m.cursor < maxRows-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.ti.Value()
	m.ti, cmd = m.ti.Update(msg)
	if m.ti.Value() != prev {
		m.cursor = 0
		m.recompute()
	}
	return m, cmd
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(m.ti.View() + "\n")

	n := len(m.results)
	if n > maxRows {
		n = maxRows
	}
	for i := 0; i < n; i++ {
		r := m.results[i]
		pointer := "  "
		if i == m.cursor {
			pointer = cursorStyle.Render("› ")
		}
		b.WriteString(pointer + highlight(r.name, r.positions))
		if r.count > 0 {
			b.WriteString(" " + freStyle.Render("×"+itoa(r.count)))
		}
		b.WriteString("\n")
	}
	if len(m.results) == 0 {
		b.WriteString(dimStyle.Render("  no matches") + "\n")
	}

	total := len(m.results)
	b.WriteString("\n" + dimStyle.Render(
		"↑/↓ move · enter open · esc quit · "+plural(total)+" matched"))
	return b.String()
}

// highlight renders name with matched rune positions emphasized.
func highlight(name string, pos map[int]bool) string {
	if len(pos) == 0 {
		return name
	}
	var b strings.Builder
	for i, r := range []rune(name) {
		s := string(r)
		if pos[i] {
			b.WriteString(matchStyle.Render(s))
		} else {
			b.WriteString(s)
		}
	}
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return "1 repo"
	}
	return itoa(n) + " repos"
}

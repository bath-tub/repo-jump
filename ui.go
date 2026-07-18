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
	matchStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")) // green (unselected rows)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))           // help text

	// Inverse style: the cursor is a full-width bar. Every segment on the bar
	// sets the same background so it reads as one continuous strip.
	selBg    = lipgloss.Color("239")
	barBase  = lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color("231"))
	barMatch = lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color("48")).Bold(true)
	barCount = lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color("180"))
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
	cursor  int // absolute index into results
	offset  int // index of the first visible row (scroll window)
	width   int
	height  int

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
	m.offset = 0
}

// visibleRows is how many result rows fit: terminal height minus the prompt
// line, a blank line, and the help line, capped for sanity. Before the first
// window-size message (height 0) we fall back to maxRows.
func (m model) visibleRows() int {
	v := maxRows
	if m.height > 0 {
		v = m.height - 3
	}
	if v < 1 {
		v = 1
	}
	if v > maxRows {
		v = maxRows
	}
	if v > len(m.results) {
		v = len(m.results)
	}
	return v
}

// reconcile keeps the scroll window around the cursor.
func (m *model) reconcile() {
	vis := m.visibleRows()
	if vis < 1 {
		m.offset = 0
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.reconcile()
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
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
				m.reconcile()
			}
			return m, nil
		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.reconcile()
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

	if len(m.results) == 0 {
		b.WriteString(dimStyle.Render("  no matches") + "\n")
	}

	vis := m.visibleRows()
	for k := 0; k < vis; k++ {
		i := m.offset + k
		if i >= len(m.results) {
			break
		}
		r := m.results[i]
		if i == m.cursor {
			b.WriteString(m.renderBar(r)) // full-width selection bar
		} else {
			b.WriteString(highlight(r.name, r.positions)) // matches only, no count
		}
		b.WriteString("\n")
	}

	total := len(m.results)
	b.WriteString("\n" + dimStyle.Render(
		"↑/↓ move · enter open · esc quit · "+plural(total)+" matched"))
	return b.String()
}

// renderBar draws the selected row as one continuous full-width bar: name (with
// matched chars picked out), the ×N count, then padding — every segment sharing
// the bar background so it reads as a single strip.
func (m model) renderBar(r scored) string {
	runes := []rune(r.name)
	var b strings.Builder

	// Group consecutive matched/unmatched runs to keep the escape count low.
	for i := 0; i < len(runes); {
		matched := r.positions[i]
		j := i
		for j < len(runes) && r.positions[j] == matched {
			j++
		}
		seg := string(runes[i:j])
		if matched {
			b.WriteString(barMatch.Render(seg))
		} else {
			b.WriteString(barBase.Render(seg))
		}
		i = j
	}

	visible := len(runes)
	if r.count > 0 {
		tail := "  ×" + itoa(r.count)
		b.WriteString(barCount.Render(tail))
		visible += len([]rune(tail))
	}
	if pad := m.width - visible; pad > 0 {
		b.WriteString(barBase.Render(strings.Repeat(" ", pad)))
	}
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

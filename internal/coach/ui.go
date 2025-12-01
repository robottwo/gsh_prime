package coach

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	engine         *Engine
	analysis       *AnalysisEngine
	stats          *UserStats
	insights       []Insight
	progress       progress.Model
	width          int
	height         int
	quitting       bool
	err            error
}

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	titleStyle = lipgloss.NewStyle().
			MarginLeft(1).
			MarginRight(5).
			Padding(0, 1).
			Italic(true).
			Foreground(lipgloss.Color("#FFF7DB")).
			SetString("Coach Dashboard")

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(1, 2)

	statStyle = lipgloss.NewStyle().
			Foreground(subtle)

	insightStyle = lipgloss.NewStyle().
			Foreground(special).
			Italic(true)
)

func NewModel(am *analytics.AnalyticsManager) model {
	engine := NewEngine(am)
	analysis := NewAnalysisEngine(am)

	stats, _ := engine.CalculateStats() // Handle error in view or init
	insights, _ := analysis.GenerateInsights()

	prog := progress.New(progress.WithGradient("#5A56E0", "#EE6FF8"))

	return model{
		engine:   engine,
		analysis: analysis,
		stats:    stats,
		insights: insights,
		progress: prog,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 20
		if m.progress.Width < 0 {
			m.progress.Width = 0
		}
		if m.progress.Width > 50 {
			m.progress.Width = 50
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.stats == nil {
		return "Loading stats..."
	}

	doc := strings.Builder{}

	// Header
	doc.WriteString(titleStyle.Render() + "\n")
	doc.WriteString(fmt.Sprintf("Level %d: %s\n", m.stats.Level, lipgloss.NewStyle().Bold(true).Render(m.stats.Title)))
	doc.WriteString(m.progress.ViewAs(m.stats.Progress) + "\n")
	doc.WriteString(fmt.Sprintf("%d / %d XP to next level\n\n", m.stats.TotalXP, m.stats.NextLevelXP))

	// Stats Grid
	statsView := fmt.Sprintf("Total Commands: %d\nUnique Commands: %d\nCurrent Streak: %d days\n",
		m.stats.TotalCommands, m.stats.UniqueCommands, m.stats.Streak)
	doc.WriteString(boxStyle.Render(statsView) + "\n\n")

	// Insights (Coach's Corner)
	doc.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("Coach's Corner") + "\n")
	if len(m.insights) > 0 {
		// Just show one tip or insight randomly or the first one
		// Ideally we rotate them.
		tip := m.insights[0]
		doc.WriteString(insightStyle.Render("💡 " + tip.Message) + "\n")

		// If there is an alias suggestion, show it too
		for _, in := range m.insights {
			if in.Type == "alias" {
				doc.WriteString(insightStyle.Render("🚀 " + in.Message) + "\n")
				break
			}
		}
	} else {
		doc.WriteString("Keep using the shell to get insights!\n")
	}
	doc.WriteString("\n")

	// Achievements
	doc.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("Recent Achievements") + "\n")
	count := 0
	for _, a := range m.stats.Achievements {
		if a.Unlocked {
			doc.WriteString(fmt.Sprintf("%s %s - %s\n", a.Icon, a.Name, statStyle.Render(a.Description)))
			count++
			if count >= 5 { break } // Limit to 5
		}
	}
	if count == 0 {
		doc.WriteString(statStyle.Render("No achievements yet. Type more commands!") + "\n")
	}

	return doc.String() // let bubbletea handle fullscreen
}

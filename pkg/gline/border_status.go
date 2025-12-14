package gline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/git"
	"github.com/atinylittleshell/gsh/internal/system"
	"github.com/charmbracelet/lipgloss"
)

// Command Classification
type CommandKind int

const (
	KindRawShell CommandKind = iota
	KindAgentChat
	KindAgentControl
	KindSubagent
	KindUnknown
)

// Execution Risk States
type RiskLevel int

const (
	RiskCalm RiskLevel = iota
	RiskWarning
	RiskAlert
)

// BorderStatusModel manages the state and rendering of border status elements
type BorderStatusModel struct {
	width int

	// Input State
	commandBuffer string
	kind          CommandKind
	riskScore     int
	riskLevel     RiskLevel

	// Context State
	user      string
	host      string
	cwd       string
	gitStatus *git.RepoStatus

	// Resource State
	resources *system.Resources

	// Styles
	styles BorderStyles
}

type BorderStyles struct {
	BadgeRaw     lipgloss.Style
	BadgeAgent   lipgloss.Style
	BadgeControl lipgloss.Style
	BadgeSub     lipgloss.Style

	RiskCalm    lipgloss.Style
	RiskWarning lipgloss.Style
	RiskAlert   lipgloss.Style

	ContextUser lipgloss.Style
	ContextDir  lipgloss.Style
	ContextGit  lipgloss.Style
	Divider     lipgloss.Style

	ResCool  lipgloss.Style
	ResWarm  lipgloss.Style
	ResHot   lipgloss.Style
	ResLabel lipgloss.Style
}

func NewBorderStatusModel() BorderStatusModel {
	s := BorderStyles{
		BadgeRaw:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")), // gray
		BadgeAgent:   lipgloss.NewStyle().Foreground(lipgloss.Color("75")),  // blue
		BadgeControl: lipgloss.NewStyle().Foreground(lipgloss.Color("208")), // orange
		BadgeSub:     lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // purple

		RiskCalm:    lipgloss.NewStyle().Foreground(lipgloss.Color("77")),  // green
		RiskWarning: lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // amber
		RiskAlert:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")), // red

		ContextUser: lipgloss.NewStyle().Foreground(lipgloss.Color("241")), // dim gray
		ContextDir:  lipgloss.NewStyle().Foreground(lipgloss.Color("39")),  // blueish
		ContextGit:  lipgloss.NewStyle().Foreground(lipgloss.Color("246")), // gray default
		Divider:     lipgloss.NewStyle().Foreground(lipgloss.Color("238")), // dark gray

		ResCool:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),  // green
		ResWarm:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // amber
		ResHot:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")), // red
		ResLabel: lipgloss.NewStyle().Foreground(lipgloss.Color("240")), // dim
	}

	return BorderStatusModel{
		styles: s,
	}
}

func (m *BorderStatusModel) SetWidth(w int) {
	m.width = w
}

func (m *BorderStatusModel) UpdateInput(input string) {
	if m.commandBuffer == input {
		return
	}
	m.commandBuffer = input
	m.classifyCommand()
	m.computeRisk()
}

func (m *BorderStatusModel) UpdateContext(user, host, cwd string) {
	m.user = user
	m.host = host
	m.cwd = cwd
}

func (m *BorderStatusModel) UpdateGit(status *git.RepoStatus) {
	m.gitStatus = status
}

func (m *BorderStatusModel) UpdateResources(res *system.Resources) {
	m.resources = res
}

func (m *BorderStatusModel) classifyCommand() {
	input := strings.TrimSpace(m.commandBuffer)
	if strings.HasPrefix(input, "@!") {
		m.kind = KindAgentControl
	} else if strings.HasPrefix(input, "@") {
		// Check for subagent?
		// For now simple heuristic: @name ...
		parts := strings.Fields(input)
		if len(parts) > 0 && len(parts[0]) > 1 {
			// Assume subagent if it looks like one, or agent chat if just @
			// Spec says: starts with @ followed by text but not @! is agent chat.
			// starts with @name where name matches subagent is Subagent.
			// We don't have subagent list here easily.
			// We'll treat all @name as subagent/agent potentially.
			// Let's stick to simple: @ is chat, @name is subagent?
			// Spec: "Agent chat: starts with @ followed by text... (e.g. @explain)"
			// Wait, "@explain" is a command to the default agent? Or is "explain" a subagent?
			// In gsh, "@" invokes default agent. "@@" selects subagent?
			// Spec says: "Subagent: starts with @name where name matches a discovered subagent."
			// Since we don't know discovered subagents here, we might need to default to Agent Chat
			// unless we can verify.
			// Let's simplify: @! -> Control. @... -> Chat (default).
			// If we want to distinguish Subagent, we'd need injection of known subagents.
			// For now, map all @... to KindAgentChat or KindSubagent based on simple rule?
			// Let's map @ (alone) or @ followed by space to Chat.
			// @word to Subagent?
			if input == "@" || strings.HasPrefix(input, "@ ") {
				m.kind = KindAgentChat
			} else {
				// @something
				m.kind = KindSubagent
			}
		} else {
			m.kind = KindAgentChat
		}
	} else {
		m.kind = KindRawShell
	}
}

func (m *BorderStatusModel) computeRisk() {
	// Simple heuristic implementation
	score := 0
	input := m.commandBuffer

	// Destructive flags
	if strings.Contains(input, "rm -rf") || strings.Contains(input, ":(){") || strings.Contains(input, "mkfs") {
		score += 5
	}
	if strings.Contains(input, "dd if=") || strings.Contains(input, "chmod -R") || strings.Contains(input, "chown -R") {
		score += 3
	}

	// Privilege
	if strings.HasPrefix(input, "sudo") || strings.HasPrefix(input, "doas") || strings.HasPrefix(input, "su -") {
		score += 2
	}

	// Network
	if strings.Contains(input, "curl") || strings.Contains(input, "wget") || strings.Contains(input, "ssh") || strings.Contains(input, "scp") {
		score += 1
	}

	// Safe signals
	if strings.HasPrefix(input, "echo") || strings.HasPrefix(input, "ls") || strings.HasPrefix(input, "pwd") {
		score -= 1
	}

	if score < 0 {
		score = 0
	}
	m.riskScore = score

	if score <= 2 {
		m.riskLevel = RiskCalm
	} else if score <= 5 {
		m.riskLevel = RiskWarning
	} else {
		m.riskLevel = RiskAlert
	}
}

// Rendering

func (m BorderStatusModel) RenderTopLeft() string {
	// Badge
	var badge string
	var style lipgloss.Style
	switch m.kind {
	case KindRawShell:
		badge = "$"
		style = m.styles.BadgeRaw
	case KindAgentChat:
		badge = "🤖" // or @
		style = m.styles.BadgeAgent
	case KindAgentControl:
		badge = "!"
		style = m.styles.BadgeControl
	case KindSubagent:
		badge = "◇"
		style = m.styles.BadgeSub
	default:
		badge = "?"
		style = m.styles.BadgeRaw
	}

	// Risk Meter
	// ▂▆█
	// 3-5 cells.
	// Calm: muted/green thin bar
	// Warning: amber bar 2-3 ticks
	// Alert: red bar full

	var riskBar string
	var riskStyle lipgloss.Style

	switch m.riskLevel {
	case RiskCalm:
		riskBar = "▂"
		riskStyle = m.styles.RiskCalm
	case RiskWarning:
		riskBar = "▂▆"
		riskStyle = m.styles.RiskWarning
	case RiskAlert:
		riskBar = "▂▆█"
		riskStyle = m.styles.RiskAlert
	}

	// If space is tight, we might compress.
	// For now render full.

	return style.Render(badge) + " " + riskStyle.Render(riskBar)
}

func (m BorderStatusModel) RenderTopContext(maxWidth int) string {
	// [user@host]──[~/repo/subdir]──[git]
	// Separator: ╲ or ─
	sep := m.styles.Divider.Render("─")

	// User@Host
	host := m.host
	if len(host) > 8 {
		host = host[:8]
	}
	uHost := fmt.Sprintf("%s@%s", m.user, host)
	if m.user == "" {
		uHost = ""
	}

	// Dir
	// Collapse middle if long
	dir := m.cwd
	// Try to replace home
	// We don't have user home here easily, assume cwd is already processed or we do best effort
	// Let's just use base if too long
	if len(dir) > 20 {
		dir = filepath.Base(dir)
		if dir == "/" {
			dir = "/"
		} else {
			dir = ".../" + dir
		}
	}

	// Git
	gitStr := ""
	var gitStyle lipgloss.Style = m.styles.ContextGit

	if m.gitStatus != nil {
		// repo:branch clean/dirty
		repo := m.gitStatus.RepoName
		branch := m.gitStatus.Branch
		if len(branch) > 12 {
			branch = branch[:11] + "…"
		}

		var symbol string
		if !m.gitStatus.Clean {
			if m.gitStatus.Conflict {
				symbol = "!"
				gitStyle = m.styles.RiskAlert
			} else {
				symbol = "●"
				gitStyle = m.styles.RiskWarning
			}
		} else {
			symbol = "✓" // or just clean
			gitStyle = m.styles.RiskCalm
		}

		gitStr = fmt.Sprintf("%s:%s %s", repo, branch, symbol)
		if m.gitStatus.Ahead > 0 {
			gitStr += fmt.Sprintf(" ⬆%d", m.gitStatus.Ahead)
		}
		if m.gitStatus.Behind > 0 {
			gitStr += fmt.Sprintf(" ⬇%d", m.gitStatus.Behind)
		}
	}

	// Compose with priority
	// 1. Badge+Risk (already handled in TopLeft, this is Top Edge)
	// Actually TopLeft is separate.
	// We are rendering the line content.

	// We need to fit: uHost, dir, gitStr
	// If constrained, drop git, then uHost

	parts := []string{}
	if uHost != "" {
		parts = append(parts, m.styles.ContextUser.Render(uHost))
	}
	if dir != "" {
		parts = append(parts, m.styles.ContextDir.Render(dir))
	}
	if gitStr != "" {
		parts = append(parts, gitStyle.Render(gitStr))
	}

	// Join with separators
	full := strings.Join(parts, sep)
	if lipgloss.Width(full) <= maxWidth {
		return full
	}

	// Drop Git
	if gitStr != "" {
		parts = parts[:len(parts)-1]
		full = strings.Join(parts, sep)
		if lipgloss.Width(full) <= maxWidth {
			return full
		}
	}

	// Drop UserHost
	if uHost != "" && len(parts) > 1 {
		// Keep Dir
		return m.styles.ContextDir.Render(dir)
	}

	// Just Dir (maybe truncated more)
	return m.styles.ContextDir.Render(dir)
}

func (m BorderStatusModel) RenderBottomLeft() string {
	if m.resources == nil {
		return m.styles.ResLabel.Render("C? R?")
	}

	// CPU
	cpu := m.resources.CPUPercent
	cpuBar := m.renderMiniBar(cpu/100.0)
	cpuStr := m.styles.ResLabel.Render("C") + cpuBar

	// RAM
	ramRatio := 0.0
	if m.resources.RAMTotal > 0 {
		ramRatio = float64(m.resources.RAMUsed) / float64(m.resources.RAMTotal)
	}
	ramBar := m.renderMiniBar(ramRatio)
	ramStr := m.styles.ResLabel.Render("R") + ramBar

	return cpuStr + " " + ramStr
}

func (m BorderStatusModel) renderMiniBar(ratio float64) string {
	// 3 chars: ▂▆█
	// logic:
	// < 0.33: ▂
	// < 0.66: ▂▆
	// else: ▂▆█

	// Color:
	// < 0.5: Cool (Green)
	// < 0.8: Warm (Amber)
	// > 0.8: Hot (Red)

	var style lipgloss.Style
	if ratio < 0.5 {
		style = m.styles.ResCool
	} else if ratio < 0.8 {
		style = m.styles.ResWarm
	} else {
		style = m.styles.ResHot
	}

	var bar string
	if ratio < 0.1 {
		bar = " " // empty
	} else if ratio < 0.4 {
		bar = "▂"
	} else if ratio < 0.7 {
		bar = "▂▆"
	} else {
		bar = "▂▆█"
	}

	return style.Render(bar)
}

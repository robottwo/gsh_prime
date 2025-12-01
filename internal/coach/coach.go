package coach

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"go.uber.org/zap"
)

// Coach provides shell usage insights and optimization suggestions
type Coach struct {
	analyticsManager *analytics.AnalyticsManager
	logger           *zap.Logger
}

// NewCoach creates a new Coach instance
func NewCoach(analyticsManager *analytics.AnalyticsManager, logger *zap.Logger) *Coach {
	return &Coach{
		analyticsManager: analyticsManager,
		logger:           logger,
	}
}

// Insight represents a single coaching insight
type Insight struct {
	Type        string // "alias", "accuracy", "error", "tip", "achievement"
	Title       string
	Description string
	Command     string // For alias suggestions
	Alias       string // Suggested alias name
	Count       int    // Relevant count (e.g., command usage count)
	Percentage  float64
}

// Report contains all coaching insights
type Report struct {
	TotalCommands    int64
	PeriodDays       int
	PredictionRate   float64
	TopCommands      []CommandCount
	AliasSuggestions []Insight
	Insights         []Insight
	GeneratedAt      time.Time
}

// CommandCount represents a command and its usage count
type CommandCount struct {
	Command string
	Count   int
}

// GenerateReport creates a comprehensive coaching report
func (c *Coach) GenerateReport() (*Report, error) {
	report := &Report{
		GeneratedAt:      time.Now(),
		PeriodDays:       7,
		TopCommands:      []CommandCount{},
		AliasSuggestions: []Insight{},
		Insights:         []Insight{},
	}

	// Get total count
	totalCount, err := c.analyticsManager.GetTotalCount()
	if err != nil {
		c.logger.Warn("failed to get total count", zap.Error(err))
	}
	report.TotalCommands = totalCount

	// Get recent entries for analysis (last 7 days worth, up to 10000)
	entries, err := c.analyticsManager.GetRecentEntries(10000)
	if err != nil {
		c.logger.Warn("failed to get entries for coach analysis", zap.Error(err))
		return report, nil
	}

	if len(entries) == 0 {
		report.Insights = append(report.Insights, Insight{
			Type:        "tip",
			Title:       "Getting Started",
			Description: "Keep using gsh to collect usage data. Insights will appear as you use the shell!",
		})
		return report, nil
	}

	// Filter to recent entries (last 7 days)
	cutoff := time.Now().AddDate(0, 0, -7)
	var recentEntries []analytics.AnalyticsEntry
	for _, e := range entries {
		if e.CreatedAt.After(cutoff) {
			recentEntries = append(recentEntries, e)
		}
	}

	// If not enough recent entries, use all entries
	if len(recentEntries) < 10 {
		recentEntries = entries
		report.PeriodDays = 0 // Indicates all-time
	}

	// Analyze command frequency
	commandCounts := c.analyzeCommandFrequency(recentEntries)
	report.TopCommands = commandCounts

	// Calculate prediction accuracy
	report.PredictionRate = c.calculatePredictionAccuracy(recentEntries)

	// Generate alias suggestions
	report.AliasSuggestions = c.generateAliasSuggestions(commandCounts)

	// Generate additional insights
	report.Insights = c.generateInsights(recentEntries, commandCounts, report.PredictionRate)

	return report, nil
}

// analyzeCommandFrequency counts how often each command is used
func (c *Coach) analyzeCommandFrequency(entries []analytics.AnalyticsEntry) []CommandCount {
	counts := make(map[string]int)

	for _, entry := range entries {
		cmd := normalizeCommand(entry.Actual)
		if cmd != "" {
			counts[cmd]++
		}
	}

	// Convert to slice and sort by count
	var result []CommandCount
	for cmd, count := range counts {
		result = append(result, CommandCount{Command: cmd, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	// Return top 10
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// normalizeCommand extracts the base command for frequency analysis
func normalizeCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	// Split on common delimiters and take the main command structure
	// For git commands, keep the subcommand (e.g., "git status")
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}

	// Handle git and similar multi-word commands
	if len(parts) >= 2 && isCompoundCommand(parts[0]) {
		return parts[0] + " " + parts[1]
	}

	return parts[0]
}

// isCompoundCommand checks if a command typically has important subcommands
func isCompoundCommand(cmd string) bool {
	compounds := map[string]bool{
		"git":     true,
		"docker":  true,
		"kubectl": true,
		"npm":     true,
		"yarn":    true,
		"go":      true,
		"cargo":   true,
		"pip":     true,
		"apt":     true,
		"brew":    true,
		"systemctl": true,
	}
	return compounds[cmd]
}

// calculatePredictionAccuracy calculates how often predictions matched actual commands
func (c *Coach) calculatePredictionAccuracy(entries []analytics.AnalyticsEntry) float64 {
	if len(entries) == 0 {
		return 0
	}

	matches := 0
	total := 0

	for _, entry := range entries {
		// Skip entries without predictions
		if entry.Prediction == "" {
			continue
		}
		total++

		// Check if prediction matches (exact or prefix match)
		if entry.Prediction == entry.Actual ||
			strings.HasPrefix(entry.Actual, entry.Prediction) ||
			strings.HasPrefix(entry.Prediction, entry.Actual) {
			matches++
		}
	}

	if total == 0 {
		return 0
	}

	return float64(matches) / float64(total) * 100
}

// generateAliasSuggestions creates alias recommendations for frequent commands
func (c *Coach) generateAliasSuggestions(commandCounts []CommandCount) []Insight {
	var suggestions []Insight

	// Common alias patterns
	aliasPatterns := map[string]string{
		"git status":      "gs",
		"git add":         "ga",
		"git commit":      "gc",
		"git push":        "gp",
		"git pull":        "gl",
		"git checkout":    "gco",
		"git branch":      "gb",
		"git diff":        "gd",
		"git log":         "glog",
		"docker ps":       "dps",
		"docker images":   "dimg",
		"docker compose":  "dc",
		"kubectl get":     "kg",
		"kubectl describe": "kd",
		"npm install":     "ni",
		"npm run":         "nr",
		"cd ..":           "..",
		"ls -la":          "ll",
		"ls -l":           "l",
	}

	for _, cc := range commandCounts {
		// Only suggest for commands used more than 10 times
		if cc.Count < 10 {
			continue
		}

		// Check if we have a known alias pattern
		if alias, ok := aliasPatterns[cc.Command]; ok {
			suggestions = append(suggestions, Insight{
				Type:        "alias",
				Title:       fmt.Sprintf("Create alias '%s'", alias),
				Description: fmt.Sprintf("You typed `%s` %d times. Consider: `alias %s='%s'`", cc.Command, cc.Count, alias, cc.Command),
				Command:     cc.Command,
				Alias:       alias,
				Count:       cc.Count,
			})
		} else if len(cc.Command) > 5 && cc.Count >= 20 {
			// For other long commands used frequently, suggest a generic alias
			alias := generateAlias(cc.Command)
			suggestions = append(suggestions, Insight{
				Type:        "alias",
				Title:       fmt.Sprintf("Create alias '%s'", alias),
				Description: fmt.Sprintf("You typed `%s` %d times. Consider: `alias %s='%s'`", cc.Command, cc.Count, alias, cc.Command),
				Command:     cc.Command,
				Alias:       alias,
				Count:       cc.Count,
			})
		}
	}

	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// generateAlias creates a short alias name from a command
func generateAlias(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "cmd"
	}

	// For compound commands, use first letter of each word
	if len(parts) >= 2 {
		alias := ""
		for _, p := range parts {
			if len(p) > 0 {
				alias += string(p[0])
			}
		}
		return alias
	}

	// For single commands, abbreviate
	if len(parts[0]) > 3 {
		return parts[0][:3]
	}
	return parts[0]
}

// generateInsights creates various coaching insights
func (c *Coach) generateInsights(entries []analytics.AnalyticsEntry, topCommands []CommandCount, accuracy float64) []Insight {
	var insights []Insight

	// Prediction accuracy insight
	if accuracy > 0 {
		var desc string
		var title string
		switch {
		case accuracy >= 80:
			title = "Excellent Prediction Accuracy!"
			desc = fmt.Sprintf("gsh predicted %.1f%% of your commands correctly. You're in sync with your AI assistant!", accuracy)
		case accuracy >= 60:
			title = "Good Prediction Accuracy"
			desc = fmt.Sprintf("gsh predicted %.1f%% of your commands. Keep building consistent patterns!", accuracy)
		case accuracy >= 40:
			title = "Growing Prediction Accuracy"
			desc = fmt.Sprintf("gsh predicted %.1f%% of your commands. The more you use gsh, the better it learns!", accuracy)
		default:
			title = "Learning Your Style"
			desc = fmt.Sprintf("gsh is learning your command patterns (%.1f%% accuracy so far).", accuracy)
		}
		insights = append(insights, Insight{
			Type:        "accuracy",
			Title:       title,
			Description: desc,
			Percentage:  accuracy,
		})
	}

	// Command diversity insight
	if len(entries) > 0 && len(topCommands) > 0 {
		topCount := 0
		for _, tc := range topCommands[:min(3, len(topCommands))] {
			topCount += tc.Count
		}
		concentration := float64(topCount) / float64(len(entries)) * 100

		if concentration > 50 {
			insights = append(insights, Insight{
				Type:        "tip",
				Title:       "Power User Pattern",
				Description: fmt.Sprintf("Your top 3 commands account for %.0f%% of your usage. Great candidates for aliases!", concentration),
			})
		}
	}

	// Achievement badges based on total commands
	totalCommands, _ := c.analyticsManager.GetTotalCount()
	if totalCommands >= 1000 {
		insights = append(insights, Insight{
			Type:        "achievement",
			Title:       "Shell Master",
			Description: fmt.Sprintf("You've run %d commands with gsh! You're a power user!", totalCommands),
			Count:       int(totalCommands),
		})
	} else if totalCommands >= 500 {
		insights = append(insights, Insight{
			Type:        "achievement",
			Title:       "Command Line Pro",
			Description: fmt.Sprintf("You've run %d commands with gsh! Keep it up!", totalCommands),
			Count:       int(totalCommands),
		})
	} else if totalCommands >= 100 {
		insights = append(insights, Insight{
			Type:        "achievement",
			Title:       "Getting Comfortable",
			Description: fmt.Sprintf("You've run %d commands with gsh. The journey continues!", totalCommands),
			Count:       int(totalCommands),
		})
	}

	return insights
}

// GetQuickTip returns a brief insight suitable for the assistant box
func (c *Coach) GetQuickTip() string {
	report, err := c.GenerateReport()
	if err != nil || report == nil {
		return ""
	}

	// Priority: alias suggestion > accuracy > achievement
	if len(report.AliasSuggestions) > 0 {
		s := report.AliasSuggestions[0]
		return fmt.Sprintf("Tip: `%s` (%dx) - try `alias %s='%s'`", s.Command, s.Count, s.Alias, s.Command)
	}

	if report.PredictionRate > 0 {
		return fmt.Sprintf("Prediction accuracy: %.0f%% | Commands: %d", report.PredictionRate, report.TotalCommands)
	}

	if report.TotalCommands > 0 {
		return fmt.Sprintf("Commands tracked: %d | Use @!coach for insights", report.TotalCommands)
	}

	return "Use gsh to track your shell patterns | @!coach for insights"
}

// FormatReport returns a formatted string of the coaching report
func (c *Coach) FormatReport(report *Report) string {
	var sb strings.Builder

	// Header
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║              gsh Coach - Your Shell Wrapped                   ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Period
	if report.PeriodDays > 0 {
		sb.WriteString(fmt.Sprintf("Period: Last %d days\n", report.PeriodDays))
	} else {
		sb.WriteString("Period: All time\n")
	}
	sb.WriteString(fmt.Sprintf("Total Commands: %d\n\n", report.TotalCommands))

	// Top Commands
	if len(report.TopCommands) > 0 {
		sb.WriteString("═══ Your Top Commands ═══\n")
		for i, tc := range report.TopCommands {
			bar := strings.Repeat("█", min(20, tc.Count/max(1, report.TopCommands[0].Count/20+1)))
			sb.WriteString(fmt.Sprintf("%2d. %-20s %s (%d)\n", i+1, tc.Command, bar, tc.Count))
		}
		sb.WriteString("\n")
	}

	// Prediction Accuracy
	if report.PredictionRate > 0 {
		sb.WriteString("═══ Prediction Stats ═══\n")
		sb.WriteString(fmt.Sprintf("Accuracy: %.1f%%\n", report.PredictionRate))
		// Visual bar
		filledBars := int(report.PredictionRate / 5)
		emptyBars := 20 - filledBars
		sb.WriteString(fmt.Sprintf("[%s%s]\n\n", strings.Repeat("█", filledBars), strings.Repeat("░", emptyBars)))
	}

	// Alias Suggestions
	if len(report.AliasSuggestions) > 0 {
		sb.WriteString("═══ Alias Suggestions ═══\n")
		for _, s := range report.AliasSuggestions {
			sb.WriteString(fmt.Sprintf("• %s\n", s.Description))
		}
		sb.WriteString("\n")
	}

	// Other Insights
	if len(report.Insights) > 0 {
		sb.WriteString("═══ Insights ═══\n")
		for _, insight := range report.Insights {
			emoji := getInsightEmoji(insight.Type)
			sb.WriteString(fmt.Sprintf("%s %s: %s\n", emoji, insight.Title, insight.Description))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("Run @!coach to see this report anytime\n")

	return sb.String()
}

func getInsightEmoji(insightType string) string {
	switch insightType {
	case "accuracy":
		return "📊"
	case "achievement":
		return "🏆"
	case "tip":
		return "💡"
	case "alias":
		return "⚡"
	default:
		return "•"
	}
}

// ApplyAlias writes an alias to the user's .gshrc file
func (c *Coach) ApplyAlias(aliasName, command string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	gshrcPath := filepath.Join(homeDir, ".gshrc")

	// Read existing content
	content, err := os.ReadFile(gshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gshrc: %w", err)
	}

	// Check if alias already exists
	aliasLine := fmt.Sprintf("alias %s='%s'", aliasName, command)
	if strings.Contains(string(content), aliasLine) {
		return fmt.Errorf("alias '%s' already exists in .gshrc", aliasName)
	}

	// Append the alias
	f, err := os.OpenFile(gshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gshrc: %w", err)
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to .gshrc: %w", err)
		}
	}

	// Write comment and alias
	comment := fmt.Sprintf("\n# Added by gsh coach on %s\n", time.Now().Format("2006-01-02"))
	if _, err := f.WriteString(comment + aliasLine + "\n"); err != nil {
		return fmt.Errorf("failed to write alias to .gshrc: %w", err)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

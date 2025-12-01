package coach

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// Coach provides shell usage insights and optimization suggestions
type Coach struct {
	analyticsManager *analytics.AnalyticsManager
	historyManager   *history.HistoryManager
	logger           *zap.Logger
	runner           *interp.Runner

	// Fast model for quick tips (assistant box)
	fastLLMClient   *openai.Client
	fastModelId     string
	fastTemperature *float64

	// Slow model for detailed tips (@!coach command)
	slowLLMClient   *openai.Client
	slowModelId     string
	slowTemperature *float64

	// Cache for quick tips (populated once per session)
	quickTipCache      []string
	quickTipCacheMutex sync.Mutex
	quickTipCacheError string
}

// NewCoach creates a new Coach instance
func NewCoach(analyticsManager *analytics.AnalyticsManager, historyManager *history.HistoryManager, runner *interp.Runner, logger *zap.Logger) *Coach {
	fastClient, fastConfig := utils.GetLLMClient(runner, utils.FastModel)
	slowClient, slowConfig := utils.GetLLMClient(runner, utils.SlowModel)
	return &Coach{
		analyticsManager: analyticsManager,
		historyManager:   historyManager,
		logger:           logger,
		runner:           runner,
		fastLLMClient:    fastClient,
		fastModelId:      fastConfig.ModelId,
		fastTemperature:  fastConfig.Temperature,
		slowLLMClient:    slowClient,
		slowModelId:      slowConfig.ModelId,
		slowTemperature:  slowConfig.Temperature,
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
		"git":       true,
		"docker":    true,
		"kubectl":   true,
		"npm":       true,
		"yarn":      true,
		"go":        true,
		"cargo":     true,
		"pip":       true,
		"apt":       true,
		"brew":      true,
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

	// Get existing aliases to avoid suggesting duplicates
	existingAliases := getExistingAliases()

	// Common alias patterns
	aliasPatterns := map[string]string{
		"git status":       "gs",
		"git add":          "ga",
		"git commit":       "gc",
		"git push":         "gp",
		"git pull":         "gl",
		"git checkout":     "gco",
		"git branch":       "gb",
		"git diff":         "gd",
		"git log":          "glog",
		"docker ps":        "dps",
		"docker images":    "dimg",
		"docker compose":   "dc",
		"kubectl get":      "kg",
		"kubectl describe": "kd",
		"npm install":      "ni",
		"npm run":          "nr",
		"cd ..":            "..",
		"ls -la":           "ll",
		"ls -l":            "l",
	}

	for _, cc := range commandCounts {
		// Only suggest for commands used more than 10 times
		if cc.Count < 10 {
			continue
		}

		// Skip if this command already has an alias
		if commandHasAlias(cc.Command, existingAliases) {
			continue
		}

		// Check if we have a known alias pattern
		if alias, ok := aliasPatterns[cc.Command]; ok {
			// Skip if this alias name is already in use
			if aliasNameExists(alias, existingAliases) {
				continue
			}
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
			// Skip if this alias name is already in use
			if aliasNameExists(alias, existingAliases) {
				continue
			}
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

// defaultQuickTip is shown when there's not enough history for LLM-generated tips
const defaultQuickTip = "TIP: Use gsh to track your shell patterns | @!coach for insights"

// quickTipCacheSize is the number of tips to cache per session
const quickTipCacheSize = 10

// quickTipPrefix is prepended to all quick tips in the assistant box
const quickTipPrefix = "TIP: "

// GetQuickTip returns a brief insight suitable for the assistant box
// Uses LLM-generated tips when there's enough history, otherwise shows default tip
// Tips are cached per session and selected randomly
func (c *Coach) GetQuickTip() string {
	report, err := c.GenerateReport()
	if err != nil {
		return defaultQuickTip
	}
	if report == nil {
		return defaultQuickTip
	}

	// If fewer than 25 commands in history, use default tip instead of LLM
	if report.TotalCommands < 25 {
		return defaultQuickTip
	}

	// Check if cache needs to be populated
	c.quickTipCacheMutex.Lock()
	defer c.quickTipCacheMutex.Unlock()

	// If we have a cached error, return it
	if c.quickTipCacheError != "" {
		return fmt.Sprintf("Coach tip unavailable: %s", c.quickTipCacheError)
	}

	// If cache is empty, populate it
	if len(c.quickTipCache) == 0 {
		tips, errReason := c.generateQuickTips(report, quickTipCacheSize)
		if len(tips) > 0 {
			c.quickTipCache = tips
		} else {
			c.quickTipCacheError = errReason
			return fmt.Sprintf("Coach tip unavailable: %s", errReason)
		}
	}

	// Return a random tip from cache with prefix
	return quickTipPrefix + c.quickTipCache[rand.Intn(len(c.quickTipCache))]
}

// generateQuickTips creates multiple contextual tips using the slow LLM for caching
// Returns tips and error reason (empty if successful)
func (c *Coach) generateQuickTips(report *Report, count int) ([]string, string) {
	if c.slowLLMClient == nil {
		return nil, "LLM client not configured"
	}
	if report == nil || count <= 0 {
		return nil, "No report available"
	}

	// Build context about user's shell usage
	var contextParts []string

	if len(report.TopCommands) > 0 {
		topCmds := make([]string, 0, min(5, len(report.TopCommands)))
		for _, tc := range report.TopCommands[:min(5, len(report.TopCommands))] {
			topCmds = append(topCmds, fmt.Sprintf("%s (%d times)", tc.Command, tc.Count))
		}
		contextParts = append(contextParts, "Top commands: "+strings.Join(topCmds, ", "))
	}

	if report.PredictionRate > 0 {
		contextParts = append(contextParts, fmt.Sprintf("Prediction accuracy: %.0f%%", report.PredictionRate))
	}

	contextParts = append(contextParts, fmt.Sprintf("Total commands: %d", report.TotalCommands))

	usageContext := strings.Join(contextParts, ". ")

	systemPrompt := fmt.Sprintf(`You are a helpful shell productivity coach. Based on the user's shell usage patterns, generate exactly %d brief, actionable tips to help them be more productive.

Guidelines:
- Keep each tip under 80 characters
- Be specific and actionable
- Focus on shell productivity, shortcuts, or workflow improvements
- Don't suggest aliases (those are handled separately)
- Make each tip unique and different from the others
- Examples of good tips:
  - "Try 'cd -' to return to your previous directory"
  - "Use '!!' to repeat your last command"
  - "Ctrl+R searches your command history"
  - "Try 'git stash' to save uncommitted changes temporarily"

Respond with exactly %d tips, one per line, no numbering or bullets.`, count, count)

	userPrompt := fmt.Sprintf("User's shell usage: %s\n\nGenerate %d helpful productivity tips:", usageContext, count)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	request := openai.ChatCompletionRequest{
		Model: c.slowModelId,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   500,
		Temperature: 1.0, // Higher temperature for more varied tips
	}

	resp, err := c.slowLLMClient.CreateChatCompletion(ctx, request)
	if err != nil {
		c.logger.Debug("failed to generate quick tips", zap.Error(err))
		return nil, fmt.Sprintf("LLM request failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, "LLM returned no response"
	}

	// Parse the response - split by newlines and filter empty lines
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, "LLM returned empty content"
	}

	lines := strings.Split(content, "\n")
	var tips []string
	for _, line := range lines {
		tip := strings.TrimSpace(line)
		// Remove common prefixes like "1.", "- ", "• ", etc.
		tip = strings.TrimLeft(tip, "0123456789.-•* ")
		tip = strings.TrimSpace(tip)
		if tip != "" {
			// Truncate if too long
			if len(tip) > 100 {
				tip = tip[:97] + "..."
			}
			tips = append(tips, tip)
		}
	}

	if len(tips) == 0 {
		return nil, "LLM returned empty tips"
	}

	return tips, ""
}

// generateMultipleLLMTips creates multiple contextual tips using the slow LLM
// Returns tips and an error reason string (empty if successful)
func (c *Coach) generateMultipleLLMTips(report *Report, count int) ([]string, string) {
	if c.slowLLMClient == nil {
		return nil, "LLM client not configured"
	}
	if report == nil || count <= 0 {
		return nil, "Invalid report or count"
	}

	// Build context about user's shell usage
	var contextParts []string

	if len(report.TopCommands) > 0 {
		topCmds := make([]string, 0, min(10, len(report.TopCommands)))
		for _, tc := range report.TopCommands[:min(10, len(report.TopCommands))] {
			topCmds = append(topCmds, fmt.Sprintf("%s (%d times)", tc.Command, tc.Count))
		}
		contextParts = append(contextParts, "Top commands: "+strings.Join(topCmds, ", "))
	}

	if report.PredictionRate > 0 {
		contextParts = append(contextParts, fmt.Sprintf("Prediction accuracy: %.0f%%", report.PredictionRate))
	}

	contextParts = append(contextParts, fmt.Sprintf("Total commands: %d", report.TotalCommands))

	// Get recent command history for more context
	var commandHistoryContext string
	if c.historyManager != nil {
		historyEntries, err := c.historyManager.GetRecentEntries("", 50)
		if err == nil && len(historyEntries) > 0 {
			var commands []string
			for _, entry := range historyEntries {
				// Exclude "exit" command from history context
				if strings.TrimSpace(entry.Command) == "exit" {
					continue
				}
				commands = append(commands, entry.Command)
			}
			if len(commands) > 0 {
				commandHistoryContext = "\n\nRecent command history:\n" + strings.Join(commands, "\n")
			}
		}
	}

	usageContext := strings.Join(contextParts, ". ")

	systemPrompt := fmt.Sprintf(`You are a helpful shell productivity coach. Based on the user's shell usage patterns and recent command history, generate exactly %d brief, actionable tips to help them be more productive.

Guidelines:
- Keep each tip under 80 characters
- Be specific and actionable based on the user's actual commands
- Focus on shell productivity, shortcuts, or workflow improvements
- Try to provide a diverse set of suggestions
- Don't suggest commands that are already in the user's top commands
- Don't suggest aliases (those are handled separately)
- Make each tip unique and different from the others
- Tailor tips to the tools and workflows the user actually uses
- Examples of good tips:
  - "Try 'cd -' to return to your previous directory"
  - "Use '!!' to repeat your last command"
  - "Ctrl+R searches your command history"
  - "Try 'git stash' to save uncommitted changes temporarily"

Respond with exactly %d tips, one per line, no numbering or bullets.`, count, count)

	userPrompt := fmt.Sprintf("User's shell usage: %s%s\n\nGenerate %d helpful productivity tips:", usageContext, commandHistoryContext, count)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	request := openai.ChatCompletionRequest{
		Model: c.slowModelId,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 500,
	}
	if c.slowTemperature != nil {
		request.Temperature = float32(*c.slowTemperature)
	}

	resp, err := c.slowLLMClient.CreateChatCompletion(ctx, request)
	if err != nil {
		c.logger.Debug("failed to generate LLM tips", zap.Error(err))
		return nil, fmt.Sprintf("LLM request failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, "LLM returned no response"
	}

	// Parse the response - split by newlines and filter empty lines
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	lines := strings.Split(content, "\n")

	var tips []string
	for _, line := range lines {
		tip := strings.TrimSpace(line)
		// Remove common prefixes like "1.", "- ", "• ", etc.
		tip = strings.TrimLeft(tip, "0123456789.-•* ")
		tip = strings.TrimSpace(tip)
		if tip != "" {
			// Truncate if too long
			if len(tip) > 100 {
				tip = tip[:97] + "..."
			}
			tips = append(tips, tip)
		}
	}

	// Return up to the requested count
	if len(tips) > count {
		tips = tips[:count]
	}

	if len(tips) == 0 {
		return nil, "LLM returned empty tips"
	}

	return tips, ""
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

	// Top Commands (excluding short one-word commands)
	if len(report.TopCommands) > 0 {
		sb.WriteString("═══ Your Top Commands ═══\n")
		displayIndex := 0
		for _, tc := range report.TopCommands {
			// Skip one-word commands fewer than 6 characters
			if !strings.Contains(tc.Command, " ") && len(tc.Command) < 6 {
				continue
			}
			displayIndex++
			bar := strings.Repeat("█", min(20, tc.Count/max(1, report.TopCommands[0].Count/20+1)))
			sb.WriteString(fmt.Sprintf("%2d. %-20s %s (%d)\n", displayIndex, tc.Command, bar, tc.Count))
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

	// LLM-generated tips (using slow model with command history context)
	sb.WriteString("═══ AI-Powered Tips ═══\n")
	llmTips, errReason := c.generateMultipleLLMTips(report, 5)
	if len(llmTips) > 0 {
		for _, tip := range llmTips {
			sb.WriteString(fmt.Sprintf("💡 %s\n", tip))
		}
	} else {
		sb.WriteString(fmt.Sprintf("⚠️  Could not generate AI tips: %s\n", errReason))
	}
	sb.WriteString("\n")

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

	var writeErr error
	defer func() {
		if closeErr := f.Close(); closeErr != nil && writeErr == nil {
			writeErr = fmt.Errorf("failed to close %s: %w", gshrcPath, closeErr)
		}
	}()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			writeErr = fmt.Errorf("failed to write to .gshrc: %w", err)
			return writeErr
		}
	}

	// Write comment and alias
	comment := fmt.Sprintf("\n# Added by gsh coach on %s\n", time.Now().Format("2006-01-02"))
	if _, err := f.WriteString(comment + aliasLine + "\n"); err != nil {
		writeErr = fmt.Errorf("failed to write alias to .gshrc: %w", err)
		return writeErr
	}

	return writeErr
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

// aliasRegex matches alias definitions like: alias name='command' or alias name="command" or alias name=command
var aliasRegex = regexp.MustCompile(`(?m)^\s*alias\s+(\w+)=['"]?([^'"#\n]+)['"]?`)

// getExistingAliases reads .gshrc and returns a map of alias names to their commands
func getExistingAliases() map[string]string {
	aliases := make(map[string]string)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return aliases
	}

	gshrcPath := filepath.Join(homeDir, ".gshrc")
	content, err := os.ReadFile(gshrcPath)
	if err != nil {
		return aliases
	}

	matches := aliasRegex.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) >= 3 {
			aliasName := match[1]
			command := strings.TrimSpace(match[2])
			aliases[aliasName] = command
		}
	}

	return aliases
}

// commandHasAlias checks if a command already has an alias defined
func commandHasAlias(command string, existingAliases map[string]string) bool {
	for _, aliasCmd := range existingAliases {
		if aliasCmd == command {
			return true
		}
	}
	return false
}

// aliasNameExists checks if an alias name is already in use
func aliasNameExists(aliasName string, existingAliases map[string]string) bool {
	_, exists := existingAliases[aliasName]
	return exists
}

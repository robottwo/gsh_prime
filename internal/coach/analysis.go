package coach

import (
	"strconv"

	"github.com/atinylittleshell/gsh/internal/analytics"
)

type AnalysisEngine struct {
	Manager *analytics.AnalyticsManager
}

type Insight struct {
	Type    string // "alias", "tip", "fact"
	Message string
}

func NewAnalysisEngine(manager *analytics.AnalyticsManager) *AnalysisEngine {
	return &AnalysisEngine{Manager: manager}
}

func (ae *AnalysisEngine) GenerateInsights() ([]Insight, error) {
	var insights []Insight

	freqs, err := ae.Manager.GetCommandFrequencies()
	if err != nil {
		return nil, err
	}

	// Alias Suggestions
	for cmd, count := range freqs {
		if len(cmd) > 10 && count > 5 {
			// Suggest alias
			// Simple heuristic: take first letters? or just generic "alias this"
			insights = append(insights, Insight{
				Type:    "alias",
				Message: "You type '" + cmd + "' often (" + countString(count) + " times). Consider aliasing it!",
			})
		}
	}

	// Mistake Detection / Corrections
	// We need recent entries to detect "typo -> correction" patterns
	entries, err := ae.Manager.GetRecentEntries(50) // look at last 50
	if err == nil {
		for i := 0; i < len(entries)-1; i++ {
			// entries are desc (newest first). so i is newer than i+1
			curr := entries[i].Actual
			prev := entries[i+1].Actual

			// If very similar but different
			if curr != prev && len(curr) > 3 && len(prev) > 3 {
				dist := levenshtein(curr, prev)
				if dist > 0 && dist < 3 {
					// Likely a correction
					insights = append(insights, Insight{
						Type:    "mistake",
						Message: "Detected correction: '" + prev + "' -> '" + curr + "'.",
					})
				}
			}
		}
	}

	// Generic Tips
	tips := []string{
		"Did you know? `ctrl+r` searches rich history!",
		"Use `ctrl+a` to go to the start of the line.",
		"Use `ctrl+e` to go to the end of the line.",
		"Type `@?` to ask the AI to fix your last error.",
	}
	// Add a random tip
	// deterministic "random" based on something? or just first one not yet shown?
	// For now, just add all tips, UI can pick one.
	for _, t := range tips {
		insights = append(insights, Insight{Type: "tip", Message: t})
	}

	return insights, nil
}

func countString(n int64) string {
	return strconv.Itoa(int(n))
}

// Levenshtein distance
func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 { return m }
	if m == 0 { return n }

	row := make([]int, n+1)
	for i := 0; i <= n; i++ { row[i] = i }

	for j := 1; j <= m; j++ {
		prev := row[0]
		row[0] = j
		for i := 1; i <= n; i++ {
			temp := row[i]
			cost := 1
			if r1[i-1] == r2[j-1] { cost = 0 }

			// min(row[i-1]+1, row[i]+1, prev+cost)
			// del, ins, sub
			val := row[i-1] + 1
			if row[i]+1 < val { val = row[i] + 1 }
			if prev+cost < val { val = prev + cost }

			row[i] = val
			prev = temp
		}
	}
	return row[n]
}

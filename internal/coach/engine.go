package coach

import (
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/analytics"
)

// Engine handles the gamification logic: XP, Levels, Titles, Achievements
type Engine struct {
	Manager *analytics.AnalyticsManager
}

type UserStats struct {
	TotalXP      int
	Level        int
	Title        string
	NextLevelXP  int
	Progress     float64 // 0.0 to 1.0
	TotalCommands int
	UniqueCommands int
	Streak       int
	Achievements []Achievement
}

type Achievement struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Unlocked    bool
}

var allAchievements = []Achievement{
	{ID: "pipe_dream", Name: "Pipe Dream", Description: "Used a command with 3+ pipes", Icon: "🔧"},
	{ID: "git_guru", Name: "Git Guru", Description: "Used 10 unique git subcommands", Icon: "🐙"},
	{ID: "night_owl", Name: "Night Owl", Description: "Used the shell between 2 AM and 5 AM", Icon: "🦉"},
	{ID: "sudoer", Name: "Sudoer", Description: "Successfully used sudo", Icon: "🛡️"},
	{ID: "script_kiddie", Name: "Hello World", Description: " ran 10 commands", Icon: "👶"},
	{ID: "regular", Name: "Regular", Description: "ran 100 commands", Icon: "🧑‍💻"},
	{ID: "veteran", Name: "Veteran", Description: "ran 1000 commands", Icon: "🧙"},
}

func NewEngine(manager *analytics.AnalyticsManager) *Engine {
	return &Engine{Manager: manager}
}

func (e *Engine) CalculateStats() (*UserStats, error) {
	entries, err := e.Manager.GetAllEntries()
	if err != nil {
		return nil, err
	}

	stats := &UserStats{
		Achievements: make([]Achievement, len(allAchievements)),
	}
	copy(stats.Achievements, allAchievements)

	uniqueCmds := make(map[string]bool)
	uniqueGit := make(map[string]bool)

	// Track dates for streak
	activityDates := make(map[string]bool)

	for _, entry := range entries {
		cmd := strings.TrimSpace(entry.Actual)
		if cmd == "" {
			continue
		}
		stats.TotalCommands++

		// XP Calculation
		xp := 10 // Base XP
		xp += strings.Count(cmd, "|") * 5
		xp += strings.Count(cmd, "&&") * 5
		xp += strings.Count(cmd, "||") * 5

		// Flag Bonus
		for _, part := range strings.Fields(cmd) {
			if strings.HasPrefix(part, "-") && len(part) > 1 {
				xp += 2
			}
		}

		// Diversity bonus check (simplified: strictly if seen first time ever)
		// Ideally we'd persist "seen commands" to avoid re-awarding on restart if we didn't process all history every time.
		// Since we process all history here, it works.
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			prog := parts[0]
			if !uniqueCmds[prog] {
				xp += 50
				uniqueCmds[prog] = true
			}

			// Achievement Check: Git Guru
			if prog == "git" && len(parts) > 1 {
				uniqueGit[parts[1]] = true
			}
			// Achievement Check: Sudoer
			if prog == "sudo" {
				e.unlockAchievement(stats, "sudoer")
			}
		}

		// Achievement Check: Pipe Dream
		if strings.Count(cmd, "|") >= 3 {
			e.unlockAchievement(stats, "pipe_dream")
		}

		// Achievement Check: Night Owl
		h := entry.CreatedAt.Hour()
		if h >= 2 && h < 5 {
			e.unlockAchievement(stats, "night_owl")
		}

		stats.TotalXP += xp

		dateStr := entry.CreatedAt.Format("2006-01-02")
		activityDates[dateStr] = true
	}

	stats.UniqueCommands = len(uniqueCmds)

	if stats.TotalCommands >= 10 {
		e.unlockAchievement(stats, "script_kiddie")
	}
	if stats.TotalCommands >= 100 {
		e.unlockAchievement(stats, "regular")
	}
	if stats.TotalCommands >= 1000 {
		e.unlockAchievement(stats, "veteran")
	}
	if len(uniqueGit) >= 10 {
		e.unlockAchievement(stats, "git_guru")
	}

	// Calculate Level
	// Level = floor(sqrt(TotalXP / 100))
	// Level 1: 100 XP
	// Level 2: 400 XP
	// Level 3: 900 XP
	level := 0
	if stats.TotalXP > 0 {
		// simple sqrt approximation
		// for n such that n*n <= TotalXP/100
		val := stats.TotalXP / 100
		r := 0
		for (r+1)*(r+1) <= val {
			r++
		}
		level = r
	}
	if level < 1 { level = 1 }
	stats.Level = level

	// Next Level progress
	currentLevelBase := level * level * 100
	nextLevelBase := (level + 1) * (level + 1) * 100
	needed := nextLevelBase - currentLevelBase
	earned := stats.TotalXP - currentLevelBase

	if needed > 0 {
		stats.Progress = float64(earned) / float64(needed)
	} else {
		stats.Progress = 1.0
	}
	stats.NextLevelXP = nextLevelBase

	// Title
	stats.Title = e.getTitle(level)

	// Streak Calculation (simplified: iterate backwards from today)
	// This assumes the shell runs with correct system time
	// We need 'now' - but for deterministic tests we might rely on last entry?
	// Let's use system time for 'today'.
	now := time.Now()
	streak := 0
	for {
		d := now.AddDate(0, 0, -streak)
		ds := d.Format("2006-01-02")
		if activityDates[ds] {
			streak++
		} else {
			// If today has no activity yet, don't break streak if yesterday had activity?
			// Typically streaks include today. If today is empty, streak is effectively from yesterday?
			// Let's say if today is empty, we check yesterday.
			if streak == 0 {
				// check yesterday
				prev := now.AddDate(0, 0, -1)
				if activityDates[prev.Format("2006-01-02")] {
					// Streak continues from yesterday, but is currently 0 for "today" if we count strictly?
					// Usually apps say "1 day streak" if you did it yesterday.
					// Let's count consecutive days including today.
					// If today is missing, we check if yesterday is present.
					// If yesterday is present, streak is at least 1 (yesterday).
					// If today is present, streak starts at 1.

					// Let's just iterate back.
					streak = 0
					// Check today
					if !activityDates[now.Format("2006-01-02")] {
						// Today not done. Check yesterday.
						now = now.AddDate(0, 0, -1)
					}
					// Now iterate back
					for {
						check := now.AddDate(0, 0, -streak)
						if activityDates[check.Format("2006-01-02")] {
							streak++
						} else {
							break
						}
					}
					break
				} else {
					break
				}
			}
			break
		}
	}
	stats.Streak = streak

	return stats, nil
}

func (e *Engine) unlockAchievement(stats *UserStats, id string) {
	for i := range stats.Achievements {
		if stats.Achievements[i].ID == id {
			stats.Achievements[i].Unlocked = true
			return
		}
	}
}

func (e *Engine) getTitle(level int) string {
	switch {
	case level < 5:
		return "Script Kiddie"
	case level < 10:
		return "Bash Apprentice"
	case level < 20:
		return "Shell Wizard"
	case level < 50:
		return "Terminal Titan"
	default:
		return "Root Omnipotent"
	}
}

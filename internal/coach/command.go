package coach

import (
	"context"
	"fmt"

	"github.com/atinylittleshell/gsh/internal/analytics"
	tea "github.com/charmbracelet/bubbletea"
	"mvdan.cc/sh/v3/interp"
)

func NewCoachCommandHandler(analyticsManager *analytics.AnalyticsManager) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			if args[0] != "coach" {
				return next(ctx, args)
			}

			// Run the UI
			p := tea.NewProgram(NewModel(analyticsManager), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running coach: %v", err)
			}

			return nil
		}
	}
}

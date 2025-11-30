package config

import (
	"context"
	"fmt"

	"mvdan.cc/sh/v3/interp"
)

// Global runner reference
var globalRunner *interp.Runner

// SetConfigRunner sets the global runner reference for the config command handler
func SetConfigRunner(runner *interp.Runner) {
	globalRunner = runner
}

// NewConfigCommandHandler creates a new ExecHandler for the config command
func NewConfigCommandHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Handle "config" and "gsh config"
			// The caller (bash.SetBuiltinHandler) might pass args starting with "config"
			// But if it's "gsh config", it might be different.
			// Let's assume this handles the built-in "config" command.
			if args[0] != "config" && args[0] != "gsh_config" {
				return next(ctx, args)
			}

			if globalRunner == nil {
				return fmt.Errorf("config: runner not initialized")
			}

			return runConfigUI(globalRunner)
		}
	}
}

package evaluate

import (
	"context"
	"testing"

	"github.com/robottwo/bishop/internal/analytics"
	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
)

func TestEvaluateCommandHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		shouldCallNext bool
		expectError    bool
	}{
		{
			name:           "empty args should call next handler",
			args:           []string{},
			shouldCallNext: true,
			expectError:    false,
		},
		{
			name:           "non-gsh_evaluate command should call next handler",
			args:           []string{"some_other_command"},
			shouldCallNext: true,
			expectError:    false,
		},
		{
			name:           "help flag should print help",
			args:           []string{"gsh_evaluate", "--help"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "help short flag should print help",
			args:           []string{"gsh_evaluate", "-h"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "default values when no args provided",
			args:           []string{"gsh_evaluate", "--limit", "3"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom limit",
			args:           []string{"gsh_evaluate", "--limit", "2"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom limit short flag",
			args:           []string{"gsh_evaluate", "-l", "2"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom model",
			args:           []string{"gsh_evaluate", "--limit", "3", "--model", "gpt-4"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom model short flag",
			args:           []string{"gsh_evaluate", "-l", "3", "-m", "gpt-4"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom iterations",
			args:           []string{"gsh_evaluate", "-l", "3", "--iterations", "2"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "custom iterations short flag",
			args:           []string{"gsh_evaluate", "-l", "3", "-i", "2"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "both custom limit and model",
			args:           []string{"gsh_evaluate", "-l", "2", "-m", "gpt-4"},
			shouldCallNext: false,
			expectError:    false,
		},
		{
			name:           "error when limit exceeds available entries",
			args:           []string{"gsh_evaluate", "--limit", "10"},
			shouldCallNext: false,
			expectError:    true,
		},
		{
			name:           "error when limit exceeds available entries with custom model",
			args:           []string{"gsh_evaluate", "--limit", "10", "--model", "gpt-4"},
			shouldCallNext: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new in-memory analytics manager for each test
			analyticsManager, err := analytics.NewAnalyticsManager(":memory:")
			assert.NoError(t, err)

			// Set up a shell runner
			runner, err := interp.New()
			assert.NoError(t, err)
			analyticsManager.Runner = runner

			// Create a few sample entries for testing
			commands := []string{
				"ls -la",
				"cd /tmp",
				"git status",
			}
			for _, cmd := range commands {
				err = analyticsManager.NewEntry(cmd, cmd, cmd)
				assert.NoError(t, err)
			}

			// Create a next handler that tracks if it was called
			nextHandlerCalled := false
			nextHandler := func(ctx context.Context, args []string) error {
				nextHandlerCalled = true
				return nil
			}

			// Create the handler
			handler := NewEvaluateCommandHandler(analyticsManager)

			// Execute the handler
			err = handler(nextHandler)(context.Background(), tt.args)

			// Verify expectations
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not enough entries to evaluate")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.shouldCallNext, nextHandlerCalled,
				"Next handler called state doesn't match expected for test case: %s", tt.name)
		})
	}
}

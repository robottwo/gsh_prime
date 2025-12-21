package environment

import (
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// bishVariableNames contains the list of BISH-specific environment variables
// This variable is used to maintain consistency across the codebase
var bishVariableNames = []string{
	"BISH_PROMPT", "BISH_APROMPT", "BISH_BUILD_VERSION", "BISH_LOG_LEVEL",
	"BISH_CLEAN_LOG_FILE", "BISH_MINIMUM_HEIGHT", "BISH_ASSISTANT_HEIGHT",
	"BISH_AGENT_NAME", "BISH_FAST_MODEL_API_KEY", "BISH_FAST_MODEL_BASE_URL",
	"BISH_FAST_MODEL_ID", "BISH_SLOW_MODEL_API_KEY", "BISH_SLOW_MODEL_BASE_URL",
	"BISH_SLOW_MODEL_ID", "BISH_AGENT_CONTEXT_WINDOW_TOKENS", "BISH_PAST_COMMANDS_CONTEXT_LIMIT",
	"BISH_CONTEXT_TYPES_FOR_AGENT", "BISH_CONTEXT_TYPES_FOR_PREDICTION_WITH_PREFIX",
	"BISH_CONTEXT_TYPES_FOR_PREDICTION_WITHOUT_PREFIX", "BISH_CONTEXT_TYPES_FOR_EXPLANATION",
	"BISH_CONTEXT_NUM_HISTORY_CONCISE", "BISH_CONTEXT_NUM_HISTORY_VERBOSE",
	"BISH_AGENT_APPROVED_BASH_COMMAND_REGEX", "BISH_AGENT_MACROS", "BISH_DEFAULT_TO_YES",
}

// DynamicEnviron implements expand.Environ to provide a dynamic environment
// that includes both system environment variables and BISH-specific variables
type DynamicEnviron struct {
	systemEnv expand.Environ
	bishVars  map[string]string
}

// NewDynamicEnviron creates a new DynamicEnviron that wraps the system environment
// and adds BISH-specific variables
func NewDynamicEnviron() *DynamicEnviron {
	return &DynamicEnviron{
		systemEnv: expand.ListEnviron(os.Environ()...),
		bishVars:  make(map[string]string),
	}
}

// Get retrieves a variable by name, checking GSH variables first, then system environment
func (de *DynamicEnviron) Get(name string) expand.Variable {
	// Check GSH variables first
	if value, exists := de.bishVars[name]; exists {
		return expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}
	}

	// Fall back to system environment
	return de.systemEnv.Get(name)
}

// Each iterates over all variables, including both GSH and system variables
func (de *DynamicEnviron) Each(fn func(name string, vr expand.Variable) bool) {
	// First, iterate over GSH variables
	for name, value := range de.bishVars {
		if !fn(name, expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}) {
			return
		}
	}

	// Then iterate over system environment, skipping GSH variables we already added
	de.systemEnv.Each(func(name string, vr expand.Variable) bool {
		if _, isGSH := de.bishVars[name]; !isGSH {
			return fn(name, vr)
		}
		return true
	})
}

// UpdateBishVar updates a BISH variable in the dynamic environment
func (de *DynamicEnviron) UpdateBishVar(name, value string) {
	de.bishVars[name] = value
}

// UpdateSystemEnv updates the system environment wrapper
func (de *DynamicEnviron) UpdateSystemEnv() {
	de.systemEnv = expand.ListEnviron(os.Environ()...)
}

// SyncVariablesToEnv syncs bish's internal variables to system environment variables
// This makes variables like BISH_PROMPT visible to external commands like 'env'
func SyncVariablesToEnv(runner *interp.Runner) {
	// Check if we already have a DynamicEnviron, if not create one
	var dynamicEnv *DynamicEnviron
	if existingDynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
		dynamicEnv = existingDynamicEnv
	} else {
		dynamicEnv = NewDynamicEnviron()
	}

	for _, varName := range bishVariableNames {
		if varValue, exists := runner.Vars[varName]; exists {
			value := varValue.String()
			if err := os.Setenv(varName, value); err != nil {
				return
			}
			dynamicEnv.UpdateBishVar(varName, value)
			continue
		}

		_ = os.Unsetenv(varName)
		delete(dynamicEnv.bishVars, varName)
	}

	// Update the system environment in the dynamic environment
	dynamicEnv.UpdateSystemEnv()

	// Set the runner's environment to our dynamic environment
	runner.Env = dynamicEnv
}

// SyncVariableToEnv syncs a single bish variable to system environment
func SyncVariableToEnv(runner *interp.Runner, varName string) {
	if varValue, exists := runner.Vars[varName]; exists {
		value := varValue.String()
		if err := os.Setenv(varName, value); err != nil {
			return
		}

		// Update in the dynamic environment
		if dynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
			dynamicEnv.UpdateBishVar(varName, value)
		}
		return
	}

	if err := os.Unsetenv(varName); err != nil {
		return
	}
	if dynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
		delete(dynamicEnv.bishVars, varName)
	}
}

// IsBishVariable checks if a variable name is a bish-specific variable that should be synced
func IsBishVariable(name string) bool {
	for _, bishVar := range bishVariableNames {
		if name == bishVar {
			return true
		}
	}
	return strings.HasPrefix(name, "BISH_")
}

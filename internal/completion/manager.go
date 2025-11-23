package completion

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// CompletionType represents the type of completion
type CompletionType string

const (
	// WordListCompletion represents word list based completion (-W option)
	WordListCompletion CompletionType = "W"
	// FunctionCompletion represents function based completion (-F option)
	FunctionCompletion CompletionType = "F"
	// CommandCompletion represents command based completion (-C option)
	CommandCompletion CompletionType = "C"
)

// CompletionSpec represents a completion specification for a command
type CompletionSpec struct {
	Command string
	Type    CompletionType
	Value   string   // function name, wordlist, or command
	Options []string // additional options like -o dirname
}

// CompletionManager manages command completion specifications
type CompletionManager struct {
	specs map[string]CompletionSpec
}

// NewCompletionManager creates a new CompletionManager
func NewCompletionManager() *CompletionManager {
	return &CompletionManager{
		specs: make(map[string]CompletionSpec),
	}
}

// AddSpec adds or updates a completion specification
func (m *CompletionManager) AddSpec(spec CompletionSpec) {
	m.specs[spec.Command] = spec
}

// RemoveSpec removes a completion specification
func (m *CompletionManager) RemoveSpec(command string) {
	delete(m.specs, command)
}

// GetSpec retrieves a completion specification
func (m *CompletionManager) GetSpec(command string) (CompletionSpec, bool) {
	spec, ok := m.specs[command]
	return spec, ok
}

// ListSpecs returns all completion specifications
func (m *CompletionManager) ListSpecs() []CompletionSpec {
	specs := make([]CompletionSpec, 0, len(m.specs))
	for _, spec := range m.specs {
		specs = append(specs, spec)
	}
	return specs
}

// ExecuteCompletion executes a completion specification for a given command line
// and returns the list of possible completions
func (m *CompletionManager) ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string, line string, pos int) ([]string, error) {
	switch spec.Type {
	case WordListCompletion:
		words := strings.Fields(spec.Value)
		completions := make([]string, 0)
		word := ""
		if len(args) > 0 {
			word = args[len(args)-1]
		}
		for _, w := range words {
			if word == "" || strings.HasPrefix(w, word) {
				completions = append(completions, w)
			}
		}
		return completions, nil

	case FunctionCompletion:
		fn := NewCompletionFunction(spec.Value, runner)
		return fn.Execute(ctx, args)

	case CommandCompletion:
		return m.RunExternalCompleter(ctx, spec.Value, args, line, pos)

	default:
		return nil, fmt.Errorf("unsupported completion type: %s", spec.Type)
	}
}

// RunExternalCompleter executes an external command to generate completions
func (m *CompletionManager) RunExternalCompleter(ctx context.Context, command string, args []string, line string, pos int) ([]string, error) {
	// Prepare arguments for the external command
	// $1 is the command name being completed
	// $2 is the word being completed
	// $3 is the word preceding that
	var arg1, arg2, arg3 string
	if len(args) > 0 {
		arg1 = args[0]
		arg2 = args[len(args)-1]
		if len(args) > 1 {
			arg3 = args[len(args)-2]
		}
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("COMP_LINE=%s", line))
	env = append(env, fmt.Sprintf("COMP_POINT=%d", pos))
	env = append(env, "COMP_KEY=9")  // 9 is TAB
	env = append(env, "COMP_TYPE=9") // 9 is TAB

	// Reconstruct COMP_WORDS array string if needed, but usually bash/external commands
	// rely on arguments or parse COMP_LINE themselves.
	// Bash export format for arrays is tricky, but often simple commands just use args.
	// For compatibility, we'll just set these standard ones.

	// Run the command
	// The command string might contain arguments itself, so we should probably run it via sh -c
	// or parse it. Bash 'complete -C command' usually runs 'command' directly if it's a path,
	// or looks it up in PATH. It passes the 3 args.

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%s \"$@\"", command), "--", arg1, arg2, arg3)
	cmd.Env = env

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// If command fails, we return empty completions rather than error,
		// to allow fallback or just show nothing.
		return []string{}, nil
	}

	// Parse output: one completion per line
	output := out.String()
	lines := strings.Split(output, "\n")
	completions := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			completions = append(completions, l)
		}
	}

	return completions, nil
}

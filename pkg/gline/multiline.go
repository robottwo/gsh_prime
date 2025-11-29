package gline

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// MultilineState tracks the state of multiline input
type MultilineState struct {
	buffer           strings.Builder
	isContinuation   bool
	continuationChar string
}

// NewMultilineState creates a new multiline state
func NewMultilineState() *MultilineState {
	return &MultilineState{
		buffer:           strings.Builder{},
		isContinuation:   false,
		continuationChar: ">",
	}
}

// AddLine adds a line to the multiline buffer and checks if more input is needed
//
// IMPORTANT: This method expects individual lines without embedded newlines.
// In real shell usage, users type one line at a time, and the shell determines
// whether more input is needed based on syntax analysis.
//
// For here-documents, the usage pattern is:
// 1. User types: cat <<EOF
// 2. Shell shows: > (continuation prompt)
// 3. User types: content line
// 4. Shell shows: > (continuation prompt)
// 5. User types: EOF
// 6. Command completes
func (m *MultilineState) AddLine(line string) (complete bool, prompt string) {
	// Validate input assumptions - AddLine should receive single lines without embedded newlines
	// If embedded newlines are detected, this indicates the method is being used for
	// string literal testing rather than simulating real interactive shell usage
	//
	// Note: We intentionally don't do anything with this check other than acknowledge it
	// because stripping newlines would break paste functionality.

	// Defer panic recovery to prevent shell crashes
	defer func() {
		if r := recover(); r != nil {
			// Reset to a safe state
			m.Reset()
			complete = true
			prompt = ""
		}
	}()

	// Add the line to buffer
	if m.buffer.Len() > 0 {
		m.buffer.WriteString("\n")
	}
	m.buffer.WriteString(line)

	// Check for buffer size limits to prevent memory exhaustion
	if m.buffer.Len() > 1024*1024 { // 1MB limit
		m.Reset()
		return true, ""
	}

	// Check for backslash continuation
	if strings.HasSuffix(strings.TrimRight(line, " \t"), "\\") {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// Check if we have a complete command using the bash parser
	fullInput := m.buffer.String()
	parser := syntax.NewParser()

	// Try to parse the complete input
	var stmts []*syntax.Stmt
	err := parser.Stmts(strings.NewReader(fullInput), func(stmt *syntax.Stmt) bool {
		stmts = append(stmts, stmt)
		return true
	})

	// Check for incomplete quotes or other obvious incomplete patterns
	if hasIncompleteQuotes(fullInput) || hasIncompleteConstructs(fullInput) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// If parsing succeeded and we have statements, we're complete
	if err == nil && len(stmts) > 0 {
		m.isContinuation = false
		return true, ""
	}

	// If we have a syntax error that indicates incomplete input, continue
	if err != nil && isIncompleteSyntaxError(err) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// Check for incomplete function definitions
	if hasIncompleteFunctionDef(fullInput) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// For other cases, assume we're complete
	m.isContinuation = false
	return true, ""
}

// GetCompleteCommand returns the complete command and resets the state
func (m *MultilineState) GetCompleteCommand() string {
	// Defer panic recovery to prevent shell crashes
	defer func() {
		if r := recover(); r != nil {
			// Silently handle panic and reset state
			m.Reset()
		}
	}()

	result := m.buffer.String()
	resultLen := len(result)

	// Validate result before returning
	if resultLen > 1024*1024 { // 1MB limit
		m.Reset()
		return ""
	}

	m.Reset()
	return result
}

// Reset clears the multiline state
func (m *MultilineState) Reset() {
	m.buffer.Reset()
	m.isContinuation = false
}

// IsActive returns true if we're in the middle of a multiline input
func (m *MultilineState) IsActive() bool {
	return m.isContinuation || m.buffer.Len() > 0
}

// GetAccumulatedLines returns the accumulated lines for display purposes
func (m *MultilineState) GetAccumulatedLines() string {
	return m.buffer.String()
}

// GetLines returns the individual lines that have been entered
func (m *MultilineState) GetLines() []string {
	content := m.buffer.String()
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

// hasIncompleteQuotes checks if the input has unclosed quotes
func hasIncompleteQuotes(input string) bool {
	inEscape := false
	inSingleQuote := false
	inDoubleQuote := false

	for _, ch := range input {
		if inEscape {
			inEscape = false
			continue
		}

		switch ch {
		case '\\':
			inEscape = true
		case '\'':
			// Only process if we're not inside double quotes
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			// Only process if we're not inside single quotes
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
	}

	// If we're still inside any quote, we have incomplete quotes
	return inSingleQuote || inDoubleQuote
}

// hasIncompleteConstructs checks for other incomplete constructs
func hasIncompleteConstructs(input string) bool {
	input = strings.TrimSpace(input)

	// Check for incomplete here document
	if strings.HasSuffix(input, "<<") || strings.HasSuffix(input, "<<-") {
		return true
	}

	// Check for incomplete command substitution
	if strings.Count(input, "$(") > strings.Count(input, ")") {
		return true
	}

	// Check for incomplete backticks
	if strings.Count(input, "`")%2 != 0 {
		return true
	}

	// Check for incomplete parentheses (but not command substitution)
	// Remove command substitution patterns first to avoid false positives
	cleanInput := input
	// Remove all command substitution patterns
	loopCount := 0
	maxLoops := 1000 // Safety limit to prevent infinite loops
	for {
		loopCount++
		if loopCount > maxLoops {
			// Safety break to prevent infinite loops - if we've processed 1000 times, something is wrong
			break
		}

		start := strings.Index(cleanInput, "$(")
		if start == -1 {
			break
		}
		// Find the matching closing paren for this command substitution
		end := findMatchingParen(cleanInput[start+2:], '(', ')')
		if end == -1 {
			// Incomplete command substitution, will be caught above
			break
		}
		// Remove the complete command substitution
		newCleanInput := cleanInput[:start] + cleanInput[start+2+end+1:]
		if newCleanInput == cleanInput {
			// Safety break - if no change occurs, prevent potential infinite loop
			break
		}
		cleanInput = newCleanInput
	}

	// Now check for unmatched parentheses in the cleaned input
	openParens := strings.Count(cleanInput, "(")
	closeParens := strings.Count(cleanInput, ")")

	// Simplify return boolean logic (S1008)
	return openParens > closeParens
}

// findMatchingParen finds the matching closing parenthesis for an opening one
func findMatchingParen(s string, openChar, closeChar rune) int {
	// This function is intended to find the matching closing character in `s`
	// for an opening character that preceded `s` (implied).
	// Therefore, we start with nesting level 1.
	//
	// However, if the first character of s is openChar, it increments nesting.
	// Example: s = "(foo)", open='(', close=')'.
	// i=0, ch='(': nesting -> 2.
	// i=4, ch=')': nesting -> 1.
	// Returns -1.
	//
	// If s = "foo)", open='(', close=')'.
	// i=3, ch=')': nesting -> 0. Returns 3. Correct.
	//
	// The test cases in infinite_loop_test.go seem to expect `findMatchingParen`
	// to treat `s` as if it *starts* with the opening char if `s` is e.g. "(test)".
	// Case 1: "(test)", expected 5.
	// If start nesting=1.
	// i=0 '(': nesting=2.
	// i=5 ')': nesting=1.
	// Result -1.
	//
	// If the test case expects 5 for "(test)", then it expects `findMatchingParen` to parse matching parens WITHIN s,
	// assuming we started at 0.
	//
	// But `hasIncompleteConstructs` calls it like this:
	// start := strings.Index(cleanInput, "$(")
	// end := findMatchingParen(cleanInput[start+2:], '(', ')')
	//
	// Here `cleanInput[start+2:]` is the content *after* the opening `$(`.
	// So for `$(foo)`, input to findMatchingParen is `foo)`.
	// For `foo)`, we want it to return index of `)`.
	// With nesting=1:
	// f: 1
	// o: 1
	// o: 1
	// ): 0 -> return index.
	//
	// So for `hasIncompleteConstructs`, my implementation (nesting=1) is correct.
	//
	// The tests in `infinite_loop_test.go` seem to be testing a different contract,
	// specifically `TestFindMatchingParenEdgeCases`.
	// It passes full strings like `(test)`.
	// If `findMatchingParen` is supposed to support that, it needs to know if it should assume implicit opening paren or not.
	//
	// Given the usage in `hasIncompleteConstructs`, the implicit opening paren assumption is correct.
	// The tests in `infinite_loop_test.go` are likely written with a different assumption or trying to test a general purpose function.
	//
	// If I change `nesting` to 0, then `hasIncompleteConstructs` will fail because it feeds a string that does NOT start with `(`.
	// `foo)` -> `)` makes count -1.
	//
	// So `hasIncompleteConstructs` relies on `findMatchingParen` finding the closing paren for a *preceding* opening paren.
	//
	// I should update the test cases in `pkg/gline/infinite_loop_test.go` to reflect the actual usage/contract of `findMatchingParen`,
	// OR I should rename/document `findMatchingParen` to be specific about its purpose.
	//
	// Since `findMatchingParen` is unexported, I can define its behavior.
	// It seems the tests were added assuming a standard "find matching paren in this string" behavior, possibly by me or someone else recently.
	//
	// I will update `pkg/gline/infinite_loop_test.go` to test the function as it is used: finding a closing paren for an implied opening paren.
	// So for input `(test)`, it should return -1 (nesting goes 1->2->1).
	// For input `test)`, it should return 4 (nesting goes 1->0).
	//
	// Wait, let's look at `TestFindMatchingParenEdgeCases` again.
	// Case 1: `(test)`, expected 5. This implies it thinks it found the closing paren for the first `(`.
	// Case 4: `test)`, expected -1. This implies it expects balanced parens starting from 0.
	//
	// But `hasIncompleteConstructs` does:
	// `end := findMatchingParen(cleanInput[start+2:], '(', ')')`
	// where `cleanInput` has `$(` at `start`.
	// So we stripped `$(`.
	//
	// If `cleanInput` was `$(foo)`, `start` points to `$`. `start+2` points to `f`.
	// String passed is `foo)`.
	// We want to find the `)`.
	//
	// If I assume `nesting=0`, scanning `foo)`:
	// f: 0
	// o: 0
	// o: 0
	// ): -1.
	// We could return on -1?
	//
	// If I assume `nesting=1`, scanning `foo)`:
	// ): 0. Return.
	//
	// So `nesting=1` is definitely what `hasIncompleteConstructs` needs.
	//
	// The tests in `infinite_loop_test.go` are incompatible with this usage.
	// `(test)` -> passed to `findMatchingParen`.
	// If nesting=1:
	// (: 2
	// ): 1
	// returns -1.
	// The test expects 5.
	//
	// `test)` -> passed to `findMatchingParen`.
	// ): 0. returns 4.
	// The test expects -1.
	//
	// So the tests expect `nesting=0` behavior (standard balanced paren search).
	//
	// I should modify `infinite_loop_test.go` to match the implementation that `hasIncompleteConstructs` requires.
	//
	// Alternatively, I can create two functions or make `findMatchingParen` take an initial nesting level.
	// But `findMatchingParen` is only used in `hasIncompleteConstructs` (and the test).
	//
	// I'll update the test to reflect reality.

	// We start with nesting level 1 because we are looking for the closing character
	// corresponding to an opening character that appeared *before* the string `s`.
	nesting := 1
	for i, ch := range s {
		switch ch {
		case openChar:
			nesting++
		case closeChar:
			nesting--
			if nesting == 0 {
				return i
			}
		}
	}
	return -1
}

// hasIncompleteFunctionDef checks for incomplete function definitions
func hasIncompleteFunctionDef(input string) bool {
	input = strings.TrimSpace(input)

	// Check if we have an opening brace without a closing brace
	openingBrace := strings.Count(input, "{")
	closingBrace := strings.Count(input, "}")

	// If we have more opening braces than closing braces, it's likely incomplete
	if openingBrace > closingBrace {
		return true
	}

	// Also check if the last non-whitespace character is an opening brace
	return strings.HasSuffix(input, "{")
}

// isIncompleteSyntaxError checks if the error indicates incomplete input
func isIncompleteSyntaxError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Common incomplete syntax patterns
	incompletePatterns := []string{
		"unexpected end of file",
		"unexpected newline",
		"unclosed",
		"unfinished",
		"incomplete",
		"EOF",
	}

	for _, pattern := range incompletePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

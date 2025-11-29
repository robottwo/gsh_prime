package gline

import (
	"fmt"
	"testing"
	"time"
)

// TestCommandSubstitutionInfiniteLoop tests for the infinite loop scenario mentioned in the bot's comment
func TestCommandSubstitutionInfiniteLoop(t *testing.T) {
	fmt.Println("Testing command substitution infinite loop scenarios...")

	testCases := []struct {
		input   string
		desc    string
		timeout time.Duration
	}{
		{"$(echo (test", "Malformed command substitution with nested parentheses", 2 * time.Second},
		{"$(echo (test (nested)", "Multiple levels of incomplete nesting", 2 * time.Second},
		{"$(echo test", "Simple incomplete command substitution", 2 * time.Second},
		{"echo $(foo $(bar", "Nested incomplete command substitution", 2 * time.Second},
		{"$(", "Minimal incomplete command substitution", 2 * time.Second},
		{"echo $(foo $(bar $(baz)", "Deeply nested incomplete", 2 * time.Second},
		{"echo $(foo) $(bar $(baz", "Mixed complete and incomplete", 2 * time.Second},
		{"$(echo $(echo $(echo", "Multiple levels", 2 * time.Second},
		{"$(echo test)))", "Extra closing parentheses", 2 * time.Second},
		{"$(echo test (more) (parens) (extra))", "Multiple parentheses groups", 2 * time.Second},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case_%d_%s", i+1, tc.desc), func(t *testing.T) {
			fmt.Printf("Testing case %d: %q - %s\n", i+1, tc.input, tc.desc)

			state := NewMultilineState()

			// Use a channel with timeout to detect potential infinite loops
			done := make(chan struct{})
			var complete bool
			var prompt string

			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("  PANIC recovered: %v\n", r)
					}
					close(done)
				}()

				start := time.Now()
				complete, prompt = state.AddLine(tc.input)
				elapsed := time.Since(start)

				fmt.Printf("  Result: complete=%v, prompt=%q, time=%v\n", complete, prompt, elapsed)
			}()

			select {
			case <-done:
				fmt.Printf("  ✓ Completed normally\n")
			case <-time.After(tc.timeout):
				t.Errorf("  TIMEOUT after %v - Potential infinite loop detected for input: %q", tc.timeout, tc.input)
				fmt.Printf("  ⚠️  TIMEOUT - Potential infinite loop detected!\n")
			}
		})
	}
}

// BenchmarkCommandSubstitutionRemoval benchmarks the command substitution removal logic
func BenchmarkCommandSubstitutionRemoval(b *testing.B) {
	testCases := []string{
		"$(echo (test))",
		"$(echo $(date) (test))",
		"echo $(foo $(bar $(baz)))",
		"$(echo test (more) (parens) (extra))",
	}

	for i, tc := range testCases {
		b.Run(fmt.Sprintf("Case_%d", i+1), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				state := NewMultilineState()
				state.AddLine(tc)
			}
		})
	}
}

// TestFindMatchingParenEdgeCases tests edge cases for the findMatchingParen function
func TestFindMatchingParenEdgeCases(t *testing.T) {
	testCases := []struct {
		input     string
		openChar  rune
		closeChar rune
		expected  int
		desc      string
	}{
		// findMatchingParen assumes an implicit opening parenthesis BEFORE the string
		// So it's looking for the closing parenthesis that matches the implicit start

		// (test) - implicit start: ((test)
		// 0: ( -> nesting 2
		// 5: ) -> nesting 1
		// no match for implicit start
		{"(test)", '(', ')', -1, "Implicit start: ((test)"},

		// test) - implicit start: (test)
		// 4: ) -> nesting 0
		// match found at 4
		{"test)", '(', ')', 4, "Implicit start: (test)"},

		// test - implicit start: (test
		// no closing paren
		{"test", '(', ')', -1, "Implicit start: (test - no closing"},

		// (test - implicit start: ((test
		// no closing paren
		{"(test", '(', ')', -1, "Implicit start: ((test - no closing"},

		// ) - implicit start: ()
		// 0: ) -> nesting 0
		// match found at 0
		{")", '(', ')', 0, "Implicit start: ()"},

		// (test)) - implicit start: ((test))
		// 0: ( -> nesting 2
		// 5: ) -> nesting 1
		// 6: ) -> nesting 0
		// match found at 6
		{"(test))", '(', ')', 6, "Implicit start: ((test))"},

		// test (nested) ) - implicit start: (test (nested) )
		// 5: ( -> nesting 2
		// 12: ) -> nesting 1
		// 14: ) -> nesting 0
		// match found at 14
		{"test (nested) )", '(', ')', 14, "Implicit start: (test (nested) )"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case_%d_%s", i+1, tc.desc), func(t *testing.T) {
			result := findMatchingParen(tc.input, tc.openChar, tc.closeChar)
			if result != tc.expected {
				t.Errorf("findMatchingParen(%q, %c, %c) = %d, want %d",
					tc.input, tc.openChar, tc.closeChar, result, tc.expected)
			}
		})
	}
}

package shellinput

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Default theme for syntax highlighting
	defaultStyle = styles.Get("monokai")
	// Fallback lexer
	defaultLexer = lexers.Get("bash")
)

// Highlight returns a slice of strings, where each element is the ANSI-styled
// representation of the corresponding rune in the input.
func Highlight(input []rune) []string {
	if len(input) == 0 {
		return nil
	}

	inputText := string(input)
	iterator, err := defaultLexer.Tokenise(nil, inputText)
	if err != nil {
		// Fallback to unstyled text if lexing fails
		result := make([]string, len(input))
		for i, r := range input {
			result[i] = string(r)
		}
		return result
	}

	result := make([]string, 0, len(input))

	for _, token := range iterator.Tokens() {
		// Get style for this token
		styleEntry := defaultStyle.Get(token.Type)
		ansiStyle := styleEntryToLipgloss(styleEntry)

		// Apply style to each rune in the token
		for _, r := range token.Value {
			result = append(result, ansiStyle.Render(string(r)))
		}
	}

	// Safety check: if result length doesn't match input length (shouldn't happen with valid unicode),
	// pad or trim.
	if len(result) < len(input) {
		for i := len(result); i < len(input); i++ {
			result = append(result, string(input[i]))
		}
	} else if len(result) > len(input) {
		result = result[:len(input)]
	}

	return result
}

// styleEntryToLipgloss converts a chroma StyleEntry to a lipgloss Style
func styleEntryToLipgloss(entry chroma.StyleEntry) lipgloss.Style {
	style := lipgloss.NewStyle()

	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Background.IsSet() {
		style = style.Background(lipgloss.Color(entry.Background.String()))
	}
	if entry.Bold == chroma.Yes {
		style = style.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		style = style.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		style = style.Underline(true)
	}

	return style
}

package shellinput

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestViewSyncWithHistoryState(t *testing.T) {
	m := New()
	// Setup history items
	items := []HistoryItem{
		{Command: "command1", Timestamp: time.Now()},
		{Command: "command2", Timestamp: time.Now()},
		{Command: "command3", Timestamp: time.Now()},
	}
	m.SetRichHistory(items)

	// Activate reverse search
	m.inReverseSearch = true
	m.reverseSearchQuery = "comm"

	// Force update to populate filteredIndices
	m.updateHistorySearch()

	// Check View output
	viewOutput := m.View()

	// Expect the first item to be selected (command1)
	expectedMatch := items[0].Command
	assert.Contains(t, viewOutput, expectedMatch, "View should contain the selected history item")
	assert.Contains(t, viewOutput, "(reverse-i-search)", "View should be in reverse search mode")
	assert.Contains(t, viewOutput, "`comm':", "View should show the query")

	// Move selection down (next item)
	m.historySearchDown()

	// Check View output again
	viewOutput = m.View()
	expectedMatch = items[1].Command
	assert.Contains(t, viewOutput, expectedMatch, "View should update to show the new selected item")

	// Change query
	m.reverseSearchQuery = "command3"
	m.updateHistorySearch()

	viewOutput = m.View()
	expectedMatch = "command3"
	assert.Contains(t, viewOutput, expectedMatch)
}

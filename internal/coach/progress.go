package coach

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const lightning = "⚡"

// Color cycle for animation: blue → purple → orange → yellow → back
// Matches the colors in pkg/gline/llm_status.go
var animationColors = []lipgloss.Color{
	"12", "33", "57", "93", "129", "208", "214", "220",
	"214", "208", "129", "93", "57", "33",
}

// ProgressIndicator displays an animated lightning bolt with a message
type ProgressIndicator struct {
	message    string
	frameIndex int
	stopChan   chan struct{}
	doneChan   chan struct{}
	mu         sync.Mutex
	running    bool
}

// NewProgressIndicator creates a new progress indicator with the given message
func NewProgressIndicator(message string) *ProgressIndicator {
	return &ProgressIndicator{
		message:  message,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start begins the animation
func (p *ProgressIndicator) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	go func() {
		defer close(p.doneChan)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		// Initial render
		p.render()

		for {
			select {
			case <-p.stopChan:
				return
			case <-ticker.C:
				p.frameIndex = (p.frameIndex + 1) % len(animationColors)
				p.render()
			}
		}
	}()
}

// render displays the current frame
func (p *ProgressIndicator) render() {
	color := animationColors[p.frameIndex]
	bolt := lipgloss.NewStyle().Foreground(color).Render(lightning)
	// Use carriage return to overwrite the line, no newline
	fmt.Printf("\r  %s %s", p.message, bolt)
}

// Stop stops the animation and clears the line
func (p *ProgressIndicator) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopChan)
	<-p.doneChan

	// Clear the line and move cursor back
	fmt.Print("\r\033[K")
}

// StopWithMessage stops the animation and prints a final message
func (p *ProgressIndicator) StopWithMessage(message string) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		fmt.Println(message)
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopChan)
	<-p.doneChan

	// Clear the line and print the final message
	fmt.Print("\r\033[K")
	fmt.Println(message)
}

// UpdateMessage updates the message being displayed
func (p *ProgressIndicator) UpdateMessage(message string) {
	p.mu.Lock()
	p.message = message
	p.mu.Unlock()
}

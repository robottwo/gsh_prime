package gline

type Explainer interface {
	Explain(input string) (string, error)
}

type NoopExplainer struct{}

func (e *NoopExplainer) Explain(input string) (string, error) {
	return "", nil
}

// CoachTipProvider provides usage tips and insights for the assistant box
type CoachTipProvider interface {
	// GetQuickTip returns a brief coach tip for display in the assistant box
	GetQuickTip() string
}

// NoopCoachTipProvider is a no-op implementation of CoachTipProvider
type NoopCoachTipProvider struct{}

func (c *NoopCoachTipProvider) GetQuickTip() string {
	return ""
}

package predict

// NoopExplainer provides a deterministic, non-LLM explanation implementation
// that simply disables the feature while satisfying the interface.
type NoopExplainer struct{}

func NewNoopExplainer() *NoopExplainer { return &NoopExplainer{} }

func (p *NoopExplainer) UpdateContext(_ *map[string]string) {}

func (e *NoopExplainer) Explain(_ string) (string, error) { return "", nil }

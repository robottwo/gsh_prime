package predict

import (
	"strings"

	"github.com/atinylittleshell/gsh/internal/history"
	"go.uber.org/zap"
)

// HistoryPrefixPredictor provides deterministic predictions using recent history
// entries instead of querying an LLM. The most recent command matching the
// current prefix is suggested as the prediction.
type HistoryPrefixPredictor struct {
	historyManager *history.HistoryManager
	logger         *zap.Logger
}

func NewHistoryPrefixPredictor(
	historyManager *history.HistoryManager,
	logger *zap.Logger,
) *HistoryPrefixPredictor {
	return &HistoryPrefixPredictor{
		historyManager: historyManager,
		logger:         logger,
	}
}

func (p *HistoryPrefixPredictor) UpdateContext(_ *map[string]string) {}

func (p *HistoryPrefixPredictor) Predict(input string) (string, string, error) {
	if strings.HasPrefix(input, "#") {
		// Don't do prediction for agent chat messages
		return "", "", nil
	}

	if strings.TrimSpace(input) == "" {
		return "", "", nil
	}

	entries, err := p.historyManager.GetRecentEntriesByPrefix(input, 1)
	if err != nil {
		p.logger.Warn("error fetching history for prediction", zap.Error(err))
		return "", "", nil
	}

	if len(entries) == 0 {
		return "", "", nil
	}

	prediction := entries[0].Command

	p.logger.Debug(
		"history-based prediction",
		zap.String("input", input),
		zap.String("prediction", prediction),
	)

	return prediction, input, nil
}

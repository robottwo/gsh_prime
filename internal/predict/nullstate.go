package predict

import (
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// HistoryNullStatePredictor suggests the most recent command from history when
// no input has been provided, avoiding any LLM calls.
type HistoryNullStatePredictor struct {
	runner         *interp.Runner
	historyManager *history.HistoryManager
	logger         *zap.Logger
}

func NewHistoryNullStatePredictor(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	logger *zap.Logger,
) *HistoryNullStatePredictor {
	return &HistoryNullStatePredictor{
		runner:         runner,
		historyManager: historyManager,
		logger:         logger,
	}
}

func (p *HistoryNullStatePredictor) UpdateContext(_ *map[string]string) {}

func (p *HistoryNullStatePredictor) Predict(input string) (string, string, error) {
	if strings.TrimSpace(input) != "" {
		// this predictor is only for null state
		return "", "", nil
	}

	pwd := environment.GetPwd(p.runner)
	entries, err := p.historyManager.GetRecentEntries(pwd, 1)
	if err != nil {
		p.logger.Warn("error fetching history for null-state prediction", zap.Error(err))
		return "", "", nil
	}

	if len(entries) == 0 {
		return "", "", nil
	}

	prediction := entries[0].Command

	p.logger.Debug(
		"history-based null state prediction",
		zap.String("prediction", prediction),
	)

	return prediction, pwd, nil
}

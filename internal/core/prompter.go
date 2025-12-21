package core

import (
	"github.com/robottwo/bishop/pkg/gline"
	"go.uber.org/zap"
)

type UserPrompter interface {
	Prompt(
		prompt string,
		historyValues []string,
		explanation string,
		predictor gline.Predictor,
		explainer gline.Explainer,
		analytics gline.PredictionAnalytics,
		logger *zap.Logger,
		options gline.Options,
	) (string, error)
}

type DefaultUserPrompter struct{}

func (p DefaultUserPrompter) Prompt(
	prompt string,
	historyValues []string,
	explanation string,
	predictor gline.Predictor,
	explainer gline.Explainer,
	analytics gline.PredictionAnalytics,
	logger *zap.Logger,
	options gline.Options,
) (string, error) {
	return gline.Gline(prompt, historyValues, explanation, predictor, explainer, analytics, logger, options)
}

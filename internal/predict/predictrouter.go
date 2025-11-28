package predict

type PredictRouter struct {
	PrefixPredictor    *HistoryPrefixPredictor
	NullStatePredictor *HistoryNullStatePredictor
}

func (p *PredictRouter) UpdateContext(context *map[string]string) {
	if p.PrefixPredictor != nil {
		p.PrefixPredictor.UpdateContext(context)
	}

	if p.NullStatePredictor != nil {
		p.NullStatePredictor.UpdateContext(context)
	}
}

func (p *PredictRouter) Predict(input string) (string, string, error) {
	if input == "" {
		return p.NullStatePredictor.Predict(input)
	}
	return p.PrefixPredictor.Predict(input)
}

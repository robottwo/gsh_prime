package retrievers

import (
	"context"
	"fmt"
	"strings"

	"github.com/robottwo/bishop/internal/bash"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type GitStatusContextRetriever struct {
	Runner *interp.Runner
	Logger *zap.Logger
}

func (r GitStatusContextRetriever) Name() string {
	return "git_status"
}

func (r GitStatusContextRetriever) GetContext() (string, error) {
	revParseOut, _, err := bash.RunBashCommandInSubShell(context.Background(), r.Runner, "git rev-parse --show-toplevel")
	if err != nil {
		r.Logger.Debug("error running `git rev-parse --show-toplevel`", zap.Error(err))
		return "<git_status>not in a git repository</git_status>", nil
	}
	statusOut, _, err := bash.RunBashCommandInSubShell(context.Background(), r.Runner, "git status")
	if err != nil {
		r.Logger.Debug("error running `git status`", zap.Error(err))
		return "", nil
	}

	return fmt.Sprintf("<git_status>Project root: %s\n%s</git_status>", strings.TrimSpace(revParseOut), statusOut), nil
}

package retrievers

import (
	"fmt"

	"github.com/robottwo/bishop/internal/environment"
	"mvdan.cc/sh/v3/interp"
)

type WorkingDirectoryContextRetriever struct {
	Runner *interp.Runner
}

func (r WorkingDirectoryContextRetriever) Name() string {
	return "working_directory"
}

func (r WorkingDirectoryContextRetriever) GetContext() (string, error) {
	return fmt.Sprintf("<working_dir>%s</working_dir>", environment.GetPwd(r.Runner)), nil
}

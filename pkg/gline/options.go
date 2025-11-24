package gline

import "github.com/atinylittleshell/gsh/pkg/shellinput"

type Options struct {
	MinHeight          int
	CompletionProvider shellinput.CompletionProvider
}

func NewOptions() Options {
	return Options{
		MinHeight: 8,
	}
}

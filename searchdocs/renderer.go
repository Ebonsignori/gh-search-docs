package searchdocs

import (
	"github.com/charmbracelet/glamour"
)

// NewRenderer returns a Glamour renderer with the provided theme and wrap width.
func NewRenderer(theme string, wrap int) *glamour.TermRenderer {
	opts := []glamour.TermRendererOption{
		glamour.WithStandardStyle(theme),
		glamour.WithWordWrap(wrap),
	}
	r, _ := glamour.NewTermRenderer(opts...)
	return r
}

// NewAutoRenderer returns a Glamour renderer that automatically detects the best theme
func NewAutoRenderer(wrap int) *glamour.TermRenderer {
	opts := []glamour.TermRendererOption{
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrap),
	}
	r, _ := glamour.NewTermRenderer(opts...)
	return r
}

// NewAutoRendererNoWrap returns a Glamour renderer that automatically detects the best theme without word wrapping
func NewAutoRendererNoWrap() *glamour.TermRenderer {
	opts := []glamour.TermRendererOption{
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	}
	r, _ := glamour.NewTermRenderer(opts...)
	return r
}

// NewRendererNoWrap returns a Glamour renderer with the provided theme without word wrapping
func NewRendererNoWrap(theme string) *glamour.TermRenderer {
	opts := []glamour.TermRendererOption{
		glamour.WithStandardStyle(theme),
		glamour.WithWordWrap(0),
	}
	r, _ := glamour.NewTermRenderer(opts...)
	return r
}

package gomon

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
)

type FileInfo struct {
	// Root of the calling file.
	Root RootType `json:"root"`
	// File path, relative to Root.
	File string `json:"file"`
	// Line number, starting from 1.
	Line int `json:"line"`
}

// Highlight HTML contains a source code segment.
type Highlight struct {
	// Prefix HTML contains the current line, and WrapSize lines preceding it.
	Prefix string `json:"prefix,omitempty"`
	// Suffix HTML contains WrapSize lines succeeding the current line.
	Suffix string `json:"suffix,omitempty"`
}

type HighlightOptions struct {
	// WrapSize is the number of lines preceding/succeeding the current line.
	// Negative number disables highlight.
	WrapSize int `json:"wrapSize"`
}

func HighlightTokens(allTokens []chroma.Token, line int, opts HighlightOptions, hl *Highlight) error {
	if hl == nil {
		return fmt.Errorf("cannot highlight nil %T", (*Highlight)(nil))
	}
	wrapSize := opts.WrapSize
	if wrapSize < 0 {
		return nil
	}
	var tokens []chroma.Token
	var buf bytes.Buffer
	base := 1
	if wrapDiff := line - wrapSize; wrapDiff > 0 {
		base = wrapDiff
	}
	makeFormatter := func(baseLine, mark int) chroma.Formatter {
		opts := []html.Option{
			html.WithLineNumbers(true),
			html.LineNumbersInTable(true),
			html.Standalone(false),
			html.BaseLineNumber(baseLine),
			//html.WrapLongLines(true),
			html.TabWidth(3),
		}
		if mark != 0 {
			opts = append(opts, html.HighlightLines([][2]int{{mark, mark}}))
		}
		return html.New(opts...)
	}
	skipLines := line - 1 - wrapSize
	if skipLines < 0 {
		skipLines = 0
	}
	stopAtLine := line - 1 + wrapSize
	formatter := makeFormatter(base, line)
	var gotPrefix bool
	var haveLines int
	setSuffix := func() error {
		if len(tokens) == 0 {
			return nil
		}
		buf.Reset()
		err := formatter.Format(&buf, Vulcan, chroma.Literator(tokens...))
		if err != nil {
			return err
		}
		hl.Suffix = buf.String()
		tokens = tokens[:0]
		return nil
	}
	for _, tok := range allTokens {
		lfCount := strings.Count(tok.Value, "\n")
		if haveLines+lfCount < skipLines {
			haveLines += lfCount
			continue
		}
		if haveLines+lfCount >= line && !gotPrefix {
			haveLines += lfCount
			tokens = append(tokens, tok)
			buf.Reset()
			err := formatter.Format(&buf, Vulcan, chroma.Literator(tokens...))
			if err != nil {
				return err
			}
			gotPrefix = true
			tokens = tokens[:0]
			hl.Prefix = buf.String()
			if wrapSize <= 0 {
				break
			}
			formatter = makeFormatter(line+1, 0)
			continue
		}
		if haveLines+lfCount > stopAtLine {
			tokens = append(tokens, tok)
			if err := setSuffix(); err != nil {
				return err
			}
			break
		}
		for i, overflow := 0, skipLines-haveLines; i < overflow; i++ {
			_, tok.Value, _ = strings.Cut(tok.Value, "\n")
		}
		tokens = append(tokens, tok)
		haveLines += lfCount
	}
	return setSuffix()
}

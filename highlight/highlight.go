// Package highlight contains functionality to highlight source code via styled HTML.
package highlight

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/gofu/gomon/profiler"
	"github.com/gofu/gomon/style"
)

type Highlighter interface {
	Highlight(profiler.FileLine, Options, *profiler.Highlight) error
}

type Options struct {
	// WrapSize is the number of lines preceding/succeeding the current line.
	// Negative number disables highlight.
	WrapSize int `json:"wrapSize"`
}

func Tokens(allTokens []chroma.Token, line int, opts Options, hl *profiler.Highlight) error {
	if hl == nil {
		return fmt.Errorf("cannot highlight nil %T", hl)
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
		err := formatter.Format(&buf, style.Vulcan, chroma.Literator(tokens...))
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
			err := formatter.Format(&buf, style.Vulcan, chroma.Literator(tokens...))
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

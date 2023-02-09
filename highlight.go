package gomon

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
)

var ErrNilHighlight = fmt.Errorf("cannot highlight nil %T", (*Highlight)(nil))

// Highlight HTML contains a source code segment.
type Highlight struct {
	// Line number, starting from 1.
	Line int `json:"line"`
	// WrapSize is the number of lines preceding/succeeding the current line.
	// If negative, then both Prefix and Suffix are empty.
	WrapSize int `json:"wrapSize"`
	// Prefix HTML contains the current line, and WrapSize lines preceding it.
	Prefix string `json:"prefix,omitempty"`
	// Suffix HTML contains WrapSize lines succeeding the current line.
	Suffix string `json:"suffix,omitempty"`
}

func HighlightTokens(hl *Highlight, allTokens []chroma.Token) error {
	if hl == nil {
		return ErrNilHighlight
	}
	if hl.WrapSize < 0 {
		return nil
	}
	var tokens []chroma.Token
	var buf bytes.Buffer
	base := 1
	if wrapDiff := hl.Line - hl.WrapSize; wrapDiff > 0 {
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
	skipLines := hl.Line - 1 - hl.WrapSize
	if skipLines < 0 {
		skipLines = 0
	}
	stopAtLine := hl.Line - 1 + hl.WrapSize
	formatter := makeFormatter(base, hl.Line)
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
		if haveLines+lfCount >= hl.Line && !gotPrefix {
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
			if hl.WrapSize <= 0 {
				break
			}
			formatter = makeFormatter(hl.Line+1, 0)
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

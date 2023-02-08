package gomon

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers/g"
	"golang.org/x/sync/singleflight"
)

type Highlighter struct {
	mu    sync.RWMutex
	sf    singleflight.Group
	cache map[string][]chroma.Token
}

type Highlight struct {
	Prefix string
	Suffix string
}

func (h *Highlighter) Highlight(file string, line, wrapSize int) (Highlight, error) {
	var hh Highlight
	if wrapSize < 0 {
		return hh, nil
	}
	allTokens, err := h.getTokens(file)
	if err != nil {
		return hh, err
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
		err = formatter.Format(&buf, Vulcan, chroma.Literator(tokens...))
		if err != nil {
			return err
		}
		hh.Suffix = buf.String()
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
			tokens = appendToken(tokens, tok, 0)
			buf.Reset()
			err = formatter.Format(&buf, Vulcan, chroma.Literator(tokens...))
			if err != nil {
				return hh, err
			}
			gotPrefix = true
			tokens = tokens[:0]
			hh.Prefix = buf.String()
			if wrapSize <= 0 {
				break
			}
			formatter = makeFormatter(line+1, 0)
			continue
		}
		if haveLines+lfCount > stopAtLine {
			tokens = appendToken(tokens, tok, 0)
			if err = setSuffix(); err != nil {
				return hh, err
			}
			break
		}
		tokens = appendToken(tokens, tok, skipLines-haveLines)
		haveLines += lfCount
	}
	return hh, setSuffix()
}

func appendToken(tokens []chroma.Token, tok chroma.Token, overflow int) []chroma.Token {
	for i := 0; i < overflow; i++ {
		_, tok.Value, _ = strings.Cut(tok.Value, "\n")
	}
	return append(tokens, tok)
}

func (h *Highlighter) getTokens(file string) ([]chroma.Token, error) {
	h.mu.RLock()
	cached, ok := h.cache[file]
	h.mu.RUnlock()
	if ok {
		return cached, nil
	}
	v, err, _ := h.sf.Do(file, func() (interface{}, error) {
		h.mu.RLock()
		cached, ok := h.cache[file]
		h.mu.RUnlock()
		if ok {
			return cached, nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		iter, err := g.Go.Tokenise(nil, string(data))
		if err != nil {
			return nil, err
		}
		tokens := iter.Tokens()
		h.mu.Lock()
		if h.cache == nil {
			h.cache = map[string][]chroma.Token{}
		}
		h.cache[file] = tokens
		h.mu.Unlock()
		return tokens, nil
	})
	if err != nil {
		return nil, err
	}
	if cached, ok = v.([]chroma.Token); !ok {
		return nil, fmt.Errorf("expected cache type %T, got %T", cached, v)
	}
	return cached, nil
}

package gomon

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers/g"
	"golang.org/x/sync/singleflight"
)

var ErrNilHighlight = fmt.Errorf("cannot highlight nil %T", (*Highlight)(nil))

// Highlight HTML contains a source code segment.
type Highlight struct {
	// HighlightLine number, starting from 1.
	HighlightLine int `json:"highlightLine"`
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
	if wrapDiff := hl.HighlightLine - hl.WrapSize; wrapDiff > 0 {
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
	skipLines := hl.HighlightLine - 1 - hl.WrapSize
	if skipLines < 0 {
		skipLines = 0
	}
	stopAtLine := hl.HighlightLine - 1 + hl.WrapSize
	formatter := makeFormatter(base, hl.HighlightLine)
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
		if haveLines+lfCount >= hl.HighlightLine && !gotPrefix {
			haveLines += lfCount
			tokens = appendToken(tokens, tok, 0)
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
			formatter = makeFormatter(hl.HighlightLine+1, 0)
			continue
		}
		if haveLines+lfCount > stopAtLine {
			tokens = appendToken(tokens, tok, 0)
			if err := setSuffix(); err != nil {
				return err
			}
			break
		}
		tokens = appendToken(tokens, tok, skipLines-haveLines)
		haveLines += lfCount
	}
	return setSuffix()
}

func appendToken(tokens []chroma.Token, tok chroma.Token, overflow int) []chroma.Token {
	for i := 0; i < overflow; i++ {
		_, tok.Value, _ = strings.Cut(tok.Value, "\n")
	}
	return append(tokens, tok)
}

// Highlighter uses single-flight to lock filesystem reading/highlighting
// from multiple goroutines, and caches highlighted file source.
type Highlighter struct {
	FS    fs.FS
	mu    sync.RWMutex
	sf    singleflight.Group
	cache map[string][]chroma.Token
}

// Highlight source file/line with HTML. If wrapSize<0, no HTML is returned.
// If wrapSize==0, then only the current line is highlighted, meaning the
// suffix is empty. If wrapSize>0, then prefix contains 1+wrapSize lines,
// while suffix contains wrapSize lines.
func (h *Highlighter) Highlight(hl *Highlight, file string) error {
	if hl == nil {
		return ErrNilHighlight
	}
	if hl.WrapSize < 0 {
		return nil
	}
	allTokens, err := h.getTokens(file)
	if err != nil {
		return err
	}
	return HighlightTokens(hl, allTokens)
}

// getTokens returns parsed tokens from source code file,
// relying on cache and single-flight.
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
		readFS := h.FS
		h.mu.RUnlock()
		if ok {
			return cached, nil
		}
		data, err := readFile(readFS, file)
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

// readFile reads file content from readFS, or local filesystem if readFS==nil.
func readFile(readFS fs.FS, file string) ([]byte, error) {
	if readFS == nil {
		return os.ReadFile(file)
	}
	f, err := readFS.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

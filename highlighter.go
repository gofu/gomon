package gomon

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers/g"
	"golang.org/x/sync/singleflight"
)

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

// Package highlightfs provides cached file loading and syntax highlighting.
package highlightfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers/g"
	"github.com/gofu/gomon/env"
	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/profiler"
	"golang.org/x/sync/singleflight"
)

// FS uses single-flight to lock filesystem reading/highlighting
// from multiple goroutines, and caches highlighted file source.
type FS struct {
	FS    fs.FS
	Env   env.Env
	mu    sync.RWMutex
	sf    singleflight.Group
	cache map[string][]chroma.Token
}

// Highlight source file/line with HTML. If wrapSize<0, no HTML is returned.
// If wrapSize==0, then only the current line is highlighted, meaning the
// suffix is empty. If wrapSize>0, then prefix contains 1+wrapSize lines,
// while suffix contains wrapSize lines.
func (h *FS) Highlight(file profiler.FileLine, opts highlight.Options, hl *profiler.Highlight) error {
	if opts.WrapSize < 0 {
		return nil
	}
	allTokens, err := h.getTokens(path.Join(h.Env.RootPath(file.Root), file.File))
	if err != nil {
		return err
	}
	return highlight.Tokens(allTokens, file.Line, opts, hl)
}

// getTokens returns parsed tokens from source code file,
// relying on cache and single-flight.
func (h *FS) getTokens(file string) ([]chroma.Token, error) {
	h.mu.RLock()
	cached, ok := h.cache[file]
	h.mu.RUnlock()
	if ok {
		return cached, nil
	}
	v, err, _ := h.sf.Do(file, func() (any, error) {
		h.mu.RLock()
		cached, ok := h.cache[file]
		readFS := h.FS
		h.mu.RUnlock()
		if ok {
			return cached, nil
		}
		data, err := readFSFile(readFS, file)
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
	cached, _ = v.([]chroma.Token)
	return cached, err
}

// readFSFile reads file content from readFS, or local filesystem if readFS==nil.
func readFSFile(readFS fs.FS, file string) ([]byte, error) {
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

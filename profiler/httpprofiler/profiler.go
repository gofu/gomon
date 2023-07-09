// Package httpprofiler calls remote /debug/pprof pages to provide profiler data.
package httpprofiler

import (
	"fmt"
	"golang.org/x/exp/slices"
	"net/http"
	"strings"

	"github.com/gofu/gomon/env"
	"github.com/gofu/gomon/profiler"
	"github.com/gofu/gomon/profiler/httpparser"
)

// Profiler parses running goroutines from remote /debug/pprof/ pages.
type Profiler struct {
	url    string
	parser httpparser.Goroutine
}

// New expects pprofURL to be a default /debug/pprof/ page.
// The env defines file path prefixes for the parser, to group them
// by their defining package group (source, GOROOT, GOPATH).
func New(pprofURL string, env env.Env) *Profiler {
	pprofURL = strings.TrimRight(pprofURL, "/")
	if len(pprofURL) != 0 && !strings.Contains(pprofURL, "://") {
		pprofURL = "http://" + pprofURL
	}
	return &Profiler{
		url:    pprofURL,
		parser: httpparser.Goroutine{Env: env.Normalized()},
	}
}

// Source returns the full remote /debug/pprof URL.
func (s *Profiler) Source() string { return s.url }

// Goroutines parses running goroutines from remote URL.
func (s *Profiler) Goroutines() ([]profiler.Goroutine, error) {
	uri := s.url + "/goroutine?debug=2"
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	running, err := s.parser.Parse(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", s.url+uri, err)
	}
	slices.SortStableFunc(running, func(i, j profiler.Goroutine) bool {
		if iFirst, jFirst := i.ID == 1, j.ID == 1; iFirst != jFirst {
			return iFirst
		}
		return i.Duration > j.Duration
	})
	return running, nil
}

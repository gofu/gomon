// Package httpprofiler calls remote /debug/pprof pages to provide profiler data.
package httpprofiler

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/gofu/gomon/config"
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
func New(pprofURL string, env config.Env) *Profiler {
	pprofURL = strings.TrimRight(pprofURL, "/")
	if len(pprofURL) != 0 && !strings.Contains(pprofURL, "://") {
		pprofURL = "http://" + pprofURL
	}
	return &Profiler{
		url:    pprofURL,
		parser: httpparser.Goroutine{Env: env.Normalize()},
	}
}

// Source returns the full remote /debug/pprof URL.
func (s *Profiler) Source() string { return s.url }

// Goroutines parses running goroutines from remote URL.
func (s *Profiler) Goroutines() ([]profiler.Goroutine, error) {
	uri := s.url + "/goroutine?debug=2"
	res, err := request(uri)
	if err != nil {
		return nil, err
	}
	running, err := s.parser.Parse(res)
	_ = res.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", s.url+uri, err)
	}
	sort.Slice(running, func(i, j int) bool {
		if len(running[i].CallStack) > 0 && !running[i].CallStack[0].Caller {
			return true
		}
		return running[i].Duration > running[j].Duration
	})
	return running, nil
}

// request calls uri relative to /debug/pprof page, and returns non-closed HTTP body.
func request(uri string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

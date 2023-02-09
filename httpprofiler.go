package gomon

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// HTTPProfiler parses running goroutines from remote /debug/pprof/ pages.
type HTTPProfiler struct {
	pprofURL string
	parser   PProfParser
}

// NewHTTPProfiler expects pprofURL to be a default /debug/pprof/ page.
// The env defines file path prefixes for the parser, to group them
// by their defining package group (source, GOROOT, GOPATH).
func NewHTTPProfiler(pprofURL string, env EnvConfig) *HTTPProfiler {
	pprofURL = strings.TrimRight(pprofURL, "/")
	if len(pprofURL) != 0 && !strings.Contains(pprofURL, "://") {
		pprofURL = "http://" + pprofURL
	}
	return &HTTPProfiler{
		pprofURL: pprofURL,
		parser:   PProfParser{EnvConfig: env},
	}
}

// Goroutines parses running goroutines from remote URL.
func (s *HTTPProfiler) Goroutines() ([]Goroutine, error) {
	uri := "/goroutine?debug=2"
	res, err := s.request(uri)
	if err != nil {
		return nil, err
	}
	running, err := s.parser.Parse(res)
	_ = res.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", s.pprofURL+uri, err)
	}
	sort.Slice(running, func(i, j int) bool {
		if len(running[i].Stack) > 0 && !running[i].Stack[0].Caller {
			return true
		}
		return running[i].Duration > running[j].Duration
	})
	return running, nil
}

// request calls uri relative to /debug/pprof page, and returns non-closed HTTP body.
func (s *HTTPProfiler) request(uri string) (io.ReadCloser, error) {
	uri = s.pprofURL + uri
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

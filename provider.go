package gomon

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type HTTPProvider struct {
	url    string
	parser Parser
}

func NewHTTPProvider(pprofURL string, env EnvConfig) *HTTPProvider {
	pprofURL = strings.TrimRight(pprofURL, "/")
	if len(pprofURL) != 0 && !strings.Contains(pprofURL, "://") {
		pprofURL = "http://" + pprofURL
	}
	return &HTTPProvider{
		url:    pprofURL,
		parser: Parser{EnvConfig: env},
	}
}

func (s *HTTPProvider) GetRunning() ([]Goroutine, error) {
	uri := "/goroutine?debug=2"
	res, err := s.request(uri)
	if err != nil {
		return nil, err
	}
	running, err := s.parser.parseGoroutines(res)
	_ = res.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", s.url+uri, err)
	}
	sort.Slice(running, func(i, j int) bool {
		if len(running[i].Stack) > 0 && !running[i].Stack[0].Caller {
			return true
		}
		return running[i].Duration > running[j].Duration
	})
	return running, nil
}

func (s *HTTPProvider) request(uri string) (io.ReadCloser, error) {
	uri = s.url + uri
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

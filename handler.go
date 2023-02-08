package gomon

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type Handler struct {
	env         EnvConfig
	provider    GoroutineProvider
	highlighter Highlighter
}

func NewHandler(env EnvConfig, provider GoroutineProvider) *Handler {
	return &Handler{env: env, provider: provider}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.serveIndex(w, r)
	case "/json":
		h.serveJSON(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	running, err := h.provider.GetRunning()
	if err != nil {
		h.serveError(w, r, err)
		return
	}
	data := IndexData{
		Durations: fibonacciSlice(20, time.Minute),
		Markups:   fibonacciSlice(10, 1),
		Contexts:  fibonacciSlice(10, 1),
		Total:     len(running),
		Running:   running,
	}
	data.RequestData, err = ParseRequestData(r.URL.Query())
	if err != nil {
		h.serveError(w, r, err)
		return
	}
	if data.Min > 0 {
		var filtered []Goroutine
		for _, gr := range running {
			if gr.Duration < data.Min {
				data.Skipped++
				continue
			}
			filtered = append(filtered, gr)
		}
		data.Running = filtered
	}
	if data.Lines >= 0 {
		for i, goroutine := range data.Running {
			if data.MarkupLimit != 0 && i > data.MarkupLimit {
				break
			}
			for j, s := range goroutine.Stack {
				hh, err := h.highlightStack(s, data.Lines)
				if err != nil {
					h.serveError(w, r, err)
					return
				}
				goroutine.Stack[j].Prefix = template.HTML(hh.Prefix)
				goroutine.Stack[j].Suffix = template.HTML(hh.Suffix)
			}
		}
	}
	var buf bytes.Buffer
	if err = indexTpl.Execute(&buf, data); err != nil {
		h.serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (h *Handler) serveJSON(w http.ResponseWriter, r *http.Request) {
	running, err := h.provider.GetRunning()
	if err != nil {
		h.serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(running)
}

func (h *Handler) highlightStack(s StackElem, wrap int) (Highlight, error) {
	if wrap < 0 {
		return Highlight{}, nil
	}
	var root string
	switch s.Root {
	case "PROJECT":
		root = h.env.Root
	case "GOROOT":
		root = h.env.GoRoot
	case "GOPATH":
		root = h.env.GoPath
	default:
		return Highlight{}, nil
	}
	hh, err := h.highlighter.Highlight(path.Join(root, s.File), s.Line, wrap)
	if err != nil {
		return hh, err
	}
	return hh, nil
}

func (h *Handler) serveError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}

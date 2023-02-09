package gomon

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	provider    Profiler
	highlighter EnvHighlighter
}

func NewHandler(env EnvConfig, profiler Profiler) *Handler {
	h := &Handler{provider: profiler}
	h.highlighter.Env = env
	return h
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
	running, err := h.provider.Goroutines()
	if err != nil {
		serveError(w, r, err)
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
		serveError(w, r, err)
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
		if err = markupGoroutines(r.Context(), data.Running, &h.highlighter, data.MarkupLimit, data.Lines); err != nil {
			serveError(w, r, err)
			return
		}
	}
	var buf bytes.Buffer
	if err = indexTpl.Execute(&buf, data); err != nil {
		serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (h *Handler) serveJSON(w http.ResponseWriter, r *http.Request) {
	running, err := h.provider.Goroutines()
	if err != nil {
		serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(running)
}

func serveError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}

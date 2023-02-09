package gomon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Handler struct {
	provider    Profiler
	highlighter envHighlighter
}

func NewHandler(env EnvConfig, profiler Profiler) *Handler {
	h := &Handler{provider: profiler}
	h.highlighter.env = env
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
	data, err := h.execIndex(r.Context(), r.URL.Query())
	if err != nil {
		serveError(w, r, err)
		return
	}
	serveTemplate(w, r, indexTpl, data)
}

func (h *Handler) execIndex(ctx context.Context, query url.Values) (IndexData, error) {
	data := IndexData{
		Durations: fibSlice(20, time.Minute),
		Markups:   fibSlice(10, 1),
		Contexts:  fibSlice(10, 1),
	}
	running, err := h.goroutines()
	if err != nil {
		return data, err
	}
	data.Running, data.Skipped = filterGoroutines(running, data.Min)
	data.Total = len(running)
	data.RequestData, err = ParseRequestData(query)
	if err != nil {
		return data, err
	}
	if data.Lines >= 0 {
		err = markupGoroutines(ctx, data.Running, &h.highlighter, data.MarkupLimit, data.Lines)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

func (h *Handler) serveJSON(w http.ResponseWriter, r *http.Request) {
	running, err := h.goroutines()
	if err != nil {
		serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(running)
}

func (h *Handler) goroutines() ([]Goroutine, error) {
	if h.provider == nil {
		return nil, fmt.Errorf("%T has nil %T", h, h.provider)
	}
	return h.provider.Goroutines()
}

func serveTemplate(w http.ResponseWriter, r *http.Request, tpl *template.Template, data any) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func serveError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}

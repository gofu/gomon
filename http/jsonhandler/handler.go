// Package jsonhandler serves running goroutines in HTML format.
package jsonhandler

import (
	"net/http"

	"github.com/gofu/gomon/http/serve"
	"github.com/gofu/gomon/profiler"
)

// Handler serves running goroutines as JSON.
type Handler struct {
	prof profiler.Profiler
}

// New requires non-nil profiler.
func New(prof profiler.Profiler) Handler {
	return Handler{prof: prof}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	running, err := h.prof.Goroutines()
	if err != nil {
		serve.Error(w, r, err)
		return
	}
	profiler.Sort(running)
	serve.JSON(w, r, running)
}

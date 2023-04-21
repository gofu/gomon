// Package graphhandler serves running goroutines in graph format.
package graphhandler

import (
	"net/http"

	"github.com/gofu/gomon/http/serve"
	"github.com/gofu/gomon/profiler"
)

// Handler serves running goroutines in graph format.
type Handler struct {
	prof profiler.Profiler
}

// New returns Handler that shows data from profiler.
func New(prof profiler.Profiler) Handler {
	return Handler{prof: prof}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data Data
	gs, err := h.prof.Goroutines()
	if err != nil {
		serve.Error(w, r, err)
		return
	}
	data.Total = len(gs)
	data.Group = GroupGoroutines(gs)
	data.Unique = len(data.Group)
	serve.JSON(w, r, data)
}

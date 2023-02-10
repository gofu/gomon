// Package htmlhandler serves running goroutines in HTML format.
package htmlhandler

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/http/serve"
	"github.com/gofu/gomon/profiler"
	"golang.org/x/exp/constraints"
)

// Handler serves running goroutines as HTML. The source code of the
// call stack is also optionally showed as styled/colored HTML.
type Handler struct {
	prof profiler.Profiler
	hl   highlight.Highlighter
}

// New requires non-nil highlighter and profiler.
func New(highlighter highlight.Highlighter, prof profiler.Profiler) *Handler {
	return &Handler{
		prof: prof,
		hl:   highlighter,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h.Execute(r.Context(), r.URL.Query())
	if err != nil {
		serve.Error(w, r, err)
		return
	}
	serve.HTMLTemplate(w, r, tpl, data)
}

var (
	indexDurations = fibSlice(20, time.Minute)
	indexMarkups   = fibSlice(10, 1)
	indexContexts  = indexMarkups
)

func (h *Handler) Execute(ctx context.Context, query url.Values) (Data, error) {
	data := Data{
		Durations: indexDurations,
		Markups:   indexMarkups,
		Contexts:  indexContexts,
	}
	running, err := h.prof.Goroutines()
	if err != nil {
		return data, err
	}
	data.Total = len(running)
	data.Request, err = ParseRequest(query)
	if err != nil {
		return data, err
	}
	data.Running, data.Skipped = data.Filter.Filter(running)
	if data.WrapSize >= 0 {
		err = MarkupGoroutines(ctx, data.Running, h.hl, data.MarkupOptions)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

// fibSlice returns a slice of count elements, where
// each element is multiplied by its fibonacci order.
//
// Example:
//
//	fibSlice[int](5, 1) // []int{1,2,3,5,8}
//	fibSlice[int](5, 2) // []int{2,4,6,10,16}
func fibSlice[T constraints.Integer | constraints.Float](count int, multiplier T) []T {
	elems := make([]T, count)
	for i := 0; i < count; i++ {
		switch i {
		case 0:
			elems[i] = multiplier
		case 1:
			elems[i] = 2 * multiplier
		default:
			elems[i] = elems[i-2] + elems[i-1]
		}
	}
	return elems
}

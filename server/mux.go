package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gofu/gomon/handlers/htmlhandler"
	"github.com/gofu/gomon/handlers/indexhandler"
	"github.com/gofu/gomon/handlers/jsonhandler"
	"github.com/gofu/gomon/handlers/router"
	"github.com/gofu/gomon/handlers/statichandler"
	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/profiler"
)

// NewServeMux returns an http.Handler that handles the following pages:
//   - GET /debug/pprof - net/http/pprof handler, plaintext
//   - GET /json - list all goroutines, JSON
//   - GET /html?min&max&markup&lines - list all goroutines, HTML
//   - GET / - list all routes
func NewServeMux(hl highlight.Highlighter, prof profiler.Profiler) *http.ServeMux {
	routes := router.Default
	mux := http.NewServeMux()
	mux.HandleFunc(routes.PProf, pprof.Index)
	mux.HandleFunc(routes.PProf+"cmdline", pprof.Cmdline)
	mux.HandleFunc(routes.PProf+"profile", pprof.Profile)
	mux.HandleFunc(routes.PProf+"symbol", pprof.Symbol)
	mux.HandleFunc(routes.PProf+"trace", pprof.Trace)
	mux.Handle(statichandler.FaviconURL, statichandler.Handler{})
	mux.Handle(routes.JSON, jsonhandler.New(prof))
	mux.Handle(routes.HTML, htmlhandler.New(hl, prof))
	index := indexhandler.Data{
		PProfURL: prof.Source(),
		Links: []indexhandler.Link{
			{"index", routes.Index, "this page"},
			{"HTML", routes.HTML, "running goroutines in HTML format"},
			{"JSON", routes.JSON, "running goroutines in JSON format"},
			{"pprof", routes.PProf, "debug profiler"},
		},
	}
	mux.Handle(routes.Index, indexhandler.New(index))
	return mux
}

package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gofu/gomon/handlers"
	"github.com/gofu/gomon/handlers/htmlhandler"
	"github.com/gofu/gomon/handlers/indexhandler"
	"github.com/gofu/gomon/handlers/jsonhandler"
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
	router := handlers.Router{
		Index: "/",
		HTML:  "/html",
		JSON:  "/json",
		PProf: "/debug/pprof/",
	}
	mux := http.NewServeMux()
	mux.HandleFunc(router.PProf, pprof.Index)
	mux.HandleFunc(router.PProf+"cmdline", pprof.Cmdline)
	mux.HandleFunc(router.PProf+"profile", pprof.Profile)
	mux.HandleFunc(router.PProf+"symbol", pprof.Symbol)
	mux.HandleFunc(router.PProf+"trace", pprof.Trace)
	mux.Handle(statichandler.FaviconURL, statichandler.Handler{})
	mux.Handle(router.JSON, jsonhandler.New(prof))
	mux.Handle(router.HTML, htmlhandler.New(hl, prof))
	index := indexhandler.Data{
		PProfURL: prof.Source(),
		Links: []indexhandler.Link{
			{"index", router.Index, "this page"},
			{"HTML", router.HTML, "running goroutines in HTML format"},
			{"JSON", router.JSON, "running goroutines in JSON format"},
			{"pprof", router.PProf, "debug profiler"},
		},
	}
	mux.Handle(router.Index, indexhandler.New(index))
	return mux
}

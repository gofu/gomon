// Package server provides an HTTP handler and listener, that shows profiler info.
package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gofu/gomon/config"
	"github.com/gofu/gomon/handlers"
	"github.com/gofu/gomon/handlers/htmlhandler"
	"github.com/gofu/gomon/handlers/indexhandler"
	"github.com/gofu/gomon/handlers/jsonhandler"
	"github.com/gofu/gomon/handlers/statichandler"
	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/highlight/highlightfs"
	"github.com/gofu/gomon/profiler"
	"github.com/gofu/gomon/profiler/httpprofiler"
	"golang.org/x/sync/errgroup"
)

// StartServer starts an HTTP server on configured address, showing running
// goroutines and their call stack context, fetched from .go source files.
// Canceling ctx stops the server, and returns ctx.Err().
func StartServer(ctx context.Context, conf config.Server) error {
	ln, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on http://%s", ln.Addr())
	group, ctx := errgroup.WithContext(ctx)
	prof := httpprofiler.New(conf.PProfURL, conf.Remote.WithDefaults(conf.Local))
	hl := &highlightfs.FS{Env: conf.Local}
	srv := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           NewServeMux(hl, prof),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	group.Go(func() error {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})
	group.Go(func() error {
		<-ctx.Done()
		_ = srv.Close()
		return ctx.Err()
	})
	return group.Wait()
}

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
	mux.Handle("/favicon.ico", statichandler.Handler{})
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

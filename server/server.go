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
	"github.com/gofu/gomon/handlers/htmlhandler"
	"github.com/gofu/gomon/handlers/jsonhandler"
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
	highlighter := &highlightfs.FS{Env: conf.Local}
	srv := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           NewServeMux(highlighter, prof),
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
//   - GET /?min&max&limit&markup - list all goroutines, HTML
func NewServeMux(hl highlight.Highlighter, prof profiler.Profiler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/json", jsonhandler.New(prof))
	mux.Handle("/", htmlhandler.New(hl, prof))
	return mux
}

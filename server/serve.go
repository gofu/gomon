// Package server provides an HTTP handler and listener, that shows profiler info.
package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gofu/gomon/config"
	"github.com/gofu/gomon/highlight/highlightfs"
	"github.com/gofu/gomon/profiler/httpprofiler"
	"golang.org/x/sync/errgroup"
)

// ListenAndServe starts an HTTP server on configured address, showing running
// goroutines and their call stack context, fetched from .go source files.
// Canceling ctx stops the server, and returns ctx.Err().
func ListenAndServe(ctx context.Context, conf config.Server) error {
	ln, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on http://%s", ln.Addr())
	group, ctx := errgroup.WithContext(ctx)
	prof := httpprofiler.New(conf.PProfURL, conf.Remote.WithDefaults(conf.Local).Normalize())
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

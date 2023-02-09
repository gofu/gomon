// Command gomon starts a local HTTP server showing running goroutines from a remote Go
// process's /debug/pprof HTTP page, using local source .go files to show stack trace.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"

	"github.com/gofu/gomon/config"
	"github.com/gofu/gomon/server"
)

func main() {
	var conf config.Server
	flag.StringVar(&conf.Addr, "addr", "127.0.0.1:7656", "HTTP listen address")
	flag.StringVar(&conf.PProfURL, "url", "http://127.0.0.1:7656/debug/pprof", "Remote /debug/pprof URL")
	flag.StringVar(&conf.Local.Root, "local-root", runtime.GOROOT(), "Local project root")
	flag.StringVar(&conf.Local.GoRoot, "local-goroot", "", "Local GOROOT")
	flag.StringVar(&conf.Local.GoPath, "local-gopath", os.Getenv("GOPATH"), "Local GOPATH")
	flag.StringVar(&conf.Remote.Root, "remote-root", "", "Remote project root")
	flag.StringVar(&conf.Remote.GoRoot, "remote-goroot", "", "Remote GOROOT")
	flag.StringVar(&conf.Remote.GoPath, "remote-gopath", "", "Remote GOPATH")
	flag.Parse()
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := server.ListenAndServe(ctx, conf)
	if err != nil {
		log.Fatal(err)
	}
}

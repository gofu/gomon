package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"

	"github.com/gofu/gomon"
)

func main() {
	var conf gomon.Config
	flag.StringVar(&conf.Addr, "addr", "127.0.0.1:7656", "HTTP listen addr")
	flag.StringVar(&conf.URL, "url", "http://127.0.0.1:7656/debug/pprof/goroutine?debug=2", "Remote /debug/pprof/goroutine?debug=2 URL")
	flag.StringVar(&conf.LocalRoot, "local-root", "", "Local project root")
	flag.StringVar(&conf.LocalGoRoot, "local-goroot", "", "Local GOROOT")
	flag.StringVar(&conf.LocalGoPath, "local-gopath", "", "Local GOPATH")
	flag.StringVar(&conf.RemoteRoot, "remote-root", "", "Remote project root")
	flag.StringVar(&conf.RemoteGoRoot, "remote-goroot", "", "Remote GOROOT")
	flag.StringVar(&conf.RemoteGoPath, "remote-gopath", "", "Remote GOPATH")
	flag.Parse()
	runtime.GOROOT()
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := gomon.Run(ctx, conf)
	if err != nil {
		log.Fatal(err)
	}
}

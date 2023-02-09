// Package config contains configuration structs.
package config

// Server configuration, for running an HTTP server serving detailed profiler info.
type Server struct {
	// Addr is the HTTP address to listen on.
	Addr string
	// PProfURL is the remote /debug/pprof URL to query.
	PProfURL string
	// Local environment info, used to parse .go source files.
	Local Env
	// Remote environment info, used to map results of PProfURL to Local environment.
	Remote Env
}
